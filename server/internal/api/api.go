package api

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	"github.com/kyambuthia/go-chat-site/server/internal/handlers"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
	"github.com/kyambuthia/go-chat-site/server/internal/ws"
)

func NewAPI(dataStore store.APIStore, hub *ws.Hub) http.Handler {
	mux := http.NewServeMux()

	authHandler := &handlers.AuthHandler{Store: dataStore}
	contactsHandler := &handlers.ContactsHandler{Store: dataStore}
	inviteHandler := &handlers.InviteHandler{Store: dataStore}
	walletHandler := &handlers.WalletHandler{Store: dataStore}
	meHandler := &handlers.MeHandler{Store: dataStore}

	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/login", auth.Login(dataStore))

	mux.Handle("/api/contacts", auth.Middleware(http.HandlerFunc(contactsHandler.GetContacts)))

	mux.Handle("/api/invites", auth.Middleware(http.HandlerFunc(inviteHandler.GetInvites)))
	mux.Handle("/api/invites/send", auth.Middleware(http.HandlerFunc(inviteHandler.SendInvite)))
	mux.Handle("/api/invites/accept", auth.Middleware(http.HandlerFunc(inviteHandler.AcceptInvite)))
	mux.Handle("/api/invites/reject", auth.Middleware(http.HandlerFunc(inviteHandler.RejectInvite)))

	mux.Handle("/api/me", auth.Middleware(http.HandlerFunc(meHandler.GetMe)))
	mux.Handle("/api/wallet", auth.Middleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("/api/wallet/send", auth.Middleware(http.HandlerFunc(walletHandler.SendMoney)))

	authenticator := func(token string) (int, string, error) {
		claims, err := auth.ValidateToken(token)
		if err != nil {
			return 0, "", err
		}
		user, err := dataStore.GetUserByID(claims.UserID)
		if err != nil {
			return 0, "", err
		}
		return user.ID, user.Username, nil
	}

	resolve := func(username string) (int, error) {
		user, err := dataStore.GetUserByUsername(username)
		if err != nil {
			return 0, err
		}
		return user.ID, nil
	}

	mux.HandleFunc("/ws", ws.WebSocketHandler(hub, authenticator, resolve))

	return loggingMiddleware(mux)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if reqID == "" {
			reqID = newRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("request_id=%s method=%s path=%s status=%d duration_ms=%d", reqID, r.Method, r.URL.Path, rec.status, time.Since(start).Milliseconds())
	})
}

func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
