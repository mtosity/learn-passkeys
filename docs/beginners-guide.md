# WebAuthn Beginner's Guide

## What Problem Does WebAuthn Solve?

### The Password Problem

Passwords have been around forever, but they have serious issues:

1. **Users choose weak passwords** - "password123", "qwerty", etc.
2. **Password reuse** - Same password on multiple sites
3. **Phishing attacks** - Fake sites steal passwords
4. **Data breaches** - Hackers steal password databases
5. **Password reset flows** - Complex and frustrating
6. **Hard to remember** - Especially strong, unique passwords

### The WebAuthn Solution

WebAuthn eliminates passwords entirely by using **public key cryptography**:

- No passwords to remember
- Nothing to type or transmit
- Phishing-proof (credentials bound to specific domains)
- Breach-proof (no secrets stored on server)
- Fast and convenient (biometrics or PIN)

## Core Concepts Explained Simply

### 1. Public Key Cryptography (The Magic Behind It)

Think of it like a special mailbox:

**Public Key** = Mailbox with a slot
- Anyone can put mail in (encrypt messages)
- Server stores this
- Safe to share publicly

**Private Key** = Only key that opens the mailbox
- Only you can read the mail (decrypt messages)
- Stays on your device FOREVER
- Never shared with anyone

**How it works for authentication:**
1. Server sends a challenge: "Prove you own this credential"
2. Your device signs the challenge with the private key (like a wax seal)
3. Server verifies the signature with the public key
4. If it matches, you're authenticated!

### 2. The Authenticator

An **authenticator** is the device/system that:
- Stores your private keys securely
- Prompts you to verify (fingerprint, face, PIN)
- Creates digital signatures

**Types of authenticators:**

**Platform Authenticators** (built into your device):
- Touch ID on Mac
- Face ID on iPhone
- Windows Hello
- Android biometrics

**Cross-platform Authenticators** (portable):
- YubiKey (USB security key)
- Google Titan Key
- Any FIDO2 security key

### 3. The Relying Party (RP)

That's YOUR website/application:
- The party "relying" on the authentication
- Your Go backend is the RP
- Identified by your domain (localhost:8080 in development)

### 4. Challenges (Preventing Replay Attacks)

A **challenge** is random data the server sends:

```
Server: "Sign this random data: a3f9d8e2c1b7..."
Client: *signs with private key* "Here's my signature: x7k2..."
Server: *verifies signature* "Signature valid, you're authenticated!"
```

**Why challenges matter:**
- Each login uses a new random challenge
- Prevents attackers from reusing old signatures
- Challenge must be used within 5 minutes (expires)
- Challenge is used exactly once (prevents replay)

### 5. Origin Binding (Preventing Phishing)

Every credential is **bound to your domain**:

```
You register on: yourbank.com
Credential works on: yourbank.com ONLY
Phishing site: yourbank-login.evil.com
Credential works?: NO! ❌
```

Even if you're tricked into visiting a fake site:
- The credential won't work on the fake domain
- The signature will include the real origin
- Your server will reject it (origin mismatch)

This makes WebAuthn **phishing-resistant** by design!

## The Two Main Operations

### Registration (Creating a Passkey)

**What happens:**
1. User wants to create an account
2. Server generates random challenge
3. Browser asks authenticator to create credential
4. User verifies with biometric/PIN
5. Authenticator creates key pair
6. Public key sent to server
7. Server stores: credential ID + public key + user info

**Analogy:**
Like making a new key for your house:
- You (authenticator) create the key
- You keep the original (private key)
- You give a copy to your friend (server gets public key)
- Your friend can verify it's the right key, but can't make new keys

### Authentication (Using a Passkey)

**What happens:**
1. User wants to log in
2. Server generates new random challenge
3. Browser asks authenticator to sign challenge
4. User verifies with biometric/PIN
5. Authenticator signs with private key
6. Signature sent to server
7. Server verifies signature with stored public key
8. If valid, user is logged in!

**Analogy:**
Like proving you have the house key:
- Friend (server) asks you to open the door (challenge)
- You use your key (private key) to open it (sign challenge)
- Friend sees the door open (verifies signature)
- Friend knows you have the right key (authenticated!)

## Key Security Properties

### 1. No Shared Secrets

**Traditional password:**
```
User knows: "password123"
Server stores: hash of "password123"
Problem: Both parties have information about the password
```

**WebAuthn:**
```
User has: Private key (never leaves device)
Server has: Public key (useless without private key)
Benefit: Even if server is hacked, attackers can't impersonate users
```

### 2. Phishing Resistance

The credential knows which website it belongs to:

```javascript
// Credential stores:
{
  rpId: "yoursite.com",
  credentialId: "abc123",
  // ... other data
}

// When used on evil.com:
origin: "evil.com" ❌ doesn't match "yoursite.com"
→ Browser rejects it before even prompting user
```

### 3. Attestation (Optional)

Attestation proves what type of authenticator was used:
- Was it a hardware security key?
- Was it platform biometrics?
- Was it a specific vendor's device?

**Use cases:**
- Require hardware security keys for admin accounts
- Require platform authenticators for normal users
- Audit which devices are being used

**For learning:** You can skip attestation - use "none" format

### 4. User Verification

**User Presence**: User is there (touch the key)
**User Verification**: User proved their identity (biometric/PIN)

You can require different levels:
- `preferred`: Ask for verification if available
- `required`: Must have verification or reject
- `discouraged`: Don't ask for verification

## Data Flow Summary

### What Travels Over the Network

**Registration:**
```
Browser → Server:
{
  credentialId: "abc123...",
  publicKey: "MFkw...",
  attestationObject: {...}
}

❌ Private key NEVER travels!
```

**Authentication:**
```
Browser → Server:
{
  credentialId: "abc123...",
  signature: "x7k2...",
  authenticatorData: {...},
  clientDataJSON: {...}
}

❌ Private key NEVER travels!
```

### What Gets Stored in Your Database

**Users Table:**
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  display_name TEXT,
  created_at TIMESTAMP
);
```

**Credentials Table:**
```sql
CREATE TABLE credentials (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  credential_id BYTEA UNIQUE NOT NULL,  -- From authenticator
  public_key BYTEA NOT NULL,            -- For signature verification
  counter INTEGER NOT NULL DEFAULT 0,   -- Signature counter
  transports TEXT[],                    -- usb, nfc, ble, internal
  created_at TIMESTAMP
);
```

**Challenges Table** (temporary storage):
```sql
CREATE TABLE challenges (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  challenge BYTEA NOT NULL,
  type TEXT NOT NULL,                   -- 'registration' or 'authentication'
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP
);
```

## Browser Support

WebAuthn works on:
- Chrome/Edge 67+
- Firefox 60+
- Safari 13+
- iOS Safari 14.5+
- Android Chrome 70+

**For development:**
- Works on localhost without HTTPS
- Production REQUIRES HTTPS

## Common Misconceptions

### "WebAuthn replaces passwords with biometrics"
**Not quite!** Biometrics are used to unlock the private key on your device. The server never sees your fingerprint. If your device doesn't have biometrics, you can use a PIN instead.

### "If I lose my device, I lose access"
**Partially true.** That's why you should:
- Allow multiple credentials per user (phone + laptop + security key)
- Implement account recovery flows
- Consider backup codes or alternative recovery methods

### "This is the same as 2FA"
**No!** 2FA is password + second factor. WebAuthn replaces the password entirely. No password to steal!

### "My server stores biometrics"
**NO!** Your server only stores public keys. Biometrics stay on the user's device and are never sent over the network.

## Learning Path

Here's how to approach learning WebAuthn:

### Phase 1: Understand the Concepts ← YOU ARE HERE!
- Read these docs
- Watch the flow diagrams
- Understand registration vs authentication
- Learn about challenges and signatures

### Phase 2: Set Up Infrastructure
- PostgreSQL database (done!)
- Database schema (users, credentials, challenges)
- Go project structure
- Install go-webauthn library

### Phase 3: Implement Registration
- Create user in database
- Generate registration options
- Handle credential creation response
- Verify attestation
- Store credential

### Phase 4: Implement Authentication
- Look up user's credentials
- Generate authentication options
- Handle assertion response
- Verify signature
- Create session

### Phase 5: Add Session Management
- Cookie-based sessions
- Middleware to protect routes
- Logout functionality

### Phase 6: Build the Frontend
- React components for register/login
- Use @simplewebauthn/browser
- Handle errors gracefully
- Good UX for authenticator prompts

### Phase 7: Test Everything
- Happy path (successful reg + auth)
- Error cases (wrong challenge, expired session)
- Multiple devices
- Edge cases

## Helpful Mental Models

### The Restaurant Analogy

**Registration = Getting a membership card**
- You visit restaurant (website)
- They create a membership (generate challenge)
- You provide your thumbprint (biometric)
- They create a card linked to your thumbprint (credential)
- They keep a copy of your thumbprint pattern (public key)
- You keep the actual thumbprint (private key)

**Authentication = Using your membership**
- You return to restaurant
- They ask you to prove membership (challenge)
- You press your thumb (sign with private key)
- They compare to stored pattern (verify signature)
- Match! You're in (authenticated)

### The Lock and Key Analogy

**Registration = Making a special lock**
- Website creates a lock (public key)
- Your device creates the only key (private key)
- Website keeps the lock
- You keep the key on your device
- Key never leaves your possession

**Authentication = Using the key**
- Website asks: "Can you open this lock?"
- Your device uses the key (signs challenge)
- Website checks if it opened (verifies signature)
- If yes, you're in!

## Questions to Ask Yourself

As you implement this project, keep asking:

1. **Where does the private key live?** (Should always be: on the user's device)
2. **What if someone steals my database?** (They only get public keys - useless!)
3. **What if someone intercepts the network traffic?** (They get a signature that's already been used - useless!)
4. **What if a phishing site tricks my user?** (Credential won't work - origin doesn't match!)
5. **What happens if the user loses their device?** (They need another registered credential or recovery method)

## Resources for Going Deeper

- **webauthn.guide** - Visual guide with animations
- **webauthn.io** - Try it yourself in real-time
- **MDN Web Authentication API** - Complete technical reference
- **go-webauthn/webauthn docs** - Your Go library documentation
- **@simplewebauthn docs** - Frontend library documentation
- **FIDO Alliance** - The organization behind WebAuthn standards

## Next Steps

Now that you understand the concepts:

1. ✅ Read registration flow diagram
2. ✅ Read authentication flow diagram
3. ✅ Read this guide
4. ⬜ Review the glossary (next!)
5. ⬜ Design your database schema
6. ⬜ Start implementing the backend step by step

Ready to start coding? I'll guide you through each step!
