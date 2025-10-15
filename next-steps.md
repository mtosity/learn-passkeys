# Next Steps for Learning Passkeys

## What are Passkeys?
Passkeys are a modern authentication method that replaces passwords with cryptographic key pairs. They use WebAuthn (Web Authentication API) to provide passwordless authentication that's more secure and user-friendly.

## Prerequisites
Before you start, ensure you have:
- Basic understanding of JavaScript/TypeScript
- Node.js installed (v18+ recommended)
- A code editor (VS Code recommended)
- A browser that supports WebAuthn (Chrome, Firefox, Safari, Edge)
- Basic understanding of async/await and promises

## Learning Path

### Step 1: Understand the Fundamentals
- Read the WebAuthn specification overview at https://webauthn.guide/
- Understand the concepts:
  - **Relying Party (RP)**: Your web application
  - **Authenticator**: Device/platform that creates and stores passkeys (fingerprint sensor, Face ID, security key)
  - **Public Key Cryptography**: Passkeys use asymmetric encryption (public/private key pairs)
  - **Challenge-Response**: Server sends challenge, client signs it with private key
  - **Attestation**: Optional verification that authenticator is legitimate
  - **Assertion**: Proving you own the private key during authentication

### Step 2: Set Up Your Development Environment
Create a simple project structure:
```
learn-passkeys/
├── server/           # Backend API
├── client/           # Frontend UI
├── docs/             # Your notes and documentation
└── examples/         # Working examples you build
```

### Step 3: Set Up Your Tech Stack
**Tech stack for this project:**
- **Backend**: Go (Golang) with `go-webauthn/webauthn` library
- **Frontend**: React with Vite + `@simplewebauthn/browser`
- **Database**: PostgreSQL running in Docker

**Initial setup steps:**
1. Create `docker-compose.yml` for PostgreSQL
2. Create `init.sql` with database schema (users and credentials tables)
3. Create `backend/` directory and initialize Go module (`go mod init`)
4. Create `frontend/` directory with Vite React template (`npm create vite@latest frontend -- --template react`)
5. Install Go dependencies: `go-webauthn/webauthn`, `lib/pq` or `pgx`, `gorilla/sessions`
6. Install frontend dependencies: `@simplewebauthn/browser`

### Step 4: Build Your First Passkey Registration Flow
Create a minimal example that:
1. Generates registration options on the server
2. Sends them to the client
3. Calls `navigator.credentials.create()` in the browser
4. Returns the credential to the server
5. Verifies and stores the credential

**Key concepts to implement:**
- User identification (username/email)
- Challenge generation (cryptographically random)
- Credential storage (public key, credential ID, counter)
- Origin verification (prevent phishing)

### Step 5: Build Your Authentication Flow
Create the login functionality:
1. Generate authentication options on the server
2. Send them to the client
3. Call `navigator.credentials.get()` in the browser
4. Return the assertion to the server
5. Verify the signature and authenticate the user

**Key concepts to implement:**
- Challenge verification (must match what was sent)
- Signature verification (using stored public key)
- Counter verification (prevent replay attacks)
- User session management

### Step 6: Test Different Scenarios
- Test on different devices (laptop, phone, tablet)
- Try different authenticators:
  - Platform authenticators (Touch ID, Face ID, Windows Hello)
  - Cross-platform authenticators (security keys like YubiKey)
  - Passkeys synced via iCloud/Google Password Manager
- Test edge cases:
  - User cancels the prompt
  - User has no authenticator available
  - Network failures during registration/authentication

### Step 7: Implement Advanced Features
Once basics work, explore:
- **Conditional UI**: Autofill passkey suggestions
- **Multi-device support**: Allow users to register multiple passkeys
- **Resident keys (discoverable credentials)**: Login without username
- **User verification requirements**: Require PIN/biometric
- **Attestation**: Verify authenticator model/manufacturer
- **Recovery flows**: What happens if user loses their device?

### Step 8: Security Considerations
Study and implement:
- HTTPS requirement (WebAuthn only works over secure connections)
- Origin validation (prevent phishing attacks)
- CORS configuration (if frontend and backend are separate)
- Rate limiting (prevent brute force attempts)
- Challenge expiration (challenges should be single-use and time-limited)
- User handle privacy (don't expose PII in user handles)

### Step 9: Learn from Real Implementations
Study how major platforms implement passkeys:
- GitHub's passkey implementation
- Google's passkey rollout
- Apple's passkey documentation
- Microsoft's passwordless authentication

### Step 10: Document Your Journey
As you learn, document:
- Challenges you faced and how you solved them
- Code snippets that helped you understand concepts
- Diagrams of the authentication flow
- Comparisons with traditional password authentication
- Performance measurements
- User experience observations

## Recommended Resources
- **go-webauthn/webauthn**: https://github.com/go-webauthn/webauthn - Your main backend reference
- **@simplewebauthn/browser**: https://simplewebauthn.dev/ - Your main frontend reference
- **webauthn.guide**: Visual guide to understand the flow
- **Go by Example**: https://gobyexample.com/ - For Go language patterns
- **React docs**: https://react.dev/ - For building React components
- **PostgreSQL Docker Hub**: https://hub.docker.com/_/postgres - For Docker setup
- **webauthn.io**: Interactive demo to see how it works
- **MDN Web Docs**: Web Authentication API reference

## Common Pitfalls to Avoid
- Not using HTTPS (required for WebAuthn)
- Hardcoding challenge values (must be cryptographically random)
- Not verifying the origin (security risk)
- Ignoring the authenticator data counter (replay attack risk)
- Storing credentials insecurely
- Not handling user cancellation gracefully
- Assuming all users have compatible devices

## Project Milestones
**Milestone 1**: Set up PostgreSQL with Docker and create database schema
**Milestone 2**: Set up Express server with basic routes
**Milestone 3**: Successfully register a passkey and see it stored in PostgreSQL
**Milestone 4**: Authenticate using that passkey and create a session
**Milestone 5**: Protect the secret message page - only show when logged in
**Milestone 6**: Implement logout functionality
**Milestone 7**: Test on different browsers and devices (optional)
**Milestone 8**: Write comprehensive blog post about your journey

## Questions to Explore
- How do passkeys compare to passwords in terms of security?
- What happens to passkeys when you switch devices?
- How do passkey managers work (iCloud Keychain, Google Password Manager)?
- Can passkeys be phished?
- What's the user experience like for non-technical users?
- How do you handle account recovery if someone loses all their passkeys?
- What's the browser support like?
- How do you migrate existing users from passwords to passkeys?
