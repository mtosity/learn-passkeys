# Building My First Passkey Authentication System: A Developer's Journey

**Published:** October 19, 2025
**Reading Time:** 15 minutes
**Tech Stack:** React, Go, PostgreSQL, WebAuthn

---

## Why I Decided to Learn Passkeys

Passwords are everywhere, and they're terrible. We all know it. I've built password authentication systems before—salt, hash, bcrypt rounds, password reset flows, rate limiting, breach detection. It works, but it's a constant battle against human nature and attackers.

Then I started seeing "Sign in with a passkey" appearing on major websites. Google, GitHub, PayPal—everyone's rolling them out. I realized this wasn't just another authentication fad. This was the future.

But reading about passkeys and actually *building* a passkey authentication system are two very different things. So I decided to dive deep: build a complete fullstack application from scratch using WebAuthn, document every mistake, and truly understand how passwordless authentication works.

This is that story.

---

## What I Learned Passkeys Actually Are

Before building anything, I needed to understand what passkeys really are beyond the marketing buzzwords.

**Here's my understanding after building this project:**

Passkeys are **public-private key pairs** stored on your device (or synced to your cloud account) that replace passwords entirely. When you register on a website:

1. Your device generates a unique key pair for that specific website
2. The **private key** never leaves your device (seriously, never)
3. The **public key** goes to the server

When you log in:
1. The server sends a challenge (random data)
2. Your device signs the challenge with the private key
3. The server verifies the signature using the public key

**Why this is brilliant:**
- **Phishing-proof:** The private key is bound to the specific domain. A fake site can't trick you.
- **Breach-proof:** Even if a server gets hacked, attackers only get public keys (useless without the private key)
- **No passwords to remember:** Your device handles authentication with biometrics or PIN
- **Faster:** No typing passwords, just Touch ID or Face ID

---

## My Development Setup

After researching the ecosystem, here's what I chose:

**Backend: Go**
- **Library:** `go-webauthn/webauthn` v0.14.0
- **Why Go:** Strong cryptography libraries, great for backend APIs, and WebAuthn support is mature

**Frontend: React + TypeScript**
- **Library:** `@simplewebauthn/browser` v13.2.2
- **Why React:** Modern, component-based, and SimpleWebAuthn makes WebAuthn API calls simple
- **UI:** Tailwind CSS + shadcn/ui for clean components

**Database: PostgreSQL**
- Running in Docker for easy local development
- Stores users, credentials (public keys), and temporary challenges

**Development Environment:**
- Backend: `http://localhost:8080`
- Frontend: `http://localhost:5173` (Vite dev server)
- Database: PostgreSQL in Docker on port 5432

---

## Phase 1: Understanding WebAuthn Concepts

Before writing code, I spent time wrapping my head around WebAuthn terminology. Here are the key concepts that clicked for me:

### Relying Party (RP)
That's your website. You're "relying" on the authenticator to verify the user's identity.

### Authenticator
The hardware or software that creates and stores passkeys:
- **Platform authenticators:** Touch ID, Face ID, Windows Hello (built into your device)
- **Cross-platform authenticators:** YubiKey, security keys (portable between devices)

### Challenge-Response Authentication
Instead of sending a secret (password), you prove you have the private key by signing a challenge:
```
Server: "Prove you're the real user. Sign this random data: abc123"
Client: *signs with private key* → "Here's the signature: xyz789"
Server: *verifies signature with public key* → "✅ Welcome back!"
```

### Registration (Attestation) vs. Login (Assertion)
- **Registration:** Creating a new passkey (server gets public key)
- **Login:** Using an existing passkey (proving you have the private key)

### The Aha Moment
WebAuthn isn't just "better passwords"—it's a completely different paradigm. Instead of shared secrets (passwords), it's asymmetric cryptography. The mental model shift took time, but once it clicked, everything made sense.

---

## Phase 2: Building the Registration Flow

Registration is where users create their first passkey. Here's what happens behind the scenes:

### The Flow
```
1. User enters username
2. Backend creates user, generates challenge
3. Backend sends "credential creation options" to frontend
4. Frontend calls navigator.credentials.create()
5. Browser shows "Create a passkey" prompt
6. User authenticates (Touch ID, Face ID, etc.)
7. Authenticator creates key pair
8. Public key sent back to server
9. Server verifies and stores public key
```

### What I Built

**Backend (Go):**
```go
// POST /register/begin
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
    // 1. Create user in database
    user := models.User{Username: req.Username}
    db.QueryRow("INSERT INTO users (username) VALUES ($1) RETURNING id",
        user.Username).Scan(&user.ID)

    // 2. Generate WebAuthn challenge
    options, sessionData, err := h.WebAuthn.BeginRegistration(&user)

    // 3. Store complete SessionData (CRITICAL!)
    sessionDataJSON, _ := json.Marshal(sessionData)
    db.Exec("INSERT INTO challenges (...) VALUES ($1, $2, ...)",
        user.ID, sessionDataJSON)

    // 4. Return options to frontend
    json.NewEncoder(w).Encode(options)
}

// POST /register/finish
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
    // 1. Parse credential from request
    parsedResponse, _ := protocol.ParseCredentialCreationResponseBody(r.Body)

    // 2. Retrieve stored SessionData
    var sessionDataJSON []byte
    db.QueryRow("SELECT challenge FROM challenges...").Scan(&sessionDataJSON)
    var sessionData webauthn.SessionData
    json.Unmarshal(sessionDataJSON, &sessionData)

    // 3. Verify credential
    credential, _ := h.WebAuthn.CreateCredential(&user, sessionData, parsedResponse)

    // 4. Save credential with ALL metadata
    db.Exec(`INSERT INTO credentials
        (id, user_id, public_key, sign_count, backup_eligible, backup_state, aaguid)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
        credential.ID, user.ID, credential.PublicKey,
        credential.Authenticator.SignCount,
        credential.Flags.BackupEligible,  // IMPORTANT!
        credential.Flags.BackupState,     // IMPORTANT!
        credential.Authenticator.AAGUID)
}
```

**Frontend (React + SimpleWebAuthn):**
```typescript
export async function registerUser(username: string) {
    // 1. Get challenge from server
    const beginResponse = await fetch(`${API_URL}/register/begin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username }),
    });
    const options = await beginResponse.json();

    // 2. Browser WebAuthn API creates credential
    const credential = await startRegistration(options);

    // 3. Send credential to server for verification
    const finishResponse = await fetch(`${API_URL}/register/finish`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credential),
    });

    return await finishResponse.json();
}
```

### Challenges I Faced

**Challenge 1: "Invalid Attestation Format" Error**

My first end-to-end test failed with this cryptic error. I spent hours debugging:
- ✅ Configured attestation preferences
- ✅ Verified the browser was sending "none" attestation correctly
- ✅ Checked library versions
- ❌ Still failing!

**The Real Problem:**
I was only storing the `challenge` field and reconstructing `SessionData` manually:
```go
// WRONG - Missing critical fields!
sessionData := webauthn.SessionData{
    Challenge: challenge,
    UserID:    user.ID[:],
}
```

But `CreateCredential()` needs the **complete SessionData** including:
- `RelyingPartyID` (to verify the domain)
- `CredParams` (allowed algorithms)
- Other verification parameters

**The Solution:**
Store the ENTIRE SessionData object from `BeginRegistration`:
```go
// RIGHT - Store everything!
sessionDataJSON, _ := json.Marshal(sessionData)  // Complete object
// Later: json.Unmarshal(sessionDataJSON, &sessionData)
```

**Lesson:** Don't reconstruct complex state. Store it opaquely as the spec recommends.

---

## Phase 3: Building the Login Flow

Once registration worked, I built the login (authentication) flow.

### The Flow
```
1. User enters username
2. Backend loads user's credentials
3. Backend generates new challenge
4. Frontend calls navigator.credentials.get()
5. Browser shows "Use your passkey" prompt
6. User authenticates (Touch ID, Face ID, etc.)
7. Authenticator signs challenge with private key
8. Signature sent to server
9. Server verifies signature with public key
10. Session created, user logged in!
```

### Challenges I Faced

**Challenge 2: "Backup Eligible Flag Inconsistency" Error**

Registration worked perfectly. But login failed with:
```
Backup Eligible flag inconsistency detected during login validation
```

**What?**

After adding debug logging, I discovered credentials have security-critical flags:
```go
Flags: {
    UserPresent: true       // User was present during creation
    UserVerified: true      // User verified identity (biometric/PIN)
    BackupEligible: true    // Credential CAN be backed up
    BackupState: true       // Credential IS backed up
}
```

I wasn't storing these! When the library compared flags between registration and login, it detected an inconsistency—a potential security issue (cloned credentials).

**The Solution:**

1. **Add database columns:**
```sql
ALTER TABLE credentials
ADD COLUMN backup_eligible BOOLEAN DEFAULT false,
ADD COLUMN backup_state BOOLEAN DEFAULT false,
ADD COLUMN aaguid BYTEA;
```

2. **Save flags during registration:**
```go
db.Exec(`INSERT INTO credentials (..., backup_eligible, backup_state, aaguid)
    VALUES (..., $5, $6, $7)`,
    credential.Flags.BackupEligible,
    credential.Flags.BackupState,
    credential.Authenticator.AAGUID)
```

3. **Load flags during login:**
```go
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

**Lesson:** Credential flags aren't optional metadata—they're security features. Store the complete credential state!

---

## Phase 4: The Moment It All Worked

After fixing the backup flags issue, I tested the complete flow:

1. ✅ Registered a new user with my MacBook's Touch ID
2. ✅ Logged in using the passkey
3. ✅ Accessed the protected "Secret" page
4. ✅ Logged out and logged back in seamlessly

**The experience was magical:**
- No password to type
- No "remember me" checkbox
- Just touch the sensor and you're in
- ~2 seconds total

This is what authentication should feel like!

---

## Security Deep Dive: What Makes Passkeys Secure?

After building this, I have a deep appreciation for WebAuthn's security model:

### 1. Phishing Resistance
The private key is cryptographically bound to your domain. If a user visits `fakegoogle.com` instead of `google.com`, the passkey won't work. The browser enforces this at the protocol level—users can't be tricked.

### 2. Breach Immunity
If my database gets hacked, attackers only steal:
- Public keys (useless without private keys)
- Usernames (not secret anyway)

They can't log in as users because the private keys never leave users' devices.

### 3. Clone Detection
The `SignCount` increments with every use. If the library detects the counter went backwards or didn't increment, it means the credential was cloned—a potential attack. The `CloneWarning` flag alerts the server.

### 4. Backup State Transparency
The `BackupEligible` and `BackupState` flags tell you if a credential is synced to iCloud Keychain or Google Password Manager. You can enforce policies based on this (e.g., require non-synced hardware keys for admin accounts).

### 5. Origin Binding
Every credential includes the origin (domain + protocol). The authenticator verifies the origin matches before signing. This prevents credential reuse across domains.

---

## Comparison: Passwords vs Passkeys

After building both systems, here's my honest comparison:

### Security

| Aspect | Passwords | Passkeys |
|--------|-----------|----------|
| **Phishing resistance** | ❌ Easily phished | ✅ Cryptographically impossible |
| **Credential stuffing** | ❌ Major problem | ✅ Not applicable (unique per site) |
| **Brute force attacks** | ⚠️ Possible with weak passwords | ✅ Not applicable (no guessing) |
| **Data breach impact** | ❌ Game over | ✅ Minimal (public keys only) |
| **Keylogger resistance** | ❌ Vulnerable | ✅ Immune (nothing to log) |

### User Experience

| Aspect | Passwords | Passkeys |
|--------|-----------|----------|
| **Registration** | Type password, confirm it, maybe verify email | Touch ID, done |
| **Login** | Remember password, type it | Touch ID, done |
| **Forgot password** | Email flow, password reset | Not applicable |
| **Multi-device** | Remember or reset everywhere | Syncs via iCloud/Google |
| **Speed** | ~10-15 seconds | ~2-3 seconds |
| **Accessibility** | Works everywhere | Requires WebAuthn support |

### Developer Experience

| Aspect | Passwords | Passkeys |
|--------|-----------|----------|
| **Implementation complexity** | Moderate | High (but worth it) |
| **Security considerations** | Many (salting, hashing, timing attacks) | Handled by protocol |
| **Password reset flows** | Required | Not needed! |
| **Session management** | Same for both | Same for both |
| **Testing** | Easy (any string works) | Requires actual authenticator |

**Verdict:** Passkeys win on security and UX. The implementation complexity is higher initially, but you skip the entire password reset nightmare.

---

## Common Mistakes I Made (So You Don't Have To)

### Mistake 1: Reconstructing SessionData Instead of Storing It
**Impact:** "Invalid attestation format" error that took hours to debug
**Fix:** Store the complete `SessionData` object from `BeginRegistration` as JSON

### Mistake 2: Not Storing Credential Flags
**Impact:** "Backup eligible flag inconsistency" error during login
**Fix:** Store `backup_eligible`, `backup_state`, `aaguid`, and `attestation_type` in the database

### Mistake 3: Returning Empty Response Bodies
**Impact:** Frontend JSON parse errors even when backend succeeded
**Fix:** Always return JSON, even for success: `{"status": "success"}`

### Mistake 4: Only Testing the Happy Path
**Impact:** Missed error cases until integration testing
**Fix:** Test with invalid challenges, expired sessions, wrong origins, etc.

### Mistake 5: Assuming Error Messages Are Literal
**Impact:** Wasted time on "Invalid attestation format" when the real issue was missing data
**Fix:** Debug with comprehensive logging (`%+v` in Go), compare expected vs actual state

---

## Lessons Learned

### Technical Lessons

1. **WebAuthn is well-designed:** Once I understood the concepts, the API made perfect sense
2. **Libraries handle the hard parts:** Cryptography, challenge generation, signature verification—all handled by `go-webauthn` and `@simplewebauthn`
3. **Printf debugging is underrated:** When integration issues arise, log everything. Compare what the library expects vs. what you're providing
4. **The spec is your friend:** When confused, read the [WebAuthn spec](https://www.w3.org/TR/webauthn-3/). It's surprisingly readable.
5. **Start simple:** Get basic registration/login working before adding features

### Product Lessons

1. **Users love passkeys:** The UX is genuinely better than passwords
2. **Browser support is excellent:** All modern browsers support WebAuthn Level 3
3. **Platform matters:** iPhone users need iOS 16+, Android needs OS 9+
4. **Backup state is transparent:** Users can see if their passkey syncs to the cloud
5. **Recovery is still needed:** What if someone loses all their devices? Plan for account recovery.

### What I'd Do Differently

If I started over:
1. **Read the integration guide first:** I should have looked for common pitfalls before coding
2. **Test incrementally:** Don't build the entire backend before testing
3. **Use the exact library versions from docs:** Stick with known-good versions initially
4. **Start with SimpleWebAuthn's example:** Their demo code is production-quality

---

## Resources That Helped Me

### Essential Resources

- **[webauthn.guide](https://webauthn.guide/)**: Visual guide that made concepts click
- **[go-webauthn docs](https://pkg.go.dev/github.com/go-webauthn/webauthn/webauthn)**: Complete API reference
- **[SimpleWebAuthn docs](https://simplewebauthn.dev/)**: Best frontend library for WebAuthn
- **[WebAuthn spec](https://www.w3.org/TR/webauthn-3/)**: Surprisingly readable official spec
- **[webauthn.io](https://webauthn.io/)**: Interactive demo to test your authenticator

### Libraries I Used

- **go-webauthn/webauthn (v0.14.0)**: Excellent Go library for backend
- **@simplewebauthn/browser (v13.2.2)**: Makes WebAuthn API calls dead simple
- **shadcn/ui**: Beautiful React components out of the box
- **Vite**: Lightning-fast frontend dev server

---

## Current State of the Project

### What Works
- ✅ Full registration flow (create passkey)
- ✅ Full login flow (authenticate with passkey)
- ✅ Protected routes (shows secret message only when logged in)
- ✅ Sign counter tracking (clone detection)
- ✅ Backup flag validation (security feature)
- ✅ CORS configured for frontend-backend communication
- ✅ Challenge expiration and cleanup
- ✅ Works with Touch ID, Face ID, security keys

### What's Not Implemented (But Could Be)
- Session management with secure cookies (currently using localStorage for simplicity)
- Logout endpoint
- Multiple credentials per user
- Credential management UI (view/delete passkeys)
- Discoverable credentials (usernameless login)
- Conditional UI (passkey autofill)
- Account recovery flow
- CloneWarning alerts
- Rate limiting

### Demo

Since this is a learning project, it runs on localhost. Here's what it looks like:

**Registration:**
1. Visit `http://localhost:5173/register`
2. Enter username
3. Click "Register with Passkey"
4. Touch ID prompt appears
5. Touch sensor → Registered!

**Login:**
1. Visit `http://localhost:5173/login`
2. Enter username
3. Click "Login with Passkey"
4. Touch ID prompt appears
5. Touch sensor → Logged in!

**Secret Page:**
Protected route that shows "O Captain! My Captain!" message only when authenticated.

---

## Future Exploration

### Topics I Want to Dive Deeper Into

1. **Discoverable credentials (resident keys):** Login without typing username
2. **Conditional UI:** Passkey suggestions in autofill dropdowns
3. **Cross-device authentication:** Use phone as authenticator for laptop
4. **Attestation verification:** Verify authenticator make/model
5. **Multi-device sync:** How iCloud Keychain and Google Password Manager work
6. **Enterprise deployment:** FIDO2 policies, hardware key requirements

### Potential Improvements

- Add session management with `gorilla/sessions`
- Implement refresh tokens for long-lived sessions
- Add credential management dashboard
- Build account recovery flow (backup codes? secondary email?)
- Test on different devices (iPhone, Android, Windows Hello)
- Add logging and monitoring
- Write integration tests with real authenticators

---

## Conclusion

Building a passkey authentication system was one of the most rewarding learning projects I've done. The technology is sophisticated but the libraries make it accessible. The debugging was challenging but taught me WebAuthn internals deeply.

**Key Takeaways:**
- Passkeys are the future of authentication
- The security model is fundamentally better than passwords
- The UX is genuinely superior
- Implementation is complex but libraries help immensely
- Common pitfalls (SessionData, credential flags) are solvable with proper debugging

### Would I Use Passkeys in Production?

**Absolutely.** But I'd:
1. Use passkeys as the *primary* method with password as fallback (during transition)
2. Implement robust account recovery
3. Thoroughly test across devices and browsers
4. Monitor adoption rates and user feedback
5. Educate users about what passkeys are (many don't know yet)

### Recommendations for Others Learning Passkeys

1. **Start with the concepts:** Understand public-key cryptography, challenge-response, and WebAuthn flows before coding
2. **Use good libraries:** `go-webauthn` for backend, `@simplewebauthn` for frontend—don't reinvent the wheel
3. **Test incrementally:** Get registration working before login. Get basic flows working before advanced features.
4. **Debug with logging:** When things fail, log SessionData, credentials, flags—compare expected vs. actual
5. **Read the spec:** The WebAuthn spec is well-written. Reference it when confused.
6. **Build something real:** Reading isn't enough. Build a complete system to truly understand it.
7. **Document your journey:** I learned so much by writing down what I learned along the way

---

## Appendix

### Glossary

- **WebAuthn**: Web Authentication API, the W3C standard for passwordless authentication
- **FIDO2**: Alliance of FIDO (Fast Identity Online) + WebAuthn + CTAP protocols
- **Authenticator**: Hardware or software that creates and stores passkeys (Touch ID, security keys, etc.)
- **Attestation**: Cryptographic proof about the authenticator during registration
- **Assertion**: Cryptographic proof of identity during authentication (login)
- **Relying Party (RP)**: Your website/application
- **Challenge**: Random data that must be signed to prove possession of private key
- **AAGUID**: Authenticator Attestation GUID, identifies make/model of authenticator
- **Sign Counter**: Increments with each use, detects cloned credentials
- **Backup State**: Whether the credential is synced to cloud (iCloud, Google Password Manager)

### Complete Code Repository

**GitHub:** [Coming soon - will publish once I clean up the code and remove debug logging]

### Timeline

- **October 14, 2025**: Database schema design, backend registration endpoints
- **October 18, 2025**: Backend login endpoints, sign counter implementation
- **October 19, 2025**: Frontend React app, integration debugging (attestation format, backup flags), complete end-to-end testing
- **Total time invested**: ~12 hours across 3 days (including debugging and documentation)

---

**Thank you for reading!** If you found this helpful, consider trying to build your own passkey system. The best way to learn is by doing—and debugging. 😅

*Questions? Feedback? Found this helpful? I'd love to hear from you.*

---

*Last updated: October 19, 2025*
*Tech Stack: React 19, Go, PostgreSQL, WebAuthn*
*Project repository: [GitHub link coming soon]*
