# Learn Passkeys - Complete Fullstack WebAuthn Implementation

A learning project that implements passwordless authentication using passkeys (WebAuthn) from scratch. This project demonstrates a complete fullstack application with registration, login, and protected routes—all without passwords.

**Live Demo:** Runs on localhost (see [Quick Start](#quick-start))
**Blog Post:** [Building My First Passkey Authentication System](./blog.md)
**Detailed Debugging Guide:** [Integration Debugging Guide](./docs/integration-debugging-guide.md)

---

## What Are Passkeys?

Passkeys are a modern replacement for passwords that use **public-key cryptography** instead of shared secrets. When you register on a website:

- Your device generates a unique key pair for that specific site
- The **private key** never leaves your device (stored securely in Touch ID, Face ID, etc.)
- The **public key** is stored on the server

To log in, you just prove you have the private key by signing a challenge. No password to remember, no password to leak.

**Why Passkeys Are Better:**
- 🔒 **Phishing-proof:** Cryptographically bound to your domain
- 💥 **Breach-proof:** Servers only store public keys (useless to attackers)
- ⚡ **Faster:** ~2 seconds vs ~10-15 seconds for passwords
- 🎯 **Simpler:** Just Touch ID/Face ID, no typing

---

## Features

✅ **Complete Authentication System**
- User registration with passkey creation
- Login with passkey authentication
- Protected routes (requires authentication)
- Session management (localStorage-based for simplicity)

✅ **Security Features**
- Challenge-based authentication
- Sign counter tracking (detects cloned credentials)
- Backup flag validation (detects credential sync changes)
- Origin verification (prevents phishing)
- Single-use challenges with expiration

✅ **Production-Ready Patterns**
- Complete SessionData storage (no reconstruction)
- Full credential state persistence (flags, AAGUID, attestation type)
- CORS configured for frontend-backend communication
- Proper error handling and JSON responses

---

## Tech Stack

### Backend
- **Language:** Go 1.21+
- **WebAuthn Library:** [`go-webauthn/webauthn`](https://github.com/go-webauthn/webauthn) v0.14.0
- **Database:** PostgreSQL 15 (running in Docker)
- **Database Driver:** `lib/pq`

### Frontend
- **Framework:** React 19 with TypeScript
- **Build Tool:** Vite 6
- **WebAuthn Library:** [`@simplewebauthn/browser`](https://simplewebauthn.dev/) v13.2.2
- **UI:** Tailwind CSS + shadcn/ui components
- **HTTP Client:** Fetch API with TanStack Query

### Infrastructure
- **Containerization:** Docker Compose (PostgreSQL)
- **Development:** Localhost only (not production-ready)

---

## Prerequisites

Before you begin, ensure you have:

- **Go** 1.21 or higher ([download](https://go.dev/dl/))
- **Node.js** 18+ and npm ([download](https://nodejs.org/))
- **Docker** and Docker Compose ([download](https://www.docker.com/))
- **A WebAuthn-compatible device:**
  - macOS with Touch ID
  - Windows 10+ with Windows Hello
  - iPhone/iPad with iOS 16+
  - Android 9+ with fingerprint/face unlock
  - Or a FIDO2 security key (YubiKey, etc.)

---

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/learn-passkeys.git
cd learn-passkeys
```

### 2. Start PostgreSQL Database

```bash
# Start PostgreSQL in Docker
docker-compose up -d

# Verify it's running
docker-compose ps

# Apply database migrations
cat init.sql | docker exec -i postgres_db psql -U postgres -d passkeys_db
cat init-2.sql | docker exec -i postgres_db psql -U postgres -d passkeys_db
cat init-3.sql | docker exec -i postgres_db psql -U postgres -d passkeys_db
```

### 3. Start the Backend

```bash
cd backend

# Install Go dependencies
go mod download

# Run the server
go run main.go

# Or build and run
go build -o server
./server
```

The backend will start on `http://localhost:8080`

### 4. Start the Frontend

```bash
cd frontend

# Install dependencies (first time only)
npm install

# Start the dev server
npm run dev
```

The frontend will start on `http://localhost:5173`

### 5. Try It Out!

1. **Register:** Visit http://localhost:5173/register
   - Enter a username
   - Click "Register with Passkey"
   - Touch ID/Face ID prompt will appear
   - Authenticate → You're registered!

2. **Login:** Visit http://localhost:5173/login
   - Enter the same username
   - Click "Login with Passkey"
   - Authenticate → You're logged in!

3. **View Secret:** Visit http://localhost:5173/secret
   - You'll see the protected message (only visible when authenticated)

---

## Project Structure

```
learn-passkeys/
├── backend/                 # Go backend server
│   ├── main.go             # Entry point, HTTP server setup, CORS
│   ├── db/
│   │   └── db.go           # PostgreSQL connection
│   ├── handlers/
│   │   ├── handler.go      # Handler struct with dependencies
│   │   ├── register.go     # POST /register/begin, /register/finish
│   │   └── login.go        # POST /login/begin, /login/finish
│   ├── models/
│   │   ├── user.go         # User model (implements webauthn.User)
│   │   └── credential.go   # Credential model
│   ├── go.mod              # Go dependencies
│   └── go.sum
│
├── frontend/               # React frontend
│   ├── src/
│   │   ├── main.tsx        # Entry point
│   │   ├── App.tsx         # Main app with routing
│   │   ├── components/
│   │   │   ├── Register.tsx    # Registration page
│   │   │   ├── Login.tsx       # Login page
│   │   │   └── Secret.tsx      # Protected page
│   │   └── lib/
│   │       └── webauthn.ts     # WebAuthn API helpers
│   ├── package.json        # Frontend dependencies
│   └── vite.config.ts      # Vite configuration
│
├── docs/                   # Documentation
│   ├── integration-debugging-guide.md  # Complete debugging guide
│   ├── session-notes.md                # Development journal
│   └── registration-flow-design.md     # Design decisions
│
├── init.sql                # Database schema (users, credentials)
├── init-2.sql              # Challenges table
├── init-3.sql              # Credential flags migration
├── docker-compose.yml      # PostgreSQL setup
├── blog.md                 # Learning journey blog post
├── next-steps.md           # Learning roadmap
├── CLAUDE.md               # Project guidance for AI assistants
└── README.md               # This file
```

---

## How It Works

### Registration Flow

```
┌─────────┐                ┌─────────┐                ┌──────────┐
│ Browser │                │ Backend │                │ Database │
└────┬────┘                └────┬────┘                └────┬─────┘
     │                          │                          │
     │ POST /register/begin     │                          │
     │ {username: "alice"}      │                          │
     ├─────────────────────────>│                          │
     │                          │                          │
     │                          │ INSERT INTO users        │
     │                          ├─────────────────────────>│
     │                          │                          │
     │                          │ Generate challenge       │
     │                          │ (WebAuthn library)       │
     │                          │                          │
     │                          │ Store SessionData        │
     │                          ├─────────────────────────>│
     │                          │                          │
     │ Options (challenge, RP)  │                          │
     │<─────────────────────────┤                          │
     │                          │                          │
     │ navigator.credentials.   │                          │
     │   create()               │                          │
     │ (Browser WebAuthn API)   │                          │
     │                          │                          │
     │ Touch ID/Face ID prompt  │                          │
     │ User authenticates       │                          │
     │                          │                          │
     │ POST /register/finish    │                          │
     │ {credential: {...}}      │                          │
     ├─────────────────────────>│                          │
     │                          │                          │
     │                          │ Retrieve SessionData     │
     │                          │<─────────────────────────┤
     │                          │                          │
     │                          │ Verify credential        │
     │                          │ (WebAuthn library)       │
     │                          │                          │
     │                          │ Store public key + flags │
     │                          ├─────────────────────────>│
     │                          │                          │
     │ {status: "success"}      │                          │
     │<─────────────────────────┤                          │
     │                          │                          │
```

**Key Points:**
- User creation happens in `BeginRegistration` (before credential verification)
- Complete `SessionData` is stored as JSON (not reconstructed)
- Credential flags (`backup_eligible`, `backup_state`, `aaguid`) are saved

### Login Flow

```
┌─────────┐                ┌─────────┐                ┌──────────┐
│ Browser │                │ Backend │                │ Database │
└────┬────┘                └────┬────┘                └────┬─────┘
     │                          │                          │
     │ POST /login/begin        │                          │
     │ {username: "alice"}      │                          │
     ├─────────────────────────>│                          │
     │                          │                          │
     │                          │ SELECT user              │
     │                          │<─────────────────────────┤
     │                          │                          │
     │                          │ SELECT credentials       │
     │                          │ (with flags!)            │
     │                          │<─────────────────────────┤
     │                          │                          │
     │                          │ Generate challenge       │
     │                          │ (WebAuthn library)       │
     │                          │                          │
     │                          │ Store SessionData        │
     │                          ├─────────────────────────>│
     │                          │                          │
     │ Assertion options        │                          │
     │ (challenge, allowed IDs) │                          │
     │<─────────────────────────┤                          │
     │                          │                          │
     │ navigator.credentials.   │                          │
     │   get()                  │                          │
     │                          │                          │
     │ Touch ID/Face ID prompt  │                          │
     │ User authenticates       │                          │
     │ Private key signs        │                          │
     │ challenge                │                          │
     │                          │                          │
     │ POST /login/finish       │                          │
     │ {assertion: {...}}       │                          │
     ├─────────────────────────>│                          │
     │                          │                          │
     │                          │ Retrieve SessionData     │
     │                          │<─────────────────────────┤
     │                          │                          │
     │                          │ Verify signature         │
     │                          │ Check backup flags       │
     │                          │ Validate sign counter    │
     │                          │ (WebAuthn library)       │
     │                          │                          │
     │                          │ Update sign_count        │
     │                          ├─────────────────────────>│
     │                          │                          │
     │ {status: "success",      │                          │
     │  username: "alice"}      │                          │
     │<─────────────────────────┤                          │
     │                          │                          │
     │ Store in localStorage    │                          │
     │ Redirect to /secret      │                          │
     │                          │                          │
```

**Key Points:**
- Credentials loaded with ALL fields (public key, sign count, flags, AAGUID)
- Signature verified with public key
- Sign counter increments (detects cloned credentials)
- Backup flags validated (detects credential sync changes)

---

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Credentials Table
```sql
CREATE TABLE credentials (
    id BYTEA PRIMARY KEY,                    -- Credential ID (raw bytes)
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key BYTEA NOT NULL,               -- Public key for verification
    sign_count INTEGER NOT NULL DEFAULT 0,   -- Increments with each use
    transports TEXT[],                       -- How credential was created
    backup_eligible BOOLEAN DEFAULT false,   -- Can be backed up to cloud
    backup_state BOOLEAN DEFAULT false,      -- Is currently backed up
    attestation_type VARCHAR(50) DEFAULT '', -- Attestation format
    aaguid BYTEA,                            -- Authenticator GUID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Challenges Table
```sql
CREATE TABLE challenges (
    challenge BYTEA PRIMARY KEY,             -- Complete SessionData (JSON)
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,               -- 'registration' or 'login'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL            -- 5 minutes from creation
);
```

**Important:** The `challenge` column stores the complete `SessionData` object as JSON, not just the challenge bytes!

---

## API Endpoints

### Registration

**POST /register/begin**
```json
Request:
{
  "username": "alice"
}

Response:
{
  "publicKey": {
    "challenge": "...",
    "rp": { "name": "Learn Passkeys", "id": "localhost" },
    "user": { "id": "...", "name": "alice", "displayName": "alice" },
    "pubKeyCredParams": [...],
    "timeout": 60000,
    "attestation": "none"
  }
}
```

**POST /register/finish**
```json
Request:
{
  "id": "...",
  "rawId": "...",
  "response": {
    "attestationObject": "...",
    "clientDataJSON": "..."
  },
  "type": "public-key"
}

Response:
{
  "status": "success",
  "message": "Registration successful"
}
```

### Login

**POST /login/begin**
```json
Request:
{
  "username": "alice"
}

Response:
{
  "publicKey": {
    "challenge": "...",
    "timeout": 60000,
    "rpId": "localhost",
    "allowCredentials": [
      { "id": "...", "type": "public-key", "transports": ["internal", "hybrid"] }
    ],
    "userVerification": "preferred"
  }
}
```

**POST /login/finish**
```json
Request:
{
  "id": "...",
  "rawId": "...",
  "response": {
    "authenticatorData": "...",
    "clientDataJSON": "...",
    "signature": "...",
    "userHandle": "..."
  },
  "type": "public-key"
}

Response:
{
  "status": "success",
  "username": "alice"
}
```

---

## Key Learnings & Debugging

This project documents real integration challenges and their solutions:

### Problem 1: "Invalid Attestation Format" Error

**Symptom:** Registration failed even though attestation format was correct

**Root Cause:** Not storing complete `SessionData` from `BeginRegistration`

**Solution:** Store the entire object as JSON, don't reconstruct it
```go
// WRONG
sessionData := webauthn.SessionData{
    Challenge: challenge,
    UserID:    user.ID[:],
}

// RIGHT
sessionDataJSON, _ := json.Marshal(sessionData)  // Store everything!
```

📖 **Full debugging story:** [Integration Debugging Guide](./docs/integration-debugging-guide.md#problem-1-invalid-attestation-format-error)

### Problem 2: "Backup Eligible Flag Inconsistency" Error

**Symptom:** Login failed with backup flag error

**Root Cause:** Not storing credential flags (`backup_eligible`, `backup_state`, `aaguid`)

**Solution:** Store complete credential state including flags
```sql
ALTER TABLE credentials
ADD COLUMN backup_eligible BOOLEAN,
ADD COLUMN backup_state BOOLEAN,
ADD COLUMN aaguid BYTEA;
```

📖 **Full debugging story:** [Integration Debugging Guide](./docs/integration-debugging-guide.md#problem-2-backup-eligible-flag-inconsistency)

---

## Development

### Useful Commands

**Database:**
```bash
# Start PostgreSQL
docker-compose up -d

# Stop PostgreSQL
docker-compose down

# View logs
docker-compose logs -f postgres_db

# Connect to database
docker exec -it postgres_db psql -U postgres -d passkeys_db

# Run queries
docker exec -i postgres_db psql -U postgres -d passkeys_db -c "SELECT * FROM users;"
docker exec -i postgres_db psql -U postgres -d passkeys_db -c "SELECT * FROM credentials;"

# Clean test data
docker exec -i postgres_db psql -U postgres -d passkeys_db -c "DELETE FROM credentials; DELETE FROM challenges; DELETE FROM users;"
```

**Backend:**
```bash
cd backend

# Run with auto-reload (install air first: go install github.com/cosmtrek/air@latest)
air

# Build
go build -o server

# Run tests
go test ./...

# Format code
go fmt ./...
```

**Frontend:**
```bash
cd frontend

# Development
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Lint
npm run lint
```

### Environment Variables

**Backend:**
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/passkeys_db?sslmode=disable"
export PORT="8080"
export RP_ID="localhost"
export RP_ORIGINS="http://localhost:5173"
```

**Frontend:**
```bash
export VITE_API_URL="http://localhost:8080"
```

---

## Testing Across Devices

### macOS (Touch ID)
✅ Works perfectly in Chrome, Safari, Firefox

### iOS (Face ID / Touch ID)
✅ Works in Safari (requires iOS 16+)
⚠️ Chrome on iOS uses Safari's engine

### Windows (Windows Hello)
✅ Works in Chrome, Edge (requires Windows 10+)

### Android (Fingerprint / Face Unlock)
✅ Works in Chrome (requires Android 9+)

### Hardware Security Keys
✅ YubiKey, Titan Security Key work on all platforms

---

## Security Considerations

### What's Implemented

✅ **Challenge uniqueness:** Random challenges, single-use, 5-minute expiration
✅ **Origin verification:** Credentials bound to `http://localhost:5173`
✅ **Sign counter tracking:** Detects cloned credentials
✅ **Backup flag validation:** Detects credential sync changes
✅ **HTTPS enforcement:** WebAuthn only works on HTTPS (localhost exempt)

### What's Missing (For Production)

⚠️ **HTTPS:** This runs on HTTP localhost (production needs HTTPS)
⚠️ **Rate limiting:** No protection against brute force attacks
⚠️ **CSRF protection:** Should add CSRF tokens
⚠️ **Session security:** Using localStorage (should use secure HTTP-only cookies)
⚠️ **Input validation:** Minimal validation on username, etc.
⚠️ **Error messages:** Too detailed (information leakage)

---

## Resources

### Documentation
- **[WebAuthn Guide](https://webauthn.guide/)** - Visual guide to WebAuthn concepts
- **[WebAuthn Spec](https://www.w3.org/TR/webauthn-3/)** - Official W3C specification
- **[webauthn.io](https://webauthn.io/)** - Interactive demo

### Libraries Used
- **[go-webauthn](https://github.com/go-webauthn/webauthn)** - Go WebAuthn library
- **[SimpleWebAuthn](https://simplewebauthn.dev/)** - TypeScript WebAuthn library
- **[shadcn/ui](https://ui.shadcn.com/)** - React UI components

### Learning Materials
- **[This Project's Blog Post](./blog.md)** - Complete learning journey
- **[Integration Debugging Guide](./docs/integration-debugging-guide.md)** - Real debugging stories
- **[Session Notes](./docs/session-notes.md)** - Development journal

---

## FAQ

### Q: Can I use this in production?

**A:** This is a learning project, not production-ready. You'd need:
- HTTPS
- Secure session management (HTTP-only cookies, not localStorage)
- Rate limiting
- CSRF protection
- Input validation and sanitization
- Proper error handling (don't leak information)
- Account recovery flow
- Monitoring and logging
- Testing across browsers and devices

### Q: Does this work without Touch ID / Face ID?

**A:** Yes! It works with:
- Platform authenticators (Touch ID, Face ID, Windows Hello)
- Hardware security keys (YubiKey, Titan Key)
- Chrome's password manager (syncs passkeys)

### Q: What happens if I lose my device?

**A:**
- If your passkey is synced (iCloud Keychain, Google Password Manager), you can access it on other devices
- If it's device-only, you'd need account recovery (which this demo doesn't implement)
- Production apps should implement backup codes or email recovery

### Q: Can I have multiple passkeys?

**A:** This demo only stores one passkey per user, but the database schema supports multiple. You'd need to add UI for credential management.

### Q: Why Go and React?

**A:**
- Go has excellent WebAuthn libraries and strong cryptography support
- React is popular and SimpleWebAuthn makes WebAuthn API calls simple
- Both are well-documented and have active communities

---

## Contributing

This is a learning project, but suggestions are welcome!

**Ideas for contributions:**
- Add session management with secure cookies
- Implement account recovery flow
- Add credential management UI (view/delete passkeys)
- Support for discoverable credentials (usernameless login)
- Conditional UI (autofill suggestions)
- Integration tests with real authenticators
- Support for more authenticator types

**To contribute:**
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

MIT License - see [LICENSE](LICENSE) file for details

**TL;DR:** Use this code however you want, but don't blame me if something breaks! 😅

---

## Acknowledgments

- **[go-webauthn](https://github.com/go-webauthn/webauthn)** team for an excellent Go library
- **[SimpleWebAuthn](https://simplewebauthn.dev/)** by Matthew Miller for making WebAuthn accessible
- **[webauthn.guide](https://webauthn.guide/)** for the best visual explanation of WebAuthn
- The WebAuthn community for being helpful and welcoming

---

## Contact

**Questions? Found a bug? Have suggestions?**

- Open an [issue](https://github.com/yourusername/learn-passkeys/issues)
- Read the [blog post](./blog.md) for the full learning journey
- Check the [debugging guide](./docs/integration-debugging-guide.md) for common issues

---

**Built with ❤️ as a learning project**

*"The best way to learn is by building something real—and debugging it." - Every developer ever*
