package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityMiddleware(t *testing.T) {
	// A dummy handler to test if middleware passes execution to next
	called := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap handler in middleware
	handlerUnderTest := SecurityMiddleware(dummyHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handlerUnderTest.ServeHTTP(rr, req)

	// Verify the middleware passed execution to the next handler
	if !called {
		t.Error("Expected next handler to be called in middleware chain")
	}

	// Check response status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify headers were injected
	expectedHeaders := map[string]string{
		"X-Frame-Options":         "DENY",
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://unpkg.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self'",
	}

	for key, wantValue := range expectedHeaders {
		gotValue := rr.Header().Get(key)
		if gotValue != wantValue {
			t.Errorf("Header %s: got '%v' want '%v'", key, gotValue, wantValue)
		}
	}
}
