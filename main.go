package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/google/uuid"
	"github.com/mdp/qrterminal/v3"
)

type Form struct {
	Token string
}

type uploadSuccessMsg struct {
	filename string
}

type uploadErrorMsg struct {
	err error
}

type NetworkInterface struct {
	Name string
	IP   string
}

type networkSelection struct {
	ifaces []NetworkInterface
	cursor int
}

type waitingScreen struct {
	qrCodeStr string
	uploadURL string
	choseIP   string
	spinner   spinner.Model
}

type endScreen struct {
	successMsg string
	err        error
}

type model struct {
	state int
	n     networkSelection
	w     waitingScreen
	e     endScreen
}

const (
	maxUploadSize = 10 * 1024 * 1024
	stateSelectIP = iota
	stateWait
	stateDone
)

var (
	uploadDirectory string
	curSessionToken string
	p               *tea.Program
)

//go:embed index.html
var htmlForm embed.FS

func (m model) View() tea.View {
	switch m.state {
	case stateSelectIP:
		s := "Select a Network:\n"
		for i, ip := range m.n.ifaces {
			cursor := " "
			if m.n.cursor == i {
				cursor = ">"
			}
			s += fmt.Sprintf("%s %s\n", cursor, ip)
		}
		s += "\n(use up/down or j/k to move, enter to select, q to quit)\n"
		return tea.NewView(s)
	case stateWait:
		return tea.NewView(fmt.Sprintf(
			"Scan to upload:\n\n%s\n\nURL: %s\nCaution: Only files under 10MB\n%s",
			m.w.qrCodeStr,
			m.w.uploadURL,
			m.w.spinner.View(),
		))
	case stateDone:
		if m.e.err != nil {
			return tea.NewView(fmt.Sprintf("Error: %s", m.e.err.Error()))
		}
		return tea.NewView(fmt.Sprintf("Success! Saved to: %s", m.e.successMsg))
	default:
		return tea.NewView("unknown state")
	}
}

func initialModel() model {
	localIps, err := getLocalIPAddr()
	if err != nil {
		log.Fatal(err)
	}
	s := spinner.New()
	s.Spinner = spinner.Dot
	myModel := model{
		state: stateSelectIP,
		n: networkSelection{
			ifaces: localIps,
			cursor: 0,
		},
		w: waitingScreen{
			spinner: s,
		},
	}
	return myModel
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case uploadSuccessMsg:
		m.state = stateDone
		m.e.successMsg = msg.filename
		return m, tea.Quit
	case uploadErrorMsg:
		m.state = stateDone
		m.e.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.w.spinner, cmd = m.w.spinner.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.n.cursor > 0 {
				m.n.cursor--
			}
		case "down", "j":
			if m.n.cursor < len(m.n.ifaces)-1 {
				m.n.cursor++
			}
		case "enter":
			if m.state == stateSelectIP {
				m.w.choseIP = m.n.ifaces[m.n.cursor].IP
				m.state = stateWait
				qrStr, url := generateQR(m.w.choseIP, "8080")
				m.w.qrCodeStr = qrStr
				m.w.uploadURL = url
				return m, m.w.spinner.Tick
			}
		}
	}
	return m, nil
}

func getLocalIPAddr() ([]NetworkInterface, error) {
	var interfaces []NetworkInterface
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
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			// Save both the interface name and the IP
			interfaces = append(interfaces, NetworkInterface{Name: i.Name, IP: ip.String()})
		}
	}
	return interfaces, nil
}

func generateQR(localIPAddr string, port string) (string, string) {
	// generate a session token
	sessionToken := uuid.New().String()
	curSessionToken = sessionToken
	// url where the file transfer will happen
	uploadURL := fmt.Sprintf("http://%s:%s/upload?token=%s", localIPAddr, port, sessionToken)
	var buf bytes.Buffer
	// config for the qr
	cfg := qrterminal.Config{
		Level:     qrterminal.L,     // level of error tolerance
		Writer:    &buf,             // destination where the qr should be displayed
		BlackChar: qrterminal.BLACK, // color of the dark part of qr
		WhiteChar: qrterminal.WHITE, // color of the white part of qr
		QuietZone: 2,                // empty space around the qr
	}
	// generate the qr
	qrterminal.GenerateWithConfig(uploadURL, cfg)
	return buf.String(), uploadURL
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Grab the token from query params to verify it later if needed
	token := r.URL.Query().Get("token")
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	if token != curSessionToken {
		http.Error(w, "unauthorised for file transfer", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	f := Form{
		Token: token,
	}
	templ, err := template.ParseFS(htmlForm, "index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Fatal(err)
		return
	}
	templ.Execute(w, f)
}

func generateUniqueFilepath(uploadDir, filename string) string {
	basePath := filepath.Join(uploadDir, filename)
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath
	}
	ext := filepath.Ext(filename)
	nameWithoutExt := filename[:len(filename)-len(ext)]

	counter := 1
	for {
		newFilename := fmt.Sprintf("%s(%d)%s", nameWithoutExt, counter, ext)
		newPath := filepath.Join(uploadDir, newFilename)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := r.URL.Query().Get("token")
	if token != curSessionToken || curSessionToken == "" {
		http.Error(w, "unauthorised for file transfer", http.StatusUnauthorized)
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
	dstPath := generateUniqueFilepath(uploadDirectory, handler.Filename)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		http.Error(w, "Unable to save the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Error saving the file contents", http.StatusInternalServerError)
		p.Send(uploadErrorMsg{err: err})
		return
	}

	w.WriteHeader(http.StatusOK)
	savedName := filepath.Base(dstPath)
	fmt.Fprintf(w, "Successfully uploaded file as: %s", savedName)
	curSessionToken = ""
	p.Send(uploadSuccessMsg{filename: savedName})
}

func main() {
	// get the upload directory of the current user
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("not able to find the home directory")
		return
	}
	uploadDirectory = fmt.Sprintf("%s/Downloads", homeDir)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /upload", formHandler)
	mux.HandleFunc("POST /upload", uploadHandler)
	go func() {
		err = http.ListenAndServe(":8080", mux)
		if err != nil {
			log.Fatal(err)
		}
	}()
	p = tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
