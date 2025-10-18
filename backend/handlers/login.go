package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"learn-passkeys.com/m/models"
)

func (h *Handler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	db := h.DB

	// get username from request body
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// fetch user from database
	var user models.User
	if err := db.QueryRow("SELECT id, username, created_at FROM users WHERE username = $1", req.Username).Scan(&user.ID, &user.Username, &user.CreatedAt); err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// load user's credentials from db
	rows, err := db.Query("SELECT id, public_key, sign_count FROM credentials WHERE user_id = $1", user.ID)
	if err != nil {
		http.Error(w, "Failed to load user credentials", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var credentials []webauthn.Credential
	for rows.Next() {
		var creID []byte
		var publicKey []byte
		var signCount uint32

		if err := rows.Scan(&creID, &publicKey, &signCount); err != nil {
			http.Error(w, "Failed to scan credential", http.StatusInternalServerError)
			return
		}

		credentials = append(credentials, webauthn.Credential{
			ID:        creID,
			PublicKey: publicKey,
			Authenticator: webauthn.Authenticator{
				SignCount: signCount,
			},
		})
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Error iterating credentials", http.StatusInternalServerError)
		return
	}

	// Attach credientials to user
	if len(credentials) == 0 {
		http.Error(w, "No credentials found for user", http.StatusNotFound)
		return
	}
	user.Credentials = credentials

	// generate login options
	options, sessionData, err := h.WebAuthn.BeginLogin(
		&user,
	)
	if err != nil {
		http.Error(w, "Failed to begin login", http.StatusInternalServerError)
		return
	}

	// store session data
	_, err = db.Exec("INSERT INTO challenges (user_id, challenge, type, expires_at) VALUES ($1, $2, $3, $4)", user.ID, sessionData.Challenge, "login", sessionData.Expires)
	if err != nil {
		http.Error(w, "Failed to store session data", http.StatusInternalServerError)
		return
	}

	// respond with login options
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(options); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	// Parse response - Convert HTTP body to structured data
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		http.Error(w, "Invalid credential response", http.StatusBadRequest)
		return
	}

	// Extract challenge from response
	challenge := parsedResponse.Response.CollectedClientData.Challenge

	// Look up user - Find which user initiated this login attempt
	var userID uuid.UUID
	err = h.DB.QueryRow("SELECT user_id FROM challenges WHERE challenge = $1 AND type = $2", challenge, "login").Scan(&userID)
	if err != nil {
		http.Error(w, "Challenge not found", http.StatusBadRequest)
		return
	}

	// Load user data - Get user details from database
	var user models.User
	err = h.DB.QueryRow("SELECT id, username, created_at FROM users WHERE id = $1", userID).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// Load credentials - Get user's passkeys for verification
	rows, err := h.DB.Query("SELECT id, public_key, sign_count FROM credentials WHERE user_id = $1", user.ID)
	if err != nil {
		http.Error(w, "Failed to load user credentials", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var credentials []webauthn.Credential
	for rows.Next() {
		var creID []byte
		var publicKey []byte
		var signCount uint32

		if err := rows.Scan(&creID, &publicKey, &signCount); err != nil {
			http.Error(w, "Failed to scan credential", http.StatusInternalServerError)
			return
		}

		credentials = append(credentials, webauthn.Credential{
			ID:        creID,
			PublicKey: publicKey,
			Authenticator: webauthn.Authenticator{
				SignCount: signCount,
			},
		})
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Error iterating credentials", http.StatusInternalServerError)
		return
	}
	user.Credentials = credentials

	// Reconstruct SessionData - Rebuild the session info with challenge
	sessionData := &webauthn.SessionData{
		Challenge: challenge,
		UserID:    user.ID[:],
	}

	// Verify signature - The critical step! Validates the authenticator signed with the
	// correct private key
	credential, err := h.WebAuthn.ValidateLogin(&user, *sessionData, parsedResponse)
	if err != nil {
		http.Error(w, "Failed to validate login: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Update the credential's sign_count in the database (detect cloned authenticators)
	_, err = h.DB.Exec("UPDATE credentials SET sign_count = $1 WHERE id = $2",
		credential.Authenticator.SignCount, credential.ID)
	if err != nil {
		http.Error(w, "Failed to update sign count", http.StatusInternalServerError)
		return
	}

	// Delete the used challenge
	_, err = h.DB.Exec("DELETE FROM challenges WHERE challenge = $1 AND type = $2", challenge, "login")
	if err != nil {
		http.Error(w, "Failed to delete challenge", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Login successful", "user": user.Username})
}
