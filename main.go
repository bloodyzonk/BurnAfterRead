package main

import (
	"log"
	"net/http"
)

func main() {
	// Get DB path from env or default
	Config := LoadConfig()

	// Init DB
	db, err := InitStore(Config.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create server
	srv := NewServer(db, Config)

	// Background cleanup
	go srv.StartCleanup()

	// Get address from env or default
	addr := ":" + Config.Port

	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}
