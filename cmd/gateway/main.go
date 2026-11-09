package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aiujdha/ai-gateway/internal/server"
)

func main() {
	address := os.Getenv("GATEWAY_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	srv := &http.Server{Addr: address, Handler: server.New().Handler(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("AI Gateway: http://%s", address)
	log.Fatal(srv.ListenAndServe())
}
