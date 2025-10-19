package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
	"learn-passkeys.com/m/db"
	"learn-passkeys.com/m/handlers"
)

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment or use default
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "http://localhost:5173"
		}

		// Check if origin is allowed
		origin := r.Header.Get("Origin")
		for _, allowedOrigin := range strings.Split(allowedOrigins, ",") {
			if origin == strings.TrimSpace(allowedOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	db, err := db.Connect()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Get RP ID and origins from environment or use defaults
	rpID := os.Getenv("RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}

	rpOrigins := os.Getenv("RP_ORIGINS")
	var origins []string
	if rpOrigins == "" {
		origins = []string{"http://localhost:5173"}
	} else {
		for _, origin := range strings.Split(rpOrigins, ",") {
			origins = append(origins, strings.TrimSpace(origin))
		}
	}

	wconfig := &webauthn.Config{
		RPDisplayName: "Learn Passkeys",
		RPID:          rpID,
		RPOrigins:     origins,
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		panic(err)
	}

	handler := &handlers.Handler{
		DB:       db,
		WebAuthn: webAuthn,
	}

	// Create a new ServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/register/begin", handler.BeginRegistration)
	mux.HandleFunc("/register/finish", handler.FinishRegistration)
	mux.HandleFunc("/login/begin", handler.BeginLogin)
	mux.HandleFunc("/login/finish", handler.FinishLogin)

	// Wrap the mux with CORS middleware
	corsHandler := corsMiddleware(mux)

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", corsHandler)
}
