# Development Session Notes

## Session 2025-10-19: Integration Testing & Debugging

### What We Accomplished Today

Successfully integrated and debugged the complete fullstack passkey authentication system! The entire registration and login flow now works end-to-end.

#### 1. Frontend-Backend Integration Testing

**Initial State:**
- Backend: All 4 endpoints implemented (registration begin/finish, login begin/finish)
- Frontend: React app with Register, Login, and Secret pages
- Database: PostgreSQL with users, credentials, and challenges tables
- CORS: Configured for localhost:5173

**Testing Started:**
- Launched both frontend (localhost:5173) and backend (localhost:8080)
- Attempted registration flow
- Immediately hit: "Invalid attestation format" error

#### 2. Debugging Session: Invalid Attestation Format

**Problem:** Registration failed with `CreateCredential error: Invalid attestation format`

**Investigation Process:**

**Attempt 1: Configure Attestation Preferences**
- Added `WithConveyancePreference(protocol.PreferNoAttestation)` to BeginRegistration
- Added `WithAttestationFormats([]protocol.AttestationFormat{protocol.AttestationFormatNone})` to BeginRegistration
- Added `AttestationPreference: protocol.PreferNoAttestation` to Config
- **Result:** Still failed

**Attempt 2: Check Library Version**
- Downgraded from v0.14.0 to v0.13.4
- **Result:** Still failed - not a version issue

**Attempt 3: Deep Debugging with Logs**
Added comprehensive debug logging:
```go
fmt.Printf("DEBUG: Attestation format from response: %s\n", parsedResponse.Response.AttestationObject.Format)
fmt.Printf("DEBUG: SessionData: %+v\n", sessionData)
```

**Breakthrough Discovery:**
Compared SessionData from BeginRegistration vs FinishRegistration:

**BeginRegistration:**
```
RelyingPartyID: "localhost"
CredParams: [{Type:public-key Algorithm:-7} ... 10 algorithms]
```

**FinishRegistration (reconstructed):**
```
RelyingPartyID: ""  ← EMPTY!
CredParams: []      ← EMPTY!
```

**Root Cause Identified:**
We were only storing the `challenge` field in the database and reconstructing SessionData manually. The library needs the COMPLETE SessionData including RelyingPartyID and CredParams for proper attestation validation.

**The Fix:**
Store and retrieve the ENTIRE SessionData object as JSON:

```go
// BeginRegistration - Store complete SessionData
sessionDataJSON, err := json.Marshal(sessionData)
db.Exec("INSERT INTO challenges (...) VALUES (..., $2, ...)", sessionDataJSON)

// FinishRegistration - Retrieve and deserialize
var sessionDataJSON []byte
db.QueryRow("SELECT challenge FROM challenges...").Scan(&sessionDataJSON)
var sessionData webauthn.SessionData
json.Unmarshal(sessionDataJSON, &sessionData)
```

**Result:** ✅ Registration succeeded!

#### 3. Debugging Session: Backup Eligible Flag Inconsistency

**Problem:** Registration worked, but login failed with:
```
Backup Eligible flag inconsistency detected during login validation
```

**Investigation:**
Added debug logging to see credential details:
```go
fmt.Printf("DEBUG: Credential details: %+v\n", credential)
fmt.Printf("DEBUG: Authenticator: %+v\n", credential.Authenticator)
```

**Discovery:**
Credentials have security-critical flags we weren't storing:
```
Flags: {
    UserPresent: true
    UserVerified: true
    BackupEligible: true
    BackupState: true
}
Authenticator: {
    AAGUID: [186 218 85 102...]
    SignCount: 0
}
```

**Root Cause:**
Our database schema only had: id, user_id, public_key, sign_count
Missing: backup_eligible, backup_state, aaguid, attestation_type

When the library compared flags between registration and login, it detected inconsistency (missing flags = potential security issue).

**The Fix:**

**Step 1: Database Migration (init-3.sql)**
```sql
ALTER TABLE credentials
ADD COLUMN backup_eligible BOOLEAN DEFAULT false,
ADD COLUMN backup_state BOOLEAN DEFAULT false,
ADD COLUMN attestation_type VARCHAR(50) DEFAULT '',
ADD COLUMN aaguid BYTEA;
```

**Step 2: Save Flags During Registration**
```go
db.Exec(`INSERT INTO credentials
    (id, user_id, public_key, sign_count, backup_eligible, backup_state, aaguid, ...)
    VALUES ($1, $2, $3, $4, $5, $6, $7, ...)`,
    credential.ID,
    user.ID,
    credential.PublicKey,
    credential.Authenticator.SignCount,
    credential.Flags.BackupEligible,
    credential.Flags.BackupState,
    credential.Authenticator.AAGUID,
)
```

**Step 3: Load Flags During Login**
```go
rows := db.Query(`SELECT id, public_key, sign_count, backup_eligible, backup_state, aaguid
    FROM credentials WHERE user_id = $1`, user.ID)

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
```

**Result:** ✅ Login succeeded!

#### 4. Minor Fix: Empty JSON Response

**Problem:** Registration succeeded on backend but frontend showed JSON parse error

**Cause:** Backend returned HTTP 200 but no response body

**Fix:**
```go
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(map[string]string{
    "status":  "success",
    "message": "Registration successful",
})
```

### Key Learnings

#### 1. SessionData Must Be Stored Complete
**Wrong approach:**
```go
// Only store challenge, reconstruct rest
sessionData := webauthn.SessionData{
    Challenge: challenge,
    UserID:    user.ID[:],
}
```

**Correct approach:**
```go
// Store entire SessionData from BeginRegistration
sessionDataJSON, _ := json.Marshal(sessionData)
// Later: json.Unmarshal(sessionDataJSON, &sessionData)
```

**Why:** The library needs RelyingPartyID, CredParams, and other fields for proper verification.

#### 2. Credential Flags Are Security-Critical

**BackupEligible & BackupState flags:**
- Indicate if credential can be/is backed up (synced to cloud)
- Library validates these don't change unexpectedly
- Changing flags = potential credential cloning attack

**AAGUID:**
- Authenticator Attestation GUID
- Identifies the make/model of the authenticator
- Used for security policy decisions

**Must store:** backup_eligible, backup_state, aaguid, attestation_type

#### 3. Error Messages Can Be Misleading

"Invalid attestation format" sounds like a parsing issue, but was actually:
- Missing SessionData fields for verification
- The library couldn't validate because data was incomplete

"Backup flag inconsistency" was clear once we knew what to look for.

#### 4. go-webauthn v0.14.0 Works Fine

We downgraded to v0.13.4 thinking it was a version issue, but the problems were in our implementation. Upgraded back to v0.14.0 and everything works.

#### 5. Debug Logging Is Essential

Printf debugging with `%+v` format helped immensely:
- Compare expected vs actual SessionData
- See all credential flags and fields
- Understand what the library needs

### What's Working Now

**✅ Complete Authentication Flow:**
1. User registers with passkey (Touch ID, Face ID, security key, etc.)
2. Credential stored with all flags and metadata
3. User logs in with passkey
4. Session created (localStorage for now)
5. Protected routes work (Secret page)

**✅ Security Features:**
- Challenge-based authentication
- Sign counter tracking (clone detection)
- Backup flag validation
- Origin verification
- HTTPS/localhost enforcement

**✅ Database Integrity:**
- Complete credential state stored
- SessionData properly persisted
- Challenge cleanup after use
- Foreign key relationships

### Files Modified Today

**Database:**
- `init-3.sql` - Added backup flags columns

**Backend:**
- `handlers/register.go`:
  - Store complete SessionData as JSON
  - Save all credential flags (backup_eligible, backup_state, aaguid)
  - Return proper JSON response
- `handlers/login.go`:
  - Load credentials with backup flags
  - Reconstruct complete Credential objects
- `main.go`:
  - Added AttestationPreference configuration

**Documentation:**
- `docs/integration-debugging-guide.md` - Comprehensive debugging guide (NEW)
- `docs/session-notes.md` - This session documentation

### Current Stack

**Versions:**
- `@simplewebauthn/browser`: v13.2.2
- `go-webauthn/webauthn`: v0.14.0 ✅
- React: v19.1.1
- PostgreSQL: Alpine (Docker)

**Architecture:**
- Frontend: http://localhost:5173 (Vite dev server)
- Backend: http://localhost:8080 (Go server)
- Database: localhost:5432 (PostgreSQL in Docker)

### Next Steps

**Completed Core Features:**
- ✅ Registration flow
- ✅ Login flow
- ✅ Protected routes (basic localStorage check)

**Optional Enhancements:**
- [ ] Proper session management (cookies or JWT)
- [ ] Logout endpoint
- [ ] Authentication middleware
- [ ] Multiple credentials per user
- [ ] Credential management UI (view/delete passkeys)
- [ ] Error handling improvements
- [ ] Request logging
- [ ] CloneWarning detection alerts
- [ ] Expired challenge cleanup job

**Learning Goals:**
- ✅ Understand WebAuthn registration ceremony
- ✅ Understand WebAuthn authentication ceremony
- ✅ Debug real integration issues
- ✅ Learn backup flag importance
- ✅ Complete fullstack passkey implementation
- [ ] Write comprehensive blog post
- [ ] Test on multiple devices/browsers

### Resources That Helped

- **go-webauthn source code:** Reading the actual verification logic helped understand what fields were needed
- **SimpleWebAuthn docs:** Clear examples of request/response formats
- **WebAuthn spec:** Understanding backup flags and their purpose
- **Printf debugging:** The old reliable method still wins

### Time Investment

- **Total session:** ~3 hours
- **"Invalid attestation format" debugging:** ~1.5 hours
- **"Backup flag inconsistency" debugging:** ~1 hour
- **Documentation:** ~30 minutes

**Worth it:** Absolutely! Deep understanding of WebAuthn internals achieved.

---

## Session 2025-10-18: Backend Login Flow

### What We Built Today

#### 1. Login Handlers Implementation
Created complete login flow in `backend/handlers/login.go`:

**BeginLogin (`POST /login/begin`):**
1. Parse username from request body
2. Query user from database (return 404 if not found - don't create new users!)
3. Load user's existing credentials from credentials table
4. Attach credentials to `user.Credentials` field (required by webauthn.User interface)
5. Verify user has at least one credential
6. Call `webAuthn.BeginLogin(user)` to generate challenge and assertion options
7. Store challenge in challenges table with type "login" and expiration
8. Return assertion options as JSON (includes `allowCredentials` list)

**FinishLogin (`POST /login/finish`):**
1. Parse authentication assertion using `protocol.ParseCredentialRequestResponseBody()`
2. Extract challenge from the assertion
3. Look up user_id from challenges table using challenge
4. Load user from database
5. Load user's credentials from database (same pattern as BeginLogin)
6. Reconstruct `SessionData` with challenge and user ID
7. Call `webAuthn.ValidateLogin()` to verify signature (library checks counter internally!)
8. **Update credential sign_count** in database (critical for clone detection)
9. Delete used challenge from database
10. Return success response

#### 2. Routes Added to main.go
Added login endpoints:
```go
http.HandleFunc("/login/begin", handler.BeginLogin)
http.HandleFunc("/login/finish", handler.FinishLogin)
```

Server now has 4 endpoints:
- `POST /register/begin` - Start registration
- `POST /register/finish` - Complete registration
- `POST /login/begin` - Start login
- `POST /login/finish` - Complete login

#### 3. User Model Enhancement
Modified `models/user.go`:
- Added `Credentials []webauthn.Credential` field to User struct
- Updated `WebAuthnCredentials()` to return `u.Credentials` instead of empty slice
- Credentials are loaded from database and attached to user during login flow

### Key Concepts Learned

#### Login vs Registration - Critical Differences
**Registration:**
- User doesn't exist → Create new user
- No credentials exist → Create new credential
- Goal: Set up new account

**Login:**
- User must exist → Return error if not found
- Credentials must exist → Load from database
- Goal: Verify user owns the private key

#### Sign Counter Security
**Purpose:** Detect cloned authenticators (security breach indicator)

**How it works:**
- Each authenticator has a counter that increments with each signature
- Counter starts at 0, increases by 1 each login
- Server stores counter value in database
- If counter goes backwards or stays same → Possible cloned credential!

**Flow:**
1. Load old counter from database (e.g., 10)
2. `ValidateLogin()` checks: newCount > storedCount?
3. Library sets `CloneWarning = true` if suspicious
4. Update database with new counter value for next login

**Important:** The library handles counter comparison internally - we just save the updated value!

#### Database Patterns
**`db.Query()` vs `db.QueryRow()`:**
- `Query()`: Multiple rows, returns `*sql.Rows`, must call `defer rows.Close()`
- `QueryRow()`: Single row, auto-closes, no need for `Close()`

**Why `defer rows.Close()`?**
- Returns database connection to the pool
- Prevents connection leaks
- Works even if function returns early (errors, etc.)
- Without it: Eventually run out of connections, server fails

**Loading credentials pattern:**
```go
rows, err := db.Query("SELECT ... FROM credentials WHERE user_id = $1")
defer rows.Close()  // Critical!

for rows.Next() {
    rows.Scan(...)
}
rows.Err()  // Check for iteration errors
```

#### User Struct Design: Database vs Domain Models
**Question:** Why add `Credentials` field to User if it's not in the database?

**Answer:** The `webauthn.User` interface requires `WebAuthnCredentials()` method that returns credentials. We need to populate this before calling `BeginLogin()`.

**Common approaches:**
1. **Single model with optional fields** (what we're using) - Simple, good for small projects
2. **Separate UserRow and User models** - Clearer separation, better for large projects
3. **Embedded structs** - Hybrid approach

Our approach is fine for learning, but production apps often separate database models from domain models.

#### Library Responsibilities vs Our Code
**go-webauthn library handles:**
- Signature verification (cryptography)
- Counter comparison (security check)
- Setting CloneWarning flag
- Challenge validation
- Origin verification (anti-phishing)

**Our code handles:**
- Loading credentials from database
- Saving updated counter to database
- Challenge storage and retrieval
- User authentication state (sessions - not yet implemented)

### Design Decisions & Trade-offs

#### Credential Loading Strategy
**Current approach:** Query credentials and attach to user struct before calling library

**Why not query inside `WebAuthnCredentials()` method?**
- Method doesn't have access to database connection
- Would need global DB variable (bad practice)
- Interface doesn't allow passing parameters

**Solution:** Load credentials in handler, store temporarily in `user.Credentials`, library reads from there.

#### Session Management (Not Yet Implemented)
Currently `FinishLogin` just returns `{"status": "ok"}` - no persistent session!

**Next steps needed:**
- Add session storage (cookies or JWT)
- Create authentication middleware
- Add logout endpoint
- Protect routes requiring authentication

### Questions Answered

1. **Where is the counter checking happening?**
   - Inside `h.WebAuthn.ValidateLogin()` on line 163 of login.go
   - Library compares old counter (from credentials we loaded) with new counter (from authenticator)
   - Sets `CloneWarning = true` if suspicious, but doesn't fail by default
   - Returns updated credential with new counter value
   - We save the new counter to database for next login

2. **Why do we need both credentials table AND user.Credentials field?**
   - **Database table:** Permanent storage, survives server restarts
   - **Struct field:** Temporary in-memory holder for one HTTP request
   - Database is source of truth, struct is just a transport mechanism

3. **Is it common to have struct fields not stored in database?**
   - Yes! Common patterns:
     - Computed fields (e.g., FullName from FirstName + LastName)
     - Lazy-loaded relationships (e.g., credentials loaded on demand)
     - Interface requirements (e.g., webauthn.User needs credentials)
   - Small projects: Single model with optional fields (our approach)
   - Large projects: Separate database models from domain models

4. **Why call `rows.Close()`?**
   - Database connections are pooled (limited number available)
   - `Query()` borrows a connection from the pool
   - `Close()` returns connection to pool for reuse
   - Without it: Connection leak, eventually run out, server hangs
   - `defer` ensures it runs even if we return early (errors)

### Testing Done
- ✅ Code compiles with `go build`
- ✅ All 4 routes registered correctly
- ⚠️ Not tested end-to-end yet (requires frontend + real authenticator)

### What's Working Now
Backend API is **functionally complete** for basic passkey authentication:
- User registration with passkey creation
- User login with passkey verification
- Challenge-based security
- Clone detection via sign counter
- All endpoints implemented and routes registered

**Missing pieces:**
- Session management (no persistent login state)
- Frontend UI to interact with authenticator
- CORS headers for frontend-backend communication
- Protected routes demonstrating authentication

### Next Session Tasks

#### Priority 1: Session Management (Recommended Next)
Add persistent authentication state:
- Implement session storage (cookies or JWT tokens)
- Create authentication middleware to protect routes
- Add logout endpoint (`POST /logout`)
- Add protected route example (e.g., `GET /secret` - only for logged-in users)
- Consider using `gorilla/sessions` library for cookie-based sessions

**Why this matters:**
- Currently login succeeds but nothing happens
- No way to check if user is authenticated
- Can't protect routes requiring authentication

#### Priority 2: Frontend React App
Build UI to test the full flow:
- Initialize React + Vite project in `frontend/` directory
- Install `@simplewebauthn/browser` for WebAuthn client helpers
- Create registration page with form
- Create login page with form
- Create protected "secret" page (shows message only when logged in)
- Handle browser WebAuthn API calls (`navigator.credentials.create/get`)
- Display success/error states
- Add CORS configuration to backend

#### Priority 3: Polish & Production Readiness
- Add CORS headers in main.go (allow frontend origin)
- Improve error messages (more specific feedback)
- Add request logging
- Handle `CloneWarning` flag (log suspicious logins)
- Add rate limiting for registration/login endpoints
- Consider adding username validation
- Clean up expired challenges periodically

### Files Modified Today
- `backend/handlers/login.go` - Created with BeginLogin and FinishLogin
- `backend/models/user.go` - Added Credentials field to User struct
- `backend/main.go` - Added login routes

### Useful Commands
```bash
# Build backend
cd backend && go build -o server

# Run backend server
./server

# Test login begin (should fail - no user)
curl -X POST http://localhost:8080/login/begin \
  -H "Content-Type: application/json" \
  -d '{"username":"alice"}'

# Test registration begin
curl -X POST http://localhost:8080/register/begin \
  -H "Content-Type: application/json" \
  -d '{"username":"alice"}'
```

### Resources Referenced
- go-webauthn ValidateLogin: Used for signature verification and counter checking
- Go database/sql Rows: Understanding Query() vs QueryRow() and connection pooling
- defer statement: Ensuring resource cleanup with rows.Close()

---

## Session 2025-10-14: Backend Registration Flow

### What We Built Today

#### 1. Database Setup
- Created `init-2.sql` migration to add:
  - `transports` column to credentials table
  - `challenges` table for storing temporary challenges
  - Indexes for performance
- Database name: `passkeys_db`
- Connected via TablePlus for testing

#### 2. Backend Structure
Created complete Go backend structure:

```
backend/
├── db/
│   └── db.go           # PostgreSQL connection
├── models/
│   ├── user.go         # User struct + webauthn.User interface implementation
│   └── credential.go   # Credential struct
├── handlers/
│   ├── handler.go      # Handler struct with DB and WebAuthn dependencies
│   └── register.go     # Registration endpoints
└── main.go             # HTTP server setup
```

#### 3. Registration Flow Implementation

**BeginRegistration (`POST /register/begin`):**
1. Parse username from request body
2. Check if user exists (return error if exists)
3. Create new user in database
4. Call `webAuthn.BeginRegistration(user)` to generate challenge and options
5. Store challenge in challenges table with 5-minute expiration
6. Return credential creation options as JSON to client

**FinishRegistration (`POST /register/finish`):**
1. Parse credential creation response from request body using `protocol.ParseCredentialCreationResponseBody()`
2. Extract challenge from response
3. Query database to get user_id from challenge
4. Retrieve user from users table
5. Reconstruct `SessionData` with challenge and user ID
6. Call `webAuthn.CreateCredential(user, sessionData, parsedResponse)` to verify
7. Save credential to credentials table (id, user_id, public_key, sign_count)
8. Delete used challenge from challenges table
9. Return success

### Key Concepts Learned

#### Database Operations in Go
- **`QueryRow().Scan()`**: For SELECT queries and INSERT...RETURNING
  - Must scan into individual variables, not structs
  - Example: `.Scan(&user.ID, &user.Username, &user.CreatedAt)`
- **`Exec()`**: For INSERT/UPDATE/DELETE without RETURNING
  - Returns result with rows affected, not data
- **`Ping()`**: Actually tests database connection (sql.Open() is lazy)
- **PostgreSQL placeholders**: Use `$1`, `$2`, etc. (not `?` like MySQL)

#### WebAuthn Library (go-webauthn/webauthn v0.14)
- **User Interface**: Must implement 5 methods:
  - `WebAuthnID()`, `WebAuthnName()`, `WebAuthnDisplayName()`, `WebAuthnCredentials()`, `WebAuthnIcon()`
- **API naming**: Modern version uses `CreateCredential()` instead of `FinishRegistration()`
- **Challenge handling**: Challenge is base64url encoded in transit, stored as bytes in DB
- **SessionData**: Need to reconstruct from stored challenge for verification

#### Go Patterns
- **Method receivers**: `func (h *Handler) BeginRegistration(...)` - methods on structs
- **Dependency injection**: Pass DB and WebAuthn via Handler struct
- **Error handling**: Check errors immediately, return early
- **JSON marshaling**: Use `json.Marshal()` before setting headers for better error control

### Design Decisions & Trade-offs

#### User Creation Timing
**Current approach**: Create user in BeginRegistration (before passkey verified)

**Pros:**
- Simpler implementation
- User ID available immediately for foreign keys

**Cons:**
- Creates orphaned users if registration never completes
- Database pollution over time

**Alternative**: Create user in FinishRegistration (after passkey verified)
- See `docs/registration-flow-design.md` for full analysis
- Better for production, but more complex
- Would need to store username (not user_id) in challenges table

#### Challenge Storage
- Storing in database (not in-memory) for persistence
- 5-minute expiration
- Single-use (deleted after verification)
- Alternative: Could use Redis with TTL for production

### Questions Answered

1. **Why do we need Ping()?**
   - `sql.Open()` doesn't actually connect, just validates connection string
   - `Ping()` forces immediate connection attempt
   - Fail fast if database is down

2. **Why is Credential ID bytes, not UUID?**
   - Authenticator generates it, not us
   - Arbitrary binary data from hardware
   - Length varies by authenticator
   - User ID is UUID because we control it

3. **How to identify user in FinishRegistration?**
   - Extract challenge from credential response
   - Query challenges table to get user_id
   - No need to send username from client

4. **When to use QueryRow vs Exec?**
   - QueryRow: When expecting data back (SELECT, INSERT...RETURNING)
   - Exec: When just executing (INSERT, UPDATE, DELETE without RETURNING)

5. **Why not handle json.Encode() error?**
   - Headers already sent, can't call http.Error()
   - Better: Marshal first, check error, then set headers and write

### Testing Done
- ✅ Server starts successfully on http://localhost:8080
- ✅ Routes registered: `/register/begin` and `/register/finish`
- ⚠️ Not tested end-to-end yet (no frontend)

### Next Session Tasks

#### Priority 1: Login Flow
Create `handlers/login.go` with:

**BeginLogin:**
1. Parse username from request
2. Look up user in database
3. Get user's credentials from credentials table (needed for `allowCredentials`)
4. Call `webAuthn.BeginLogin(user)` to generate challenge
5. Store challenge in challenges table
6. Return assertion options to client

**FinishLogin:**
1. Parse authentication assertion from request body
2. Extract challenge, get user from database
3. Get user's credentials (for counter verification)
4. Call `webAuthn.ValidateLogin()` to verify signature
5. Update credential sign_count in database
6. Create session (cookie or JWT)
7. Return success with session token

**Key differences from registration:**
- Need to load existing credentials from DB
- Verify signature instead of creating new credential
- Must update sign counter (detect cloned credentials)
- Need session management

#### Priority 2: Session Management
- Add session storage (cookies or JWT tokens)
- Create middleware to check authentication
- Add logout endpoint
- Consider using `gorilla/sessions` library

#### Priority 3: Frontend
- Initialize React + Vite project
- Install `@simplewebauthn/browser`
- Create registration form
- Create login form
- Handle WebAuthn API calls
- Display success/error states

#### Priority 4: Testing & Polish
- Test full registration flow with real authenticator
- Test login flow
- Add CORS headers for frontend
- Error handling improvements
- Add logging

### Files to Review Next Session
- `backend/handlers/register.go` - Current implementation
- `backend/models/user.go` - webauthn.User interface
- `docs/registration-flow-design.md` - Design decisions doc

### Useful Commands
```bash
# Start database
docker-compose up -d

# Run backend server
cd backend && go run main.go

# Test registration begin
curl -X POST http://localhost:8080/register/begin \
  -H "Content-Type: application/json" \
  -d '{"username":"alice"}'

# View database
# Use TablePlus with: localhost:5432, user: postgres, pass: postgres, db: passkeys_db
```

### Resources Referenced
- go-webauthn library: https://github.com/go-webauthn/webauthn
- Go database/sql tutorial: https://go.dev/doc/tutorial/database-access
- lib/pq documentation: https://pkg.go.dev/github.com/lib/pq
- PostgreSQL BYTEA type: https://www.postgresql.org/docs/current/datatype-binary.html
