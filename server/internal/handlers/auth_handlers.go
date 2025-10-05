package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type AuthHandler struct {
	Store *store.SqliteStore
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(creds.Password) < 8 {
		http.Error(w, "Password too short", http.StatusBadRequest)
		return
	}

	id, err := h.Store.CreateUser(creds.Username, creds.Password)
	if err != nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       id,
		"username": creds.Username,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	row, err := h.Store.GetUserByUsername(creds.Username)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	var id int
	var hashedPassword string
	if err := row.Scan(&id, &hashedPassword); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Login attempt for user: %s", creds.Username)
	log.Printf("Password from request: %s", creds.Password)
	log.Printf("Hashed password from DB: %s", hashedPassword)

	if !auth.CheckPassword(creds.Password, hashedPassword) {
		log.Println("Password check failed")
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	log.Println("Password check successful")
	token, err := auth.GenerateToken(id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
