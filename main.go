package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/google/uuid"
	"github.com/mdp/qrterminal/v3"
)

type Form struct {
	Token string
}

const (
	maxUploadSize   = 10 * 1024 * 1024
	uploadDirectory = "/home/shreeda/Downloads"
)

func getLocalIPAddr() ([]string, error) {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// Skip loopback and IPv6 addresses
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	return ips, nil
}

func generateQR(localIPAddr string, port string) string {
	// generate a session token
	sessionToken := uuid.New().String()
	// url where the file transfer will happen
	uploadURL := fmt.Sprintf("http://%s:%s/upload?token=%s", localIPAddr, port, sessionToken)
	fmt.Print("Scan this QR code with your phone to upload a file:\n\n")
	// config for the qr
	cfg := qrterminal.Config{
		Level:     qrterminal.L,     // level of error tolerance
		Writer:    os.Stdout,        // destination where the qr should be displayed
		BlackChar: qrterminal.BLACK, // color of the dark part of qr
		WhiteChar: qrterminal.WHITE, // color of the white part of qr
		QuietZone: 1,                // empty space around the qr
	}
	// generate the qr
	qrterminal.GenerateWithConfig(uploadURL, cfg)
	return uploadURL
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Grab the token from query params to verify it later if needed
	token := r.URL.Query().Get("token")
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.WriteHeader(http.StatusOK)
	f := Form{
		Token: token,
	}
	templ, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}
	templ.Execute(w, f)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		http.Error(w, "invalid multipart form or file size too big", http.StatusBadRequest)
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		log.Printf("error retrieving the file:%v", err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %s\n", handler.Filename)
	fmt.Printf("File Size: %d bytes\n", handler.Size)
	err = os.MkdirAll(uploadDirectory, os.ModePerm)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	dstPath := filepath.Join(uploadDirectory, handler.Filename)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		http.Error(w, "Unable to save the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error saving the file contents", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Successfully uploaded file: %s", handler.Filename)
}

func main() {
	// get the local ip address
	localIP, err := getLocalIPAddr()
	if err != nil || len(localIP) == 0 {
		log.Printf("failed to find the local ip : %v", err)
		return
	}
	fmt.Println("Found these IPs on your machine:")
	for i, ip := range localIP {
		fmt.Printf("[%d] %s\n", i, ip)
	}
	targetIP := localIP[0]
	fmt.Printf("\n--> Generating QR for IP: %s\n", targetIP)
	// assign a port for the server to run
	port := "8080"
	uploadURL := generateQR(targetIP, port)
	fmt.Printf("\nServer will listen on: %s\n", uploadURL)
	mux := http.NewServeMux()
	addr := fmt.Sprintf(":%s", port)
	mux.HandleFunc("GET /upload", formHandler)
	mux.HandleFunc("POST /upload", uploadHandler)
	err = http.ListenAndServe(addr, mux)
	if err != nil {
		log.Printf("failed to find the local ip : %v", err)
		return
	}
}
