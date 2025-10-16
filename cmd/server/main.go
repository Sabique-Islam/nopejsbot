package main

import (
	"log"
	"net/http"
	"os"
	"example.com/nopejsbot/internal/webhook"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.Handle("/webhook", webhook.Handler())
	log.Printf("Listening on :%s …", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}