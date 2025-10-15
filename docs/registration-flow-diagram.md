# WebAuthn Registration Flow - Visual Diagram

## The Complete Registration Journey

```
┌─────────────┐                 ┌─────────────┐                 ┌─────────────────┐
│   Browser   │                 │   Server    │                 │  Authenticator  │
│  (Frontend) │                 │  (Backend)  │                 │ (Touch ID/Key)  │
└──────┬──────┘                 └──────┬──────┘                 └────────┬────────┘
       │                               │                                 │
       │  1. User clicks "Register"    │                                 │
       │──────────────────────────────>│                                 │
       │                               │                                 │
       │                         2. Generate random                      │
       │                            challenge (32 bytes)                 │
       │                               │                                 │
       │                         3. Create user in DB                    │
       │                            (if new)                             │
       │                               │                                 │
       │   4. Send registration        │                                 │
       │      options (JSON)           │                                 │
       │<──────────────────────────────│                                 │
       │   {                           │                                 │
       │     challenge: "xyz...",      │                                 │
       │     rp: {name, id},           │                                 │
       │     user: {id, name, ...}     │                                 │
       │   }                           │                                 │
       │                               │                                 │
       │  5. Call navigator.credentials.create()                         │
       │────────────────────────────────────────────────────────────────>│
       │                               │                                 │
       │                               │          6. User interacts      │
       │                               │             (Face ID, PIN, etc) │
       │                               │                                 │
       │                               │          7. Generate key pair   │
       │                               │             - Private key (stays│
       │                               │               on device!)       │
       │                               │             - Public key        │
       │                               │                                 │
       │  8. Return credential                                           │
       │<────────────────────────────────────────────────────────────────│
       │   {                           │                                 │
       │     id: "credential_id",      │                                 │
       │     rawId: [...],             │                                 │
       │     response: {               │                                 │
       │       clientDataJSON: ...,    │                                 │
       │       attestationObject: ...  │                                 │
       │     }                         │                                 │
       │   }                           │                                 │
       │                               │                                 │
       │  9. Send credential to server │                                 │
       │──────────────────────────────>│                                 │
       │                               │                                 │
       │                        10. Verify:                              │
       │                            - Challenge matches                  │
       │                            - Origin matches                     │
       │                            - User verification                  │
       │                               │                                 │
       │                        11. Extract public key                   │
       │                            from attestation                     │
       │                               │                                 │
       │                        12. Save to DB:                          │
       │                            - credential_id                      │
       │                            - public_key                         │
       │                            - counter                            │
       │                            - user_id                            │
       │                               │                                 │
       │  13. Success response         │                                 │
       │<──────────────────────────────│                                 │
       │                               │                                 │
       │  14. Redirect to login        │                                 │
       │      or protected page        │                                 │
       │                               │                                 │
```

## Key Concepts in Registration

### Step 2: The Challenge
- **What**: Random bytes (usually 32 bytes)
- **Why**: Prevents replay attacks - each registration uses a unique challenge
- **Important**: Challenge must be stored temporarily and verified in step 10

### Step 7: Key Pair Generation
- **Private Key**: NEVER leaves the authenticator device
- **Public Key**: Sent to server and stored in your database
- **This is the magic**: Because private key stays on device, even if your server is hacked, attackers can't impersonate users

### Step 10: Verification
Critical security checks:
1. **Challenge matches**: The challenge you sent matches what's in the response
2. **Origin matches**: The credential was created for YOUR domain (prevents phishing)
3. **User verification**: Confirms the user was present and verified

### Step 12: What to Store
```sql
credentials table:
├── id (primary key)
├── user_id (foreign key to users table)
├── credential_id (unique identifier from authenticator)
├── public_key (used to verify signatures)
├── counter (signature counter for security)
├── transports (optional: usb, nfc, ble, internal)
└── created_at (timestamp)
```

## Common Mistakes to Avoid

1. **Reusing challenges**: Each registration must have a unique challenge
2. **Not checking origin**: Always verify the credential was created for your domain
3. **Storing private key**: You should NEVER receive or store the private key
4. **Forgetting to clean up**: Delete used challenges after verification

## What Happens Behind the Scenes

When you call `navigator.credentials.create()`:
1. Browser validates the options
2. Browser shows native UI (Touch ID prompt, Windows Hello, etc.)
3. User authenticates (fingerprint, face, PIN)
4. Authenticator generates a new key pair
5. Authenticator signs the challenge with the private key
6. Authenticator returns public key + signature to browser
7. Browser formats the response and returns to your JavaScript

## Testing Tips

- Use Chrome DevTools Console to see any errors
- Check Network tab to see the exact JSON being sent
- Try on different devices to see different authenticators
- Use `console.log()` to inspect the credential object before sending to server
