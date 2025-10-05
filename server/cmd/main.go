package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/api"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func main() {
	dbPath := flag.String("db", "chat.db", "path to the database file")
	flag.Parse()

	db, err := store.NewSqliteStore(*dbPath)
	if err != nil {
		log.Fatal(err)
	}

	hub := ws.NewHub()
	go hub.Run()

	api := api.NewAPI(db, hub)

	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", api)
}