package main

import (
	"coder/internal/logger"
	"coder/internal/server"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

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
	fs := http.FileServer(http.Dir("./web/dist"))
	http.Handle("/", fs)

	log.Printf("Starting web server on %s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
