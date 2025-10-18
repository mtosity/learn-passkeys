package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"learn-passkeys.com/m/models"
)

func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	db := h.DB

	// get username from request body
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// check if user exists
	var user models.User
	if err := db.QueryRow("SELECT id, username, created_at FROM users WHERE username = $1", req.Username).Scan(&user.ID, &user.Username, &user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			// user does not exist, proceed with registration
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	// if user exists, return error
	if uuid.Nil != user.ID {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// create new user
	user = models.User{
		Username: req.Username,
	}
	err := db.QueryRow("INSERT INTO users (username) VALUES ($1) RETURNING id, username, created_at", user.Username).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// generate registration options
	options, sessionData, err := h.WebAuthn.BeginRegistration(
		&user,
	)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		return
	}

	// store session data in database
	_, err = db.Exec("INSERT INTO challenges (user_id, challenge, type, expires_at) VALUES ($1, $2, $3, $4)", user.ID, sessionData.Challenge, "registration", sessionData.Expires)
	if err != nil {
		http.Error(w, "Failed to store session data", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(options)
	if err != nil {
		http.Error(w, "Failed to marshal options", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	// 1. Parse credential response from request body
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		http.Error(w, "Invalid credential response", http.StatusBadRequest)
		return
	}
	// 2. Get username or user from request (how to identify the user?)
	// 3. Retrieve stored challenge from database

	challenge := parsedResponse.Response.CollectedClientData.Challenge

	var userID uuid.UUID
	err = h.DB.QueryRow("SELECT user_id FROM challenges WHERE challenge = $1 AND type = $2", challenge, "registration").Scan(&userID)
	if err != nil {
		http.Error(w, "Challenge not found", http.StatusBadRequest)
		return
	}

	var user models.User
	err = h.DB.QueryRow("SELECT id, username, created_at FROM users WHERE id = $1", userID).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// 4. Call webAuthn.FinishRegistration(user, session, response)
	sessionData := webauthn.SessionData{
		Challenge: challenge,
		UserID:    user.ID[:],
	}
	credential, err := h.WebAuthn.CreateCredential(&user, sessionData, parsedResponse)
	if err != nil {
		http.Error(w, "Failed to finish registration: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 5. Save credential to database
	_, err = h.DB.Exec("INSERT INTO credentials (id, user_id, public_key, sign_count, created_at) VALUES ($1, $2, $3, $4, $5)", credential.ID, user.ID, credential.PublicKey, credential.Authenticator.SignCount, time.Now())
	if err != nil {
		http.Error(w, "Failed to save credential", http.StatusInternalServerError)
		return
	}

	// 6. Delete used challenge
	_, err = h.DB.Exec("DELETE FROM challenges WHERE challenge = $1 AND type = $2", challenge, "registration")
	if err != nil {
		http.Error(w, "Failed to delete challenge", http.StatusInternalServerError)
		return
	}
	// 7. Return success
	w.WriteHeader(http.StatusOK)
}
