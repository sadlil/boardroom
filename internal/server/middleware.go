package server

import (
	"net/http"
)

// SecurityMiddleware applies basic security headers and protections.
func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// Prevent MIME-type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Basic Content Security Policy (whitelisting necessary CDNs for Tailwind, HTMX, marked.js, and Google Fonts)
		csp := "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://unpkg.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self'"
		w.Header().Set("Content-Security-Policy", csp)

		// Optional: basic auth layer or rate limiting logic can be injected here.

		next.ServeHTTP(w, r)
	})
}
