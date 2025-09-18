package main

import (
	"coder/internal/logger"
	"coder/internal/server"
	"coder/internal/utils"
	"coder/web"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

func main() {
	if _, err := utils.FindRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: This application must be run from within a git repository.")
		os.Exit(1)
	}
	addr := flag.String("addr", ":8084", "http service address")
	flag.Parse()

	logger.Init()

	http.HandleFunc("/ws", server.HandleConnections)

	// Serve static files for the web UI
	distFS, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		log.Fatalf("failed to create sub filesystem for web assets: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(distFS)))

	log.Printf("Starting web server on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
