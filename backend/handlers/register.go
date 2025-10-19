package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithAttestationFormats([]protocol.AttestationFormat{
			protocol.AttestationFormatNone,
		}),
	)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		return
	}

	// DEBUG: Print full SessionData
	fmt.Printf("DEBUG BeginRegistration SessionData: %+v\n", sessionData)

	// Serialize entire SessionData to JSON
	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		http.Error(w, "Failed to marshal session data", http.StatusInternalServerError)
		return
	}

	// store session data in database
	_, err = db.Exec("INSERT INTO challenges (user_id, challenge, type, expires_at) VALUES ($1, $2, $3, $4)", user.ID, sessionDataJSON, "registration", sessionData.Expires)
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

	// Get the most recent registration challenge (for development simplicity)
	var userID uuid.UUID
	var sessionDataJSON []byte
	err = h.DB.QueryRow("SELECT user_id, challenge FROM challenges WHERE type = $1 ORDER BY created_at DESC LIMIT 1", "registration").Scan(&userID, &sessionDataJSON)
	if err != nil {
		fmt.Printf("Challenge lookup error: %v\n", err)
		http.Error(w, "Challenge not found", http.StatusBadRequest)
		return
	}

	// 4. Deserialize SessionData from database
	var sessionData webauthn.SessionData
	err = json.Unmarshal(sessionDataJSON, &sessionData)
	if err != nil {
		fmt.Printf("Unmarshal error: %v\n", err)
		http.Error(w, "Failed to unmarshal session data", http.StatusInternalServerError)
		return
	}

	// Verify the challenge matches
	if string(sessionData.Challenge) != string(challenge) {
		fmt.Printf("Challenge mismatch: stored=%s, received=%s\n", sessionData.Challenge, challenge)
		http.Error(w, "Challenge mismatch", http.StatusBadRequest)
		return
	}

	var user models.User
	err = h.DB.QueryRow("SELECT id, username, created_at FROM users WHERE id = $1", userID).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	fmt.Printf("DEBUG FinishRegistration SessionData: %+v\n", sessionData)
	fmt.Printf("DEBUG: User ID: %s\n", user.ID)
	fmt.Printf("DEBUG: Challenge length: %d\n", len(challenge))
	fmt.Printf("DEBUG: Parsed response type: %T\n", parsedResponse)
	fmt.Printf("DEBUG: Attestation format from response: %s\n", parsedResponse.Response.AttestationObject.Format)
	fmt.Printf("DEBUG: Attestation statement: %+v\n", parsedResponse.Response.AttestationObject.AttStatement)

	credential, err := h.WebAuthn.CreateCredential(&user, sessionData, parsedResponse)
	if err != nil {
		fmt.Printf("CreateCredential error: %v\n", err)
		fmt.Printf("Error type: %T\n", err)
		// Try to print more details about the error
		fmt.Printf("Full error string: %+v\n", err)
		http.Error(w, "Failed to finish registration: "+err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("DEBUG: Credential created successfully, ID length: %d\n", len(credential.ID))
	fmt.Printf("DEBUG: Credential details: %+v\n", credential)
	fmt.Printf("DEBUG: Authenticator: %+v\n", credential.Authenticator)

	// 5. Save credential to database with backup flags
	_, err = h.DB.Exec(`INSERT INTO credentials
		(id, user_id, public_key, sign_count, backup_eligible, backup_state, attestation_type, aaguid, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		credential.ID,
		user.ID,
		credential.PublicKey,
		credential.Authenticator.SignCount,
		credential.Flags.BackupEligible,
		credential.Flags.BackupState,
		credential.AttestationType,
		credential.Authenticator.AAGUID,
		time.Now(),
	)
	if err != nil {
		http.Error(w, "Failed to save credential", http.StatusInternalServerError)
		return
	}

	// 6. Delete used challenge
	_, err = h.DB.Exec("DELETE FROM challenges WHERE user_id = $1 AND type = $2", user.ID, "registration")
	if err != nil {
		http.Error(w, "Failed to delete challenge", http.StatusInternalServerError)
		return
	}
	// 7. Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Registration successful",
	})
}
