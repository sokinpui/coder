package main

import (
	"coder/internal/browser"
	"coder/internal/logger"
	"coder/internal/server"
	"coder/internal/utils"
	"coder/web"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: This application must be run from within a git repository.")
		os.Exit(1)
	}

	addr := flag.String("addr", ":0", "http service address. Defaults to a random unused port.")
	noBrowser := flag.Bool("no-browser", false, "Do not open the browser automatically.")
	flag.Parse()

	logger.Init()

	// WebSocket handler
	http.HandleFunc("/ws", server.HandleConnections)

	// Serve project files (e.g., for images in .coder/images)
	// This allows the frontend to request /files/.coder/images/some-image.png
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(repoRoot))))

	// Serve static files for the web UI
	distFS, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		log.Fatalf("failed to create sub filesystem for web assets: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(distFS)))

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to create listener: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	host := "localhost"

	reqHost, _, err := net.SplitHostPort(*addr)
	if err == nil && reqHost != "" && reqHost != "0.0.0.0" && reqHost != "::" {
		host = reqHost
	}

	serverURL := fmt.Sprintf("http://%s:%d", host, port)
	log.Printf("Starting web server on %s", serverURL)
	fmt.Printf("Coder-web is running on: %s\n", serverURL)

	if !*noBrowser {
		if err := browser.Open(serverURL); err != nil {
			log.Printf("failed to open browser: %v", err)
		}
	}

	if err := http.Serve(listener, nil); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
