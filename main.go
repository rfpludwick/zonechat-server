package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	config        *Configuration
	serverStarted = false
)

func main() {
	// Process configuration
	config = processConfiguration()

	// Setup signals handler
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go handleSignals(signals)

	// Spin up server
	server := newServer()

	go server.run()

	// Setup and run HTTP server
	http.HandleFunc("/", serveClientWebInterface)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveClientWebsocket(server, w, r)
	})

	log.Println("Starting HTTP server on", config.Host)

	serverStarted = true
	err := http.ListenAndServe(config.Host, nil)

	if err != nil {
		log.Fatalln("HTTP ListenAndServe error:", err)
	}
}

func serveClientWebInterface(w http.ResponseWriter, r *http.Request) {
	log.Println("HTTP", r.Method, r.URL)

	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)

		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	http.ServeFile(w, r, "../client/web/index.html")
}

func handleSignals(signals chan os.Signal) {
	interrupt := <-signals

	log.Println("Interrupt received:", interrupt)

	if serverStarted {
		log.Println("Stopping HTTP server on", config.Host)
	}

	os.Exit(0)
}
