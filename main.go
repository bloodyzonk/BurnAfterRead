package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Get DB path from env or default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "messages.db"
	}

	// Init DB
	db, err := InitStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create server
	srv := NewServer(db)

	// Background cleanup
	go srv.StartCleanup()

	// Get address from env or default
	addr := ":8080"

	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}
