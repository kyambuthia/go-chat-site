package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyambuthia/go-chat-site/server/internal/auth"
	coreledger "github.com/kyambuthia/go-chat-site/server/internal/core/ledger"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type fakeLedgerService struct {
	accountResp  coreledger.Account
	accountErr   error
	transferResp coreledger.Transfer
	transferErr  error

	lastGetUserID     int
	lastSenderID      int
	lastRecipient     string
	lastTransferCents int64
}

func (f *fakeLedgerService) GetAccount(ctx context.Context, userID int) (coreledger.Account, error) {
	_ = ctx
	f.lastGetUserID = userID
	return f.accountResp, f.accountErr
}

func (f *fakeLedgerService) SendTransferByUsername(ctx context.Context, fromUserID int, recipientUsername string, amountCents int64) (coreledger.Transfer, error) {
	_ = ctx
	f.lastSenderID = fromUserID
	f.lastRecipient = recipientUsername
	f.lastTransferCents = amountCents
	return f.transferResp, f.transferErr
}

func authReq(method, target string, body []byte, userID int) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	return req.WithContext(auth.WithUserID(req.Context(), userID))
}

func TestWalletHandler_GetWallet_UsesLedgerServiceAndPreservesResponseShape(t *testing.T) {
	svc := &fakeLedgerService{
		accountResp: coreledger.Account{
			ID:           "1",
			OwnerUserID:  42,
			BalanceCents: 1234,
			CurrencyCode: "USD",
		},
	}
	h := &WalletHandler{Ledger: svc}

	rr := httptest.NewRecorder()
	h.GetWallet(rr, authReq(http.MethodGet, "/api/wallet", nil, 42))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if svc.lastGetUserID != 42 {
		t.Fatalf("GetAccount userID = %d, want 42", svc.lastGetUserID)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := int(resp["id"].(float64)); got != 1 {
		t.Fatalf("id = %d, want 1", got)
	}
	if got := int(resp["user_id"].(float64)); got != 42 {
		t.Fatalf("user_id = %d, want 42", got)
	}
	if got := int(resp["balance_cents"].(float64)); got != 1234 {
		t.Fatalf("balance_cents = %d, want 1234", got)
	}
}

func TestWalletHandler_SendMoney_MapsInsufficientFundsToBadRequest(t *testing.T) {
	svc := &fakeLedgerService{transferErr: store.ErrInsufficientFund}
	h := &WalletHandler{Ledger: svc}

	body := []byte(`{"username":"bob","amount":12.50}`)
	rr := httptest.NewRecorder()
	h.SendMoney(rr, authReq(http.MethodPost, "/api/wallet/send", body, 7))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if svc.lastSenderID != 7 || svc.lastRecipient != "bob" || svc.lastTransferCents != 1250 {
		t.Fatalf("unexpected transfer call sender=%d recipient=%q cents=%d", svc.lastSenderID, svc.lastRecipient, svc.lastTransferCents)
	}
}

func TestWalletHandler_SendMoney_MapsMissingRecipientToNotFound(t *testing.T) {
	svc := &fakeLedgerService{transferErr: coreledger.ErrRecipientNotFound}
	h := &WalletHandler{Ledger: svc}

	body := []byte(`{"username":"missing","amount":1}`)
	rr := httptest.NewRecorder()
	h.SendMoney(rr, authReq(http.MethodPost, "/api/wallet/send", body, 7))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("user not found")) {
		t.Fatalf("expected user not found error body, got %q", rr.Body.String())
	}
}

func TestWalletHandler_SendMoney_MapsUnexpectedServiceErrorToInternalServerError(t *testing.T) {
	svc := &fakeLedgerService{transferErr: errors.New("db unavailable")}
	h := &WalletHandler{Ledger: svc}

	body := []byte(`{"username":"bob","amount":1}`)
	rr := httptest.NewRecorder()
	h.SendMoney(rr, authReq(http.MethodPost, "/api/wallet/send", body, 7))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
