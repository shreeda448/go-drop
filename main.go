package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/mdp/qrterminal/v3"
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

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the token from query params to verify it later if needed
	token := r.URL.Query().Get("token")
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// 2. Send a massive, centered HTML heading so it is impossible to miss
	hugeText := fmt.Sprintf(`
		<html>
		<head>
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body style="background-color: #1e1e2e; color: #ffffff; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; font-family: sans-serif;">
			<h1 style="text-align: center; font-size: 50px;">
				Hello Go-Drop!<br>
				<span style="font-size: 25px; color: #a6adc8;">Token: %s</span>
			</h1>
		</body>
		</html>
	`, token)

	fmt.Fprint(w, hugeText)
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
	mux.HandleFunc("/upload", uploadHandler)
	err = http.ListenAndServe(addr, mux)
	if err != nil {
		log.Printf("failed to find the local ip : %v", err)
		return
	}
}
