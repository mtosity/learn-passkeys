# Integration Debugging Guide: SimpleWebAuthn + go-webauthn

**Date:** 2025-10-19
**Frontend:** `@simplewebauthn/browser` v13.2.2
**Backend:** `go-webauthn/webauthn` v0.14.0

This document chronicles the integration issues encountered when connecting a React frontend using SimpleWebAuthn with a Go backend using go-webauthn, and how we solved them.

---

## Problem 1: Invalid Attestation Format Error

### The Error
```
CreateCredential error: Invalid attestation format
```

### Symptoms
- Registration flow failed at `FinishRegistration`
- Frontend received: `Failed to finish registration: Invalid attestation format`
- Backend parsed the attestation format as "none" correctly
- Debug logs showed: `Attestation format from response: none`

### Root Cause
When we called `h.WebAuthn.CreateCredential()`, the library was rejecting "none" attestation format even though:
1. The browser was sending it correctly
2. We were parsing it correctly
3. The format was valid according to WebAuthn spec

**The actual issue:** We were not storing and reconstructing the complete `SessionData` from `BeginRegistration`.

### Investigation Process

#### Step 1: Added attestation configuration
First, we tried configuring the library to accept "none" attestation:

```go
// In BeginRegistration
options, sessionData, err := h.WebAuthn.BeginRegistration(
    &user,
    webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
    webauthn.WithAttestationFormats([]protocol.AttestationFormat{
        protocol.AttestationFormatNone,
    }),
)

// In main.go Config
wconfig := &webauthn.Config{
    RPDisplayName:         "Learn Passkeys",
    RPID:                  rpID,
    RPOrigins:             origins,
    AttestationPreference: protocol.PreferNoAttestation,
}
```

**Result:** Still failed with the same error.

#### Step 2: Compared SessionData
We logged the SessionData from BeginRegistration vs what we reconstructed in FinishRegistration:

**BeginRegistration produced:**
```
SessionData: {
    Challenge: "KkO3xMbdF6i46PCCNE5c8kZcZgQ50l46fUFRIiA4PDk"
    RelyingPartyID: "localhost"
    UserID: [144 40 247 138 ...]
    CredParams: [{Type:public-key Algorithm:-7} ...]  // 10 algorithms
}
```

**FinishRegistration reconstructed:**
```
SessionData: {
    Challenge: "KkO3xMbdF6i46PCCNE5c8kZcZgQ50l46fUFRIiA4PDk"
    RelyingPartyID: ""  // ❌ EMPTY!
    UserID: [144 40 247 138 ...]
    CredParams: []  // ❌ EMPTY!
}
```

### The Solution

**Store the ENTIRE SessionData** from `BeginRegistration` instead of just the challenge:

```go
// BeginRegistration - STORE COMPLETE SESSION DATA
sessionDataJSON, err := json.Marshal(sessionData)
if err != nil {
    http.Error(w, "Failed to marshal session data", http.StatusInternalServerError)
    return
}

_, err = db.Exec(`INSERT INTO challenges (user_id, challenge, type, expires_at)
    VALUES ($1, $2, $3, $4)`,
    user.ID, sessionDataJSON, "registration", sessionData.Expires)
```

```go
// FinishRegistration - RETRIEVE AND DESERIALIZE COMPLETE SESSION DATA
var sessionDataJSON []byte
err = h.DB.QueryRow(`SELECT user_id, challenge FROM challenges
    WHERE type = $1 ORDER BY created_at DESC LIMIT 1`, "registration").
    Scan(&userID, &sessionDataJSON)

var sessionData webauthn.SessionData
err = json.Unmarshal(sessionDataJSON, &sessionData)
```

**Why this fixed it:** The `CreateCredential` verification process needs:
- `RelyingPartyID` to validate the attestation object's RP ID hash
- `CredParams` to verify the credential algorithm is allowed

Without these fields, the library couldn't properly verify the credential, resulting in "Invalid attestation format" error.

### Key Learnings

1. **SessionData is opaque:** The WebAuthn spec says session data should be stored as-is and not reconstructed
2. **Don't cherry-pick fields:** Store the entire SessionData object from BeginRegistration
3. **Misleading error message:** "Invalid attestation format" was actually a validation failure, not a format parsing issue

---

## Problem 2: Backup Eligible Flag Inconsistency

### The Error
```
Login failed: Login verification failed: Failed to validate login:
Backup Eligible flag inconsistency detected during login validation
```

### Symptoms
- Registration completed successfully
- Credential saved to database
- Login flow failed at signature verification
- Error mentioned "backup eligible flag"

### Root Cause
The `webauthn.Credential` object contains **flags** that track important security properties:

```go
Flags: {
    UserPresent: true
    UserVerified: true
    BackupEligible: true   // ← Credential can be backed up (synced)
    BackupState: true      // ← Credential is currently backed up
}
```

We were **NOT storing these flags** in the database. When we loaded credentials during login, the flags were missing/default, causing the library to detect an inconsistency (flags changed between registration and login = possible security issue).

### Investigation Process

#### Step 1: Logged credential details
Added debug logging in FinishRegistration:

```go
fmt.Printf("DEBUG: Credential details: %+v\n", credential)
fmt.Printf("DEBUG: Authenticator: %+v\n", credential.Authenticator)
```

Output showed:
```
Flags: {
    UserPresent:true
    UserVerified:true
    BackupEligible:true
    BackupState:true
    raw:93
}
Authenticator: {
    AAGUID:[186 218 85 102 167 170 64 31 189 150 69 97 154 85 18 13]
    SignCount:0
    CloneWarning:false
    Attachment:platform
}
```

#### Step 2: Checked database schema
Our credentials table only had:
- `id`, `user_id`, `public_key`, `sign_count`, `created_at`

**Missing:** backup flags, AAGUID, attestation type

### The Solution

#### Part 1: Add columns to database schema

```sql
-- init-3.sql
ALTER TABLE credentials
ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS backup_state BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS attestation_type VARCHAR(50) DEFAULT '',
ADD COLUMN IF NOT EXISTS aaguid BYTEA;

CREATE INDEX IF NOT EXISTS idx_credentials_aaguid ON credentials(aaguid);
```

#### Part 2: Save flags during registration

```go
// handlers/register.go - FinishRegistration
_, err = h.DB.Exec(`INSERT INTO credentials
    (id, user_id, public_key, sign_count, backup_eligible, backup_state, attestation_type, aaguid, created_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
    credential.ID,
    user.ID,
    credential.PublicKey,
    credential.Authenticator.SignCount,
    credential.Flags.BackupEligible,      // ← NEW
    credential.Flags.BackupState,         // ← NEW
    credential.AttestationType,           // ← NEW
    credential.Authenticator.AAGUID,      // ← NEW
    time.Now(),
)
```

#### Part 3: Load flags during login

```go
// handlers/login.go - BeginLogin and FinishLogin
rows, err := db.Query(`SELECT id, public_key, sign_count, backup_eligible, backup_state, aaguid
    FROM credentials WHERE user_id = $1`, user.ID)

var credentials []webauthn.Credential
for rows.Next() {
    var creID []byte
    var publicKey []byte
    var signCount uint32
    var backupEligible bool
    var backupState bool
    var aaguid []byte

    rows.Scan(&creID, &publicKey, &signCount, &backupEligible, &backupState, &aaguid)

    credentials = append(credentials, webauthn.Credential{
        ID:        creID,
        PublicKey: publicKey,
        Flags: protocol.CredentialFlags{           // ← NEW
            BackupEligible: backupEligible,
            BackupState:    backupState,
        },
        Authenticator: webauthn.Authenticator{
            AAGUID:    aaguid,                     // ← NEW
            SignCount: signCount,
        },
    })
}
```

### Key Learnings

1. **Backup flags are security critical:** They detect if credentials are synced/cloned
2. **Store the complete credential state:** Not just ID, public key, and sign counter
3. **AAGUID matters:** Authenticator GUID is used for device identification
4. **Flags must be consistent:** The library validates that flags don't change unexpectedly between operations

### What Are These Flags?

- **BackupEligible:** Indicates if the credential CAN be backed up (e.g., stored in iCloud Keychain, Google Password Manager)
- **BackupState:** Indicates if the credential IS CURRENTLY backed up/synced
- **AAGUID:** Authenticator Attestation GUID - identifies the make/model of the authenticator
- **AttestationType:** Format of attestation ("none", "packed", "fido-u2f", etc.)

---

## Problem 3: Empty JSON Response

### The Error
```
Registration failed: Failed to execute 'json' on 'Response':
Unexpected end of JSON input
```

### Symptoms
- Backend registration succeeded
- Debug log showed: `DEBUG: Credential created successfully`
- Frontend received 200 OK but empty body
- Frontend tried to parse JSON and failed

### Root Cause
After successful registration, the backend returned:

```go
w.WriteHeader(http.StatusOK)
// No response body!
```

But the frontend expected JSON:

```typescript
const finishResponse = await fetch(`${API_URL}/register/finish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credential),
});

return await finishResponse.json(); // ← Expects JSON!
```

### The Solution

Return proper JSON response:

```go
// handlers/register.go - FinishRegistration
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]string{
    "status":  "success",
    "message": "Registration successful",
})
```

### Key Learnings

1. **Always match frontend expectations:** If frontend expects JSON, return JSON
2. **Set Content-Type header:** Helps debugging and proper parsing
3. **Empty 200 responses are confusing:** Always return meaningful response bodies

---

## Complete Working Implementation Summary

### Database Schema
```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Credentials table (complete)
CREATE TABLE credentials (
    id BYTEA PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key BYTEA NOT NULL,
    sign_count INTEGER NOT NULL DEFAULT 0,
    transports TEXT[],
    backup_eligible BOOLEAN DEFAULT false,
    backup_state BOOLEAN DEFAULT false,
    attestation_type VARCHAR(50) DEFAULT '',
    aaguid BYTEA,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Challenges table (stores full SessionData as JSON)
CREATE TABLE challenges (
    challenge BYTEA PRIMARY KEY,  -- Stores JSON-serialized SessionData
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);
```

### Backend Configuration
```go
// main.go
wconfig := &webauthn.Config{
    RPDisplayName:         "Learn Passkeys",
    RPID:                  "localhost",
    RPOrigins:             []string{"http://localhost:5173"},
    AttestationPreference: protocol.PreferNoAttestation,
}
```

### Registration Flow
```go
// BeginRegistration
options, sessionData, err := h.WebAuthn.BeginRegistration(
    &user,
    webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
    webauthn.WithAttestationFormats([]protocol.AttestationFormat{
        protocol.AttestationFormatNone,
    }),
)

// Store COMPLETE SessionData
sessionDataJSON, _ := json.Marshal(sessionData)
db.Exec("INSERT INTO challenges (user_id, challenge, type, expires_at) VALUES ($1, $2, $3, $4)",
    user.ID, sessionDataJSON, "registration", sessionData.Expires)

// FinishRegistration
var sessionDataJSON []byte
db.QueryRow("SELECT challenge FROM challenges WHERE type = $1", "registration").
    Scan(&sessionDataJSON)

var sessionData webauthn.SessionData
json.Unmarshal(sessionDataJSON, &sessionData)

credential, err := h.WebAuthn.CreateCredential(&user, sessionData, parsedResponse)

// Save with ALL fields
db.Exec(`INSERT INTO credentials (..., backup_eligible, backup_state, aaguid, ...)
    VALUES (..., $5, $6, $7, ...)`,
    credential.Flags.BackupEligible,
    credential.Flags.BackupState,
    credential.Authenticator.AAGUID)
```

### Login Flow
```go
// BeginLogin & FinishLogin - Load credentials with flags
rows := db.Query(`SELECT id, public_key, sign_count, backup_eligible, backup_state, aaguid
    FROM credentials WHERE user_id = $1`, user.ID)

for rows.Next() {
    var backupEligible, backupState bool
    var aaguid []byte
    rows.Scan(&creID, &publicKey, &signCount, &backupEligible, &backupState, &aaguid)

    credentials = append(credentials, webauthn.Credential{
        ID:        creID,
        PublicKey: publicKey,
        Flags: protocol.CredentialFlags{
            BackupEligible: backupEligible,
            BackupState:    backupState,
        },
        Authenticator: webauthn.Authenticator{
            AAGUID:    aaguid,
            SignCount: signCount,
        },
    })
}
```

---

## General Integration Tips

### 1. Always Use Debug Logging During Integration
```go
fmt.Printf("DEBUG: SessionData: %+v\n", sessionData)
fmt.Printf("DEBUG: Credential: %+v\n", credential)
fmt.Printf("DEBUG: Flags: %+v\n", credential.Flags)
```

### 2. Store Opaque Session Data
The WebAuthn spec recommends treating session data as opaque:
- Don't reconstruct it from individual fields
- Store the complete SessionData object
- Serialize to JSON or use secure session cookies

### 3. Store Complete Credential State
Don't just store the minimum (ID, public key, counter). Store:
- Backup flags (BackupEligible, BackupState)
- AAGUID (authenticator identifier)
- Attestation type
- Transports (optional but useful)

### 4. Match Library Versions Carefully
- `@simplewebauthn/browser` v13.x works with `go-webauthn/webauthn` v0.13.x - v0.14.x
- Both libraries follow WebAuthn Level 3 spec
- Check release notes for breaking changes

### 5. Error Messages Can Be Misleading
- "Invalid attestation format" → Check SessionData completeness
- "Backup flag inconsistency" → Check credential flags storage
- "Challenge not found" → Check challenge storage/lookup logic
- "Origin mismatch" → Check RPID and RPOrigins config

### 6. Test Incrementally
1. Get registration working first
2. Then test login
3. Then add session management
4. Then test logout and protected routes

### 7. Clean Test Data Between Changes
When you change database schema or credential structure:
```sql
DELETE FROM credentials;
DELETE FROM challenges;
DELETE FROM users;
```
Old credentials with missing fields will cause errors!

---

## Tools That Helped Debug

1. **Browser DevTools Network Tab:** See exact request/response payloads
2. **Backend Debug Logging:** Print SessionData, Credentials, Flags
3. **PostgreSQL Direct Queries:** Check what's actually stored
4. **Go's `%+v` format verb:** Prints struct fields with names
5. **SimpleWebAuthn's error messages:** Usually specific about what failed

---

## Resources

- **go-webauthn docs:** https://pkg.go.dev/github.com/go-webauthn/webauthn/webauthn
- **SimpleWebAuthn docs:** https://simplewebauthn.dev/
- **WebAuthn Spec:** https://www.w3.org/TR/webauthn-3/
- **WebAuthn Guide (visual):** https://webauthn.guide/

---

## Conclusion

The integration between SimpleWebAuthn browser and go-webauthn backend works well once you understand:

1. **Store complete SessionData** - don't reconstruct it
2. **Store all credential flags** - they're security-critical
3. **Return proper JSON responses** - match frontend expectations
4. **Debug systematically** - log everything during integration

These issues are common integration pitfalls, not bugs in the libraries. Both libraries correctly implement the WebAuthn specification. The challenge is understanding what data needs to be persisted and how to properly reconstruct the credential state.

**Final Status:** ✅ Full registration and login flow working with v0.14.0
