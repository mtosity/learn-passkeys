# WebAuthn Authentication Flow - Visual Diagram

## The Complete Login Journey

```
┌─────────────┐                 ┌─────────────┐                 ┌─────────────────┐
│   Browser   │                 │   Server    │                 │  Authenticator  │
│  (Frontend) │                 │  (Backend)  │                 │ (Touch ID/Key)  │
└──────┬──────┘                 └──────┬──────┘                 └────────┬────────┘
       │                               │                                 │
       │  1. User clicks "Login"       │                                 │
       │      (may enter username)     │                                 │
       │──────────────────────────────>│                                 │
       │                               │                                 │
       │                         2. Generate new                         │
       │                            challenge                            │
       │                               │                                 │
       │                         3. Look up user's                       │
       │                            registered credentials               │
       │                            from DB                              │
       │                               │                                 │
       │   4. Send authentication      │                                 │
       │      options (JSON)           │                                 │
       │<──────────────────────────────│                                 │
       │   {                           │                                 │
       │     challenge: "abc...",      │                                 │
       │     rpId: "example.com",      │                                 │
       │     allowCredentials: [       │                                 │
       │       {id: "cred_id", ...}    │                                 │
       │     ]                         │                                 │
       │   }                           │                                 │
       │                               │                                 │
       │  5. Call navigator.credentials.get()                            │
       │────────────────────────────────────────────────────────────────>│
       │                               │                                 │
       │                               │          6. User authenticates  │
       │                               │             (Touch ID, PIN, etc)│
       │                               │                                 │
       │                               │          7. Find matching       │
       │                               │             credential (private │
       │                               │             key)                │
       │                               │                                 │
       │                               │          8. Sign challenge with │
       │                               │             private key         │
       │                               │                                 │
       │                               │          9. Increment counter   │
       │                               │                                 │
       │  10. Return assertion                                           │
       │<────────────────────────────────────────────────────────────────│
       │   {                           │                                 │
       │     id: "credential_id",      │                                 │
       │     response: {               │                                 │
       │       clientDataJSON: ...,    │                                 │
       │       authenticatorData: ..., │                                 │
       │       signature: ...,         │                                 │
       │       userHandle: ...         │                                 │
       │     }                         │                                 │
       │   }                           │                                 │
       │                               │                                 │
       │  11. Send assertion to server │                                 │
       │──────────────────────────────>│                                 │
       │                               │                                 │
       │                        12. Verify:                              │
       │                            - Challenge matches                  │
       │                            - Origin matches                     │
       │                            - RP ID matches                      │
       │                            - Counter increased                  │
       │                               │                                 │
       │                        13. Verify signature                     │
       │                            using stored public key              │
       │                               │                                 │
       │                        14. Update counter in DB                 │
       │                               │                                 │
       │                        15. Create session                       │
       │                            (cookie or JWT)                      │
       │                               │                                 │
       │  16. Success response         │                                 │
       │      + Set-Cookie header      │                                 │
       │<──────────────────────────────│                                 │
       │                               │                                 │
       │  17. Redirect to protected    │                                 │
       │      page (secret message)    │                                 │
       │                               │                                 │
```

## Key Concepts in Authentication

### Step 2: Fresh Challenge
- **Must be different** from registration challenge
- **Must be random** and unpredictable
- **Expires quickly** (typically 5 minutes)
- **Single use only** - deleted after verification

### Step 4: allowCredentials
This tells the browser which credentials can be used:
```javascript
allowCredentials: [
  {
    id: base64url_encoded_credential_id,
    type: "public-key",
    transports: ["internal", "usb", "nfc", "ble"]
  }
]
```

**Two approaches:**
1. **With username**: Look up that user's credentials
2. **Discoverable credentials**: Leave array empty, let authenticator suggest credentials

### Step 8: Signing the Challenge
- Authenticator uses the **private key** (stored securely on device)
- Creates a digital signature of the challenge
- This proves the user has access to the device that created the credential
- **No password needed!**

### Step 12-13: Verification is Critical
The server must verify:

1. **Challenge matches**: Prevents replay attacks
2. **Origin matches**: The assertion came from your domain
3. **RP ID matches**: Relying Party ID is correct
4. **Signature is valid**: Using the stored public key
5. **Counter increased**: Detects cloned credentials

```
If counter didn't increase or decreased:
  → Possible cloned authenticator
  → Security alert!
  → May need to revoke credential
```

### Step 15: Session Creation
After successful verification:
- Create a session (cookie or JWT)
- Store user ID in session
- Set secure cookie flags: `HttpOnly`, `Secure`, `SameSite`
- User is now logged in!

## Comparison: Login Flow

### Traditional Password Login
```
User → enters password → Server checks hash → Session created
                          ❌ Password sent over network
                          ❌ Password stored (hashed) in DB
                          ❌ Vulnerable to phishing
                          ❌ Can be guessed/cracked
```

### Passkey Login (WebAuthn)
```
User → biometric/PIN → Authenticator signs challenge → Server verifies signature → Session created
                        ✅ No password sent
                        ✅ No password stored
                        ✅ Phishing resistant (origin-bound)
                        ✅ Can't be guessed (cryptographic)
```

## What Gets Verified?

### Client Data JSON
Contains information about the authentication ceremony:
- Challenge (must match what server sent)
- Origin (must match your domain)
- Type (must be "webauthn.get")

### Authenticator Data
Contains:
- RP ID hash (must match your domain's hash)
- Flags (user present, user verified)
- Counter (must increase from last authentication)

### Signature
- Created by signing the challenge with private key
- Verified using public key stored in your database
- If signature is valid → user is authenticated!

## Security: Why This Works

### Protection Against Phishing
Imagine evil.com tries to phish your users:

1. Evil site asks for login
2. Browser creates assertion for evil.com origin
3. User sends assertion to evil.com
4. Evil.com forwards it to yoursite.com
5. **yoursite.com rejects it** - origin doesn't match!

The credential is **bound to your domain** and won't work anywhere else.

### Protection Against Database Breach
If your database is stolen:
- Attackers get public keys (useless without private keys)
- Private keys are on users' devices, not in your database
- Attackers can't impersonate users without the device

### Protection Against Password Reuse
- No passwords to reuse!
- Each credential is unique per site
- Users can't make weak choices

## Common Authentication Patterns

### Pattern 1: Username-first
```
1. User enters username
2. Server looks up credentials for that username
3. Send allowCredentials with specific credential IDs
4. Authenticator uses matching credential
```

### Pattern 2: Discoverable Credentials
```
1. User clicks "Login with passkey"
2. Server sends empty allowCredentials
3. Authenticator shows list of available credentials
4. User picks one
5. Server identifies user from userHandle in response
```

### Pattern 3: Conditional UI
```
1. User sees autofill suggestion in username field
2. User selects passkey from autofill
3. Browser triggers authentication
4. Seamless login!
```

## Troubleshooting Authentication

### "No credentials available"
- User hasn't registered on this device
- Credential was deleted
- Browser/authenticator doesn't recognize the credential ID

### "Verification failed"
- Challenge mismatch (expired or wrong challenge)
- Origin mismatch (check your rpId configuration)
- Invalid signature (public key doesn't match)
- Counter decreased (possible cloned authenticator)

### "User cancelled"
- User dismissed the biometric prompt
- User doesn't want to authenticate
- Not an error - just handle gracefully

## What to Store for Sessions

After successful login, typical session data:
```javascript
session = {
  userId: "user_123",
  credentialId: "cred_456",  // which credential was used
  authenticatedAt: timestamp,
  expiresAt: timestamp + session_duration
}
```

## Testing Tips

1. **Test the happy path first**: Successful login
2. **Test with wrong challenge**: Should fail verification
3. **Test with expired session**: Should require re-authentication
4. **Test on multiple devices**: See how credentials are device-specific
5. **Test logout**: Clear session, require re-authentication

## Next Steps

After understanding these flows:
1. Set up database schema for users and credentials
2. Implement registration endpoint
3. Implement authentication endpoint
4. Add session management
5. Create protected routes that require authentication
