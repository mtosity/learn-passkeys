# Development Session Notes

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
