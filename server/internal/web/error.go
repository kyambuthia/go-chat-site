package web

import (
	"encoding/json"
	"log"
	"net/http"
)

func JSONError(w http.ResponseWriter, err error, code int) {
	log.Printf("error: %v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
