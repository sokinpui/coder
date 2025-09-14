package main

import (
	"coder/internal/logger"
	"coder/internal/server"
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8080", "http service address")
	flag.Parse()

	logger.Init()

	http.HandleFunc("/ws", server.HandleConnections)

	// Serve static files for the web UI
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/", fs)

	log.Printf("Starting web server on %s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
