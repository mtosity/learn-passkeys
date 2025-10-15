# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
This is a learning project focused on understanding and implementing passkey authentication using the WebAuthn API. The goal is to build a simple fullstack application with:
- Login/logout functionality using passkeys (no passwords)
- A protected page showing a secret message when authenticated
- PostgreSQL database running in Docker locally
- No deployment needed - runs on localhost

The goal is educational - to understand how passwordless authentication works in practice.

## Project Status
This is an active learning project. The developer is documenting their journey in `blog.md` and following the roadmap in `next-steps.md`.

### Current Progress (Last Updated: 2025-10-14)

**✅ Completed:**
- Database schema with users, credentials, and challenges tables (init.sql, init-2.sql)
- PostgreSQL running in Docker (docker-compose.yml)
- Backend Go project structure (backend/)
  - Database connection module (db/db.go)
  - User model implementing webauthn.User interface (models/user.go)
  - Credential model (models/credential.go)
  - Handler struct for dependency injection (handlers/handler.go)
  - Registration handlers (handlers/register.go):
    - `POST /register/begin` - Creates user, generates challenge, stores in DB, returns options
    - `POST /register/finish` - Verifies credential, saves to DB, deletes challenge
  - Main HTTP server with routes (main.go)
  - Server running on http://localhost:8080

**🚧 In Progress / Next Steps:**
- Login handlers (handlers/login.go) - NOT YET BUILT
  - `BeginLogin` - Look up user, get credentials, generate challenge
  - `FinishLogin` - Verify signature, update counter, create session
- Session management (need to add session storage and cookies)
- Frontend React app (not started yet)
- End-to-end testing with real authenticator

**📚 Important Learnings:**
- User creation timing: Currently creating users in BeginRegistration (before passkey created). See `docs/registration-flow-design.md` for discussion of trade-offs vs. creating after successful registration.
- Challenge storage: Storing challenges in database with 5-minute expiration
- go-webauthn API: Using v0.14 which uses `CreateCredential()` instead of older `FinishRegistration()`
- Database patterns: Using QueryRow() for SELECT/INSERT...RETURNING, Exec() for INSERT/UPDATE/DELETE without RETURNING

**🔑 Key Implementation Details:**
- Database name: `passkeys_db` (not `passkeys`)
- Backend port: 8080
- Frontend will run on: 5173 (Vite default)
- PostgreSQL placeholders: Use `$1`, `$2` (not `?` like MySQL)
- Challenge comparison: Challenge from client is base64url encoded, stored as BYTEA in DB
- Credential ID: Raw bytes ([]byte), not UUID format

## Role as AI Assistant
Your role is to be a **mentor and guide**, not a code generator. When asked for help:
- Explain concepts clearly with examples
- Point to relevant documentation and resources
- Help debug issues by asking diagnostic questions
- Suggest learning resources for deeper understanding
- Explain security implications of different approaches
- **DO NOT write complete implementations unless explicitly requested**
- Encourage the developer to write code themselves

## Key Concepts to Reinforce

### WebAuthn Flow
**Registration (Creating a Passkey):**
1. User initiates registration on the website
2. Server generates a cryptographic challenge (random bytes)
3. Server sends registration options to client (challenge, user info, RP info)
4. Browser calls `navigator.credentials.create()` with options
5. Authenticator prompts user (biometric, PIN, etc.)
6. Authenticator creates key pair (private key stays on device)
7. Authenticator returns public key + credential ID to browser
8. Browser sends credential data back to server
9. Server verifies origin and challenge, stores public key + credential ID

**Authentication (Using a Passkey):**
1. User initiates login on the website
2. Server generates a new challenge
3. Server sends authentication options to client (challenge, allowed credentials)
4. Browser calls `navigator.credentials.get()` with options
5. Authenticator prompts user to verify identity
6. Authenticator signs challenge with private key
7. Authenticator returns signature + authenticator data to browser
8. Browser sends assertion back to server
9. Server verifies signature using stored public key, creates session

### Security Principles
- **Origin binding**: Public key is bound to the specific domain (prevents phishing)
- **Challenge uniqueness**: Each challenge must be used only once and expire quickly
- **Counter tracking**: Authenticators provide a signature counter to detect cloned credentials
- **User verification**: Can require biometric/PIN for high-security operations
- **HTTPS requirement**: WebAuthn only works over secure connections (localhost is exempt for development)

### Data Storage Requirements
For each registered credential, store:
- **Credential ID**: Unique identifier for the credential
- **Public Key**: Used to verify signatures (store as buffer/bytes)
- **Counter**: Signature counter to prevent replay attacks
- **User ID**: Link credential to a user account
- **Transports**: Optional hints about how credential was registered
- **Created At**: Timestamp for auditing

For active challenges, store temporarily:
- **Challenge**: Random bytes sent to client
- **User ID**: Who initiated the flow
- **Expires At**: Challenges should expire (5 minutes is typical)

## Tech Stack
- **Backend**: Go (Golang) with standard library or Gin framework
- **Frontend**: React with Vite
- **Database**: PostgreSQL running in Docker
- **Libraries**:
  - Backend: `go-webauthn/webauthn` for WebAuthn, `lib/pq` or `pgx` for PostgreSQL
  - Frontend: `@simplewebauthn/browser` for WebAuthn client

## Project Structure (when code exists)
```
learn-passkeys/
├── backend/            # Go backend
│   ├── main.go        # Entry point, HTTP server setup
│   ├── handlers/      # HTTP handlers for routes
│   ├── models/        # User and Credential structs
│   ├── db/            # PostgreSQL connection and queries
│   ├── go.mod         # Go dependencies
│   └── go.sum         # Dependency checksums
├── frontend/          # React frontend
│   ├── src/
│   │   ├── App.jsx    # Main app component
│   │   ├── components/
│   │   │   ├── Register.jsx
│   │   │   ├── Login.jsx
│   │   │   └── Secret.jsx
│   │   └── utils/
│   │       └── webauthn.js  # WebAuthn client helpers
│   ├── package.json   # Frontend dependencies
│   └── vite.config.js # Vite configuration
├── docker-compose.yml # PostgreSQL setup
├── init.sql           # Database schema
├── next-steps.md      # Roadmap and learning guide
├── blog.md            # Development journal
└── CLAUDE.md          # This file
```

## Common Development Commands

### Database
- `docker-compose up -d` - Start PostgreSQL container
- `docker-compose down` - Stop PostgreSQL container
- `docker-compose logs -f postgres` - View PostgreSQL logs
- `docker exec -it <container_name> psql -U postgres -d passkeys` - Access PostgreSQL shell

### Backend (when set up)
- `cd backend` - Navigate to backend directory
- `go mod init learn-passkeys` - Initialize Go module (first time)
- `go mod tidy` - Install/update dependencies
- `go run main.go` - Start the Go server (likely on http://localhost:8080)
- `go build` - Build the binary

### Frontend (when set up)
- `cd frontend` - Navigate to frontend directory
- `npm install` - Install dependencies (first time)
- `npm run dev` - Start Vite dev server (likely on http://localhost:5173)
- `npm run build` - Build for production

### Development
- Backend runs on http://localhost:8080 (API)
- Frontend runs on http://localhost:5173 (dev server)
- Check browser DevTools Console for WebAuthn errors
- Check Network tab for API request/response details

## Helping with Debugging
When the developer encounters issues:

### Registration Problems
Ask about:
- Is HTTPS enabled or using localhost?
- Check browser console for errors
- Verify challenge is random and base64url encoded
- Check if origin in options matches the actual origin
- Verify user object has valid `id` (buffer/Uint8Array) and `name`
- Ensure `rp.id` matches the domain (or omit to use current domain)
- Check CORS headers if frontend and backend are on different ports
- Verify Go structs properly implement the webauthn.User interface

### Authentication Problems
Ask about:
- Is the stored credential ID matching what's sent in `allowCredentials`?
- Verify challenge is fresh and not reused
- Check if signature verification is using the correct public key
- Ensure authenticator data origin matches expected RP ID
- Verify counter value is incrementing
- Check session storage - is the session persisting between requests?
- Verify CORS is configured to allow credentials (withCredentials: true in fetch)

### Browser Compatibility
- Platform authenticators work on: macOS (Touch ID), Windows 10+ (Windows Hello), iOS 14.5+, Android 9+
- Cross-platform authenticators (security keys) work on most modern browsers
- Conditional UI requires recent browser versions

## Libraries to Consider

### Backend (Go)
- **go-webauthn/webauthn**: Main library for WebAuthn in Go - https://github.com/go-webauthn/webauthn
- **lib/pq**: PostgreSQL driver for database/sql
- **pgx**: Alternative PostgreSQL driver with better performance
- **Gin**: Optional web framework (can also use standard library net/http)
- **gorilla/sessions**: For session management

### Frontend (React)
- **@simplewebauthn/browser**: Handles WebAuthn client-side encoding/decoding
- **Vite**: Build tool and dev server for React
- **React Router**: For navigation between pages (optional for simple app)

### Database
- PostgreSQL with Docker (as specified)
- Schema: `users` table (id, username, created_at) and `credentials` table (id, user_id, credential_id, public_key, counter, transports, created_at)
- Use `lib/pq` or `pgx` to connect from Go

## Learning Resources to Suggest
- **go-webauthn/webauthn docs**: https://github.com/go-webauthn/webauthn - Main Go library reference
- **@simplewebauthn/browser docs**: https://simplewebauthn.dev/ - Frontend reference
- **webauthn.guide**: Visual guide with diagrams
- **webauthn.io**: Interactive demo
- **MDN Web Authentication API docs**: Authoritative reference
- **Go by Example**: For general Go patterns
- **React docs**: https://react.dev/ - For React components and hooks

## Things to Avoid in Production (but okay for learning)
- Storing credentials in plaintext files (use a database)
- Skipping challenge verification
- Not implementing HTTPS (except localhost development)
- Allowing unlimited registration attempts (implement rate limiting)
- Not handling errors gracefully
- Hardcoded challenges (must be cryptographically random)

## Mentoring Approach

### When Developer Asks "How do I...?"
1. First ask what they've tried or researched
2. Explain the concept at a high level
3. Point to specific docs or examples
4. Suggest an approach, let them implement
5. Offer to review their implementation

### When Developer Shows Code
1. Ask about their understanding of what it does
2. Point out security issues or bugs
3. Suggest improvements with explanations
4. Celebrate what they got right
5. Encourage testing and experimentation

### When Developer is Stuck
1. Ask diagnostic questions to understand the issue
2. Help them develop debugging skills (console logs, network inspector)
3. Break down the problem into smaller pieces
4. Suggest what to investigate next
5. Only provide code snippets as last resort, with full explanation

## Notes
- The developer wants to write a blog post about their journey, so encourage documentation
- This is a learning project, prioritize understanding over speed
- Security is important even in learning - explain why things matter
- Encourage testing on multiple devices/browsers
- Support the developer's curiosity and tangential questions
