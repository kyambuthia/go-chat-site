package auth

import (
	"encoding/json"
	"log"
)

func LogSecurityEvent(event string, fields map[string]any) {
	entry := map[string]any{
		"event": event,
	}
	for key, value := range fields {
		entry[key] = value
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("event=%s marshal_error=%v", event, err)
		return
	}
	log.Print(string(payload))
}
