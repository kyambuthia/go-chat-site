package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kyambuthia/go-chat-site/server/internal/api"
	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func main() {
	root, err := findProjectRoot()
	if err != nil {
		log.Fatal("failed to find project root: ", err)
	}

	logFile, err := os.OpenFile(filepath.Join(root, "server", "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		log.Fatal("failed to open log file: ", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	jwtSecret := os.Getenv("JWT_SECRET")
	if err := auth.ConfigureJWT(jwtSecret); err != nil {
		log.Fatal("invalid JWT_SECRET: ", err)
	}

	dbPath := filepath.Join(root, "chat.db")
	dbStore, err := store.NewSqliteStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer dbStore.DB.Close()

	migrationsPath := filepath.Join(root, "server", "migrations")
	if err := migrate.RunMigrations(dbStore.DB, migrationsPath); err != nil {
		log.Fatal("migration failed: ", err)
	}

	hub := ws.NewHub()
	go hub.Run()
	defer hub.Shutdown()

	handler := api.NewAPI(dbStore, hub)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
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
