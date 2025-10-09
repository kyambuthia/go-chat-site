package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kyambuthia/go-chat-site/server/internal/api"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func main() {
	// Find project root
	root, err := findProjectRoot()
	if err != nil {
		log.Fatal("Failed to find project root:", err)
	}
	dbPath := filepath.Join(root, "chat.db")

	// Open log file
	logFile, err := os.OpenFile(filepath.Join(root, "server", "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	// Redirect log output to the file
	log.SetOutput(logFile)

	db, err := store.NewSqliteStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	hub := ws.NewHub()
	go hub.Run()

	api := api.NewAPI(db, hub)

	fmt.Println("Server listening on port 8081")
	log.Fatal(http.ListenAndServe(":8081", api))
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if dir == "/" {
			return "", errors.New("go.mod not found")
		}
		dir = filepath.Dir(dir)
	}
}
