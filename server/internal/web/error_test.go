package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONError(t *testing.T) {
	rr := httptest.NewRecorder()
	JSONError(rr, errors.New("bad request"), http.StatusBadRequest)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if !strings.Contains(rr.Body.String(), `"error":"bad request"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
