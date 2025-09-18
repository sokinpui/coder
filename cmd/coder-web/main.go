package main

import (
	"embed"
	"coder/internal/logger"
	"coder/internal/server"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

//go:embed all:../../web/dist
var webAssets embed.FS

func isGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func main() {
	if !isGitRepository() {
		fmt.Fprintln(os.Stderr, "Error: This application must be run from within a git repository.")
		os.Exit(1)
	}
	addr := flag.String("addr", ":8084", "http service address")
	flag.Parse()

	logger.Init()

	http.HandleFunc("/ws", server.HandleConnections)

	// Serve static files for the web UI
	distFS, err := fs.Sub(webAssets, "web/dist")
	if err != nil {
		log.Fatalf("failed to create sub filesystem for web assets: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(distFS)))

	log.Printf("Starting web server on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
