package main

import (
	"fmt"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"learn-passkeys.com/m/db"
	"learn-passkeys.com/m/handlers"
)

func main() {
	db, err := db.Connect()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	wconfig := &webauthn.Config{
		RPDisplayName: "Learn Passkeys",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:5173"},
	}

	webAuthn, err := webauthn.New(wconfig)
	if err != nil {
		panic(err)
	}

	handler := &handlers.Handler{
		DB:       db,
		WebAuthn: webAuthn,
	}

	http.HandleFunc("/register/begin", handler.BeginRegistration)
	http.HandleFunc("/register/finish", handler.FinishRegistration)
	http.HandleFunc("/login/begin", handler.BeginLogin)
	http.HandleFunc("/login/finish", handler.FinishLogin)

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
