# go-drop 🚀

A fast, terminal-based local file transfer tool written in Go. 

**go-drop** allows you to instantly send files from your phone (or any device on the same local network) directly to your computer. Simply run the app, select your network interface, scan the generated QR code, and upload. 

Built with the beautiful [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

---

## ✨ Features

* **Interactive TUI:** Easily select the correct network interface using a slick terminal UI.
* **Instant QR Generation:** No need to type IP addresses or URLs. Just scan and upload.
* **100% Local:** Files are transferred directly over your local Wi-Fi network. No cloud, no internet required.
* **Secure Sessions:** Generates a unique UUID token for every session so only you can upload files.
* **Auto-Save:** Files are automatically saved to your user's `Downloads` directory with smart renaming to prevent overwriting existing files.

---

## 📥 Installation

You do not need to have Go installed to use this tool! Pre-compiled binaries are available for Windows, macOS, and Linux.

1. Go to the [Releases page](../../releases) *(Update this link to your actual GitHub URL)*.
2. Download the binary that matches your operating system and architecture.
3. Extract the executable and run it from your terminal.

---

## 🚀 Usage

Run the executable from your terminal:

```bash
./go-drop
```
1. **Select Network:** Use the `Up`/`Down` arrow keys (or `k`/`j`) to choose the network interface connected to your Wi-Fi or Mobile Hotspot. Press `Enter`.
2. **Scan:** Open your phone's camera and scan the QR code displayed in the terminal.
3. **Upload:** A simple webpage will open on your phone. Select a file (up to 10MB) and hit upload.
4. **Done:** The file will instantly appear in your `~/Downloads` folder!

## 🛠️ Building from Source

If you prefer to compile the project yourself, you will need Go installed on your machine.

1. Clone the repository:

   ```bash
   git clone https://github.com/shreeda448/go-drop.git
   cd go-drop
   ```
2. Download dependencies:

```bash
go mod tidy
```
3. Build the binary:

```bash
go build -o go-drop main.go
```
## ⚠️ Troubleshooting

**The webpage isn't loading on my phone!**

* **Check your connection:** Ensure your phone and computer are on the exact same Wi-Fi network or mobile hotspot.
* **Check your firewall:** By default, `go-drop` runs on port `8080`. If you are on Linux (like Fedora) or Windows, your firewall might be blocking incoming connections.
  * Linux (Firewalld): Run `sudo firewall-cmd --add-port=8080/tcp`
  * Linux (UFW): Run `sudo ufw allow 8080/tcp`

**The QR code looks distorted!**
* Ensure you are using a modern terminal emulator that supports standard ANSI escape sequences and non-breaking spaces.
