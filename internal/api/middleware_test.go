package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	corsHandler := CORSMiddleware(baseHandler)

	optionsReq := httptest.NewRequest(http.MethodOptions, "/api/files", nil)
	optionsW := httptest.NewRecorder()
	corsHandler.ServeHTTP(optionsW, optionsReq)

	if optionsW.Code != http.StatusOK {
		t.Fatalf("expected 200 for OPTIONS, got %d", optionsW.Code)
	}
	if optionsW.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing Access-Control-Allow-Origin header")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	getW := httptest.NewRecorder()
	corsHandler.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET, got %d", getW.Code)
	}
	if getW.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing Access-Control-Allow-Origin header on GET")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	loggedHandler := LoggingMiddleware(baseHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	loggedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
}
