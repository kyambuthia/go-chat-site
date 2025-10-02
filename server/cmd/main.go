package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/api"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/test/testhelpers"
)

func main() {
	db, err := store.NewSqliteStore("chat.db")
	if err != nil {
		log.Fatal(err)
	}

	if err := testhelpers.RunMigrations(db.DB, "migrations"); err != nil {
		log.Fatal(err)
	}

	api := api.NewAPI(db)

	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", api)
}