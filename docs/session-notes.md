# Development Session Notes

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
