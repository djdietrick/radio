package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequire_RejectsMissingToken(t *testing.T) {
	a := NewAuthenticator([]byte("s"), time.Hour)
	rec := httptest.NewRecorder()
	a.Require(okHandler()).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
}

func TestRequire_AcceptsBearerHeader(t *testing.T) {
	a := NewAuthenticator([]byte("s"), time.Hour)
	tok, _ := a.Issue("u1", false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	var gotUID string
	a.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUID = UserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if gotUID != "u1" {
		t.Fatalf("context user id = %q, want u1", gotUID)
	}
}

func TestRequire_AcceptsQueryTokenFallback(t *testing.T) {
	a := NewAuthenticator([]byte("s"), time.Hour)
	tok, _ := a.Issue("u1", false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/?token="+tok, nil)
	a.Require(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (query-token fallback for WS)", rec.Code)
	}
}

func TestRequireAdmin_GatesNonAdmin(t *testing.T) {
	a := NewAuthenticator([]byte("s"), time.Hour)

	// Non-admin token.
	tok, _ := a.Issue("u1", false)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	a.Require(a.RequireAdmin(okHandler())).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin got %d, want 403", rec.Code)
	}

	// Admin token.
	adminTok, _ := a.Issue("admin", true)
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("Authorization", "Bearer "+adminTok)
	a.Require(a.RequireAdmin(okHandler())).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("admin got %d, want 200", rec2.Code)
	}
}
