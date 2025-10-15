# WebAuthn Authentication Flow - Mermaid Diagram

## Interactive Sequence Diagram

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Server
    participant DB as Database
    participant Authenticator

    Note over User,Authenticator: Authentication Ceremony Begins

    User->>Browser: 1. Clicks "Login" button
    Browser->>Server: 2. POST /login/begin (username)

    Note over Server: Generate NEW random challenge
    Server->>DB: 3. Look up user's credentials
    DB-->>Server: Return credential IDs

    Server-->>Browser: 4. Authentication options (JSON)
    Note right of Server: {challenge, rpId,<br/>allowCredentials}

    Browser->>Browser: 5. Parse options
    Browser->>Authenticator: 6. navigator.credentials.get()

    Note over Authenticator: Find matching credential
    Authenticator->>User: 7. "Touch ID to log in..."
    User->>Authenticator: 8. Provides biometric/PIN

    Note over Authenticator: Verify user identity
    Note over Authenticator: Retrieve private key
    Note over Authenticator: Sign challenge with private key
    Note over Authenticator: Increment counter

    Authenticator-->>Browser: 9. Return assertion
    Note right of Authenticator: {id, response: {<br/>clientDataJSON,<br/>authenticatorData,<br/>signature,<br/>userHandle}}

    Browser->>Server: 10. POST /login/finish (assertion)

    Note over Server: Verify challenge matches
    Note over Server: Verify origin matches
    Note over Server: Verify RP ID matches

    Server->>DB: 11. Get stored public key & counter
    DB-->>Server: Return credential data

    Note over Server: Verify signature using public key
    Note over Server: Verify counter increased

    Server->>DB: 12. Update counter
    Server->>Server: 13. Create session (JWT/cookie)

    Server-->>Browser: 14. Success + Set-Cookie
    Browser->>User: 15. Redirect to protected page

    Note over User,Authenticator: Authentication Complete!
```

## Key Steps Explained

### Steps 1-4: Authentication Initiation
- User wants to log in (may enter username)
- Server generates a FRESH, unique challenge
- Server looks up which credentials belong to this user
- Server sends authentication options with `allowCredentials`

### Steps 5-8: User Verification
- Browser invokes WebAuthn API with options
- Authenticator finds the matching credential
- User proves identity (biometric/PIN)
- Authenticator retrieves the private key (never leaves device!)

### Steps 9-10: Signature Creation
- Authenticator signs the challenge with private key
- Counter is incremented (detect cloned credentials)
- Browser sends the assertion to server

### Steps 11-15: Server Verification & Session Creation
- Server verifies challenge, origin, and RP ID
- Server verifies signature using stored public key
- Server checks counter increased (security check)
- Server creates session
- User is logged in!

## Authentication vs Registration

```mermaid
graph LR
    subgraph Registration
        R1[New Key Pair] --> R2[Store Public Key]
        R2 --> R3[User Can Log In]
    end

    subgraph Authentication
        A1[Use Private Key] --> A2[Sign Challenge]
        A2 --> A3[Verify with Public Key]
        A3 --> A4[Create Session]
    end

    R3 -.->|Enables| A1

    style R2 fill:#e1f5e1
    style A3 fill:#e1f5e1
```

## Signature Verification Process

```mermaid
sequenceDiagram
    participant Server
    participant DB as Database
    participant Crypto as Crypto Library

    Note over Server: Assertion received

    Server->>DB: Get public key for credential_id
    DB-->>Server: Return public key & counter

    Server->>Crypto: Verify signature
    Note right of Crypto: signature + public_key +<br/>challenge + clientData +<br/>authenticatorData

    alt Signature Valid
        Crypto-->>Server: ✅ Valid
        Server->>Server: Check counter increased
        alt Counter OK
            Server->>DB: Update counter
            Server->>Server: Create session
            Note over Server: User authenticated!
        else Counter suspicious
            Server->>Server: Reject & alert
            Note over Server: Possible cloned credential
        end
    else Signature Invalid
        Crypto-->>Server: ❌ Invalid
        Server->>Server: Reject authentication
        Note over Server: Wrong credential or tampering
    end
```

## Two Login Patterns

### Pattern 1: Username-First Flow

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Server

    User->>Browser: Enter username
    Browser->>Server: POST /login/begin {username}
    Note over Server: Look up credentials<br/>for this user
    Server-->>Browser: allowCredentials: [specific IDs]
    Browser->>Browser: Use specified credential
    Note over Browser: Authenticator uses<br/>matching credential
```

### Pattern 2: Discoverable Credentials (Usernameless)

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Server
    participant Authenticator

    User->>Browser: Click "Login with passkey"
    Browser->>Server: POST /login/begin
    Server-->>Browser: allowCredentials: [] (empty)
    Browser->>Authenticator: Get available credentials
    Authenticator-->>Browser: Show credential list
    Browser->>User: Show picker UI
    User->>Browser: Select credential
    Note over Browser: userHandle identifies user
```

## Security Verification Flowchart

```mermaid
graph TB
    A[Receive Assertion] --> B{Challenge Valid?}
    B -->|No/Expired| FAIL1[Reject: Invalid Challenge]
    B -->|Yes| C{Origin Matches?}
    C -->|No| FAIL2[Reject: Origin Mismatch]
    C -->|Yes| D{RP ID Matches?}
    D -->|No| FAIL3[Reject: RP ID Mismatch]
    D -->|Yes| E[Get Stored Public Key]
    E --> F{Signature Valid?}
    F -->|No| FAIL4[Reject: Invalid Signature]
    F -->|Yes| G{Counter Increased?}
    G -->|No| FAIL5[Reject: Possible Clone]
    G -->|Yes| H[Update Counter]
    H --> I[Create Session]
    I --> SUCCESS[✅ Authentication Success]

    style FAIL1 fill:#ffcccc
    style FAIL2 fill:#ffcccc
    style FAIL3 fill:#ffcccc
    style FAIL4 fill:#ffcccc
    style FAIL5 fill:#ffcccc
    style SUCCESS fill:#ccffcc
```

## What Gets Verified

```mermaid
graph LR
    subgraph Client Data JSON
        CD1[Challenge]
        CD2[Origin]
        CD3[Type]
    end

    subgraph Authenticator Data
        AD1[RP ID Hash]
        AD2[Flags: UP, UV]
        AD3[Counter]
    end

    subgraph Verification
        V1[Signature]
    end

    CD1 --> V1
    CD2 --> V1
    AD1 --> V1
    AD2 --> V1
    AD3 --> V1

    V1 --> Result{Valid?}
    Result -->|Yes| Success[Authenticated ✅]
    Result -->|No| Fail[Rejected ❌]

    style Success fill:#ccffcc
    style Fail fill:#ffcccc
```

## Phishing Protection Visualization

```mermaid
sequenceDiagram
    participant User
    participant EvilSite as evil.com
    participant YourServer as yoursite.com
    participant Authenticator

    Note over User,Authenticator: Phishing Attack Scenario

    User->>EvilSite: Visits fake site
    EvilSite->>YourServer: Request challenge
    YourServer-->>EvilSite: Send challenge
    EvilSite->>User: Show fake login
    User->>Authenticator: Attempt to authenticate

    Note over Authenticator: Creates signature with<br/>origin: "evil.com"

    Authenticator-->>EvilSite: Return assertion
    EvilSite->>YourServer: Forward assertion

    Note over YourServer: Verify origin...
    Note over YourServer: Expected: "yoursite.com"
    Note over YourServer: Received: "evil.com"

    YourServer-->>EvilSite: ❌ REJECTED: Origin mismatch
    EvilSite->>User: Login failed

    Note over User,Authenticator: Phishing Attempt Blocked!

    rect rgb(255, 200, 200)
        Note over YourServer: The origin binding in WebAuthn<br/>makes phishing impossible!
    end
```

## Session Management

```mermaid
graph TB
    Start[Successful Authentication] --> Create[Create Session]
    Create --> Store{Storage Method?}

    Store -->|Cookie| C1[Set HttpOnly Cookie]
    Store -->|JWT| J1[Create JWT Token]

    C1 --> C2[Add Secure Flag]
    C2 --> C3[Add SameSite Flag]
    C3 --> End1[Store session data]

    J1 --> J2[Sign with secret]
    J2 --> J3[Set expiration]
    J3 --> End2[Return to client]

    End1 --> Protect[Protected Routes]
    End2 --> Protect

    Protect --> MW[Middleware checks session]
    MW -->|Valid| Allow[Allow access]
    MW -->|Invalid| Deny[Redirect to login]

    style Allow fill:#ccffcc
    style Deny fill:#ffcccc
```

## Counter Tracking (Clone Detection)

```mermaid
graph LR
    A[Last Auth: Counter = 5] --> B[New Auth: Counter = 6]
    B --> C{Counter Check}

    C -->|6 > 5| D[✅ Valid: Update to 6]
    C -->|6 <= 5| E[❌ Security Alert!]

    E --> F[Possible Scenarios:]
    F --> G[1. Cloned Credential]
    F --> H[2. Replay Attack]
    F --> I[3. Authenticator Rollback]

    E --> J[Actions:]
    J --> K[Revoke Credential]
    J --> L[Alert User]
    J --> M[Require Re-registration]

    style D fill:#ccffcc
    style E fill:#ffcccc
    style K fill:#ffcccc
    style L fill:#ffffcc
    style M fill:#ffffcc
```

## Complete Authentication State Machine

```mermaid
stateDiagram-v2
    [*] --> Unauthenticated

    Unauthenticated --> InitiateLogin: User clicks login
    InitiateLogin --> ChallengeGenerated: Server creates challenge
    ChallengeGenerated --> AwaitingUser: Show authenticator prompt

    AwaitingUser --> UserVerified: User provides biometric/PIN
    AwaitingUser --> Cancelled: User cancels
    Cancelled --> Unauthenticated

    UserVerified --> SignatureCreated: Authenticator signs challenge
    SignatureCreated --> VerifyingSignature: Server receives assertion

    VerifyingSignature --> CheckOrigin: Challenge valid
    VerifyingSignature --> Failed: Invalid challenge
    CheckOrigin --> CheckCounter: Origin valid
    CheckOrigin --> Failed: Wrong origin
    CheckCounter --> Authenticated: Counter increased
    CheckCounter --> Failed: Counter issue

    Authenticated --> SessionActive: Create session
    SessionActive --> SessionExpired: Timeout
    SessionActive --> LoggedOut: User logs out
    SessionExpired --> Unauthenticated
    LoggedOut --> Unauthenticated
    Failed --> Unauthenticated

    note right of Failed
        Possible reasons:
        - Invalid signature
        - Wrong origin
        - Expired challenge
        - Counter anomaly
    end note

    note right of SessionActive
        User can access
        protected resources
    end note
```

## Password vs Passkey Comparison

```mermaid
graph TB
    subgraph Traditional Password Login
        P1[User enters password] --> P2[Send over network]
        P2 --> P3[Server compares hash]
        P3 --> P4{Match?}
        P4 -->|Yes| P5[Authenticated]
        P4 -->|No| P6[Failed]

        style P2 fill:#ffcccc
        Note1[⚠️ Password transmitted<br/>⚠️ Can be phished<br/>⚠️ Can be guessed]
    end

    subgraph WebAuthn Passkey Login
        W1[User provides biometric] --> W2[Authenticator signs challenge]
        W2 --> W3[Send signature only]
        W3 --> W4[Server verifies signature]
        W4 --> W5{Valid?}
        W5 -->|Yes| W6[Authenticated]
        W5 -->|No| W7[Failed]

        style W3 fill:#ccffcc
        Note2[✅ No password sent<br/>✅ Phishing impossible<br/>✅ Can't be guessed]
    end
```

## Troubleshooting Common Issues

```mermaid
graph TD
    Issue[Authentication Failed] --> Q1{Error Message?}

    Q1 -->|No credentials found| S1[Solutions:<br/>1. User not registered<br/>2. Wrong username<br/>3. Credential deleted]

    Q1 -->|Challenge mismatch| S2[Solutions:<br/>1. Challenge expired<br/>2. Session lost<br/>3. Server restart]

    Q1 -->|Origin mismatch| S3[Solutions:<br/>1. Check rpId config<br/>2. Check CORS<br/>3. Verify domain]

    Q1 -->|Invalid signature| S4[Solutions:<br/>1. Wrong public key<br/>2. Corrupted data<br/>3. Check credential_id]

    Q1 -->|User cancelled| S5[Solutions:<br/>1. User dismissed prompt<br/>2. Try again<br/>3. Not an error!]

    style S1 fill:#ffffcc
    style S2 fill:#ffffcc
    style S3 fill:#ffffcc
    style S4 fill:#ffffcc
    style S5 fill:#ccffcc
```

## How to View These Diagrams

### Recommended: VS Code Extension

1. **Install extension:**
   - Open VS Code
   - Go to Extensions (Cmd+Shift+X)
   - Search for "Markdown Preview Mermaid Support"
   - Install it

2. **View the diagram:**
   - Open this file in VS Code
   - Press `Cmd+Shift+V` (Mac) or `Ctrl+Shift+V` (Windows/Linux)
   - See beautiful rendered diagrams!

### Alternative: Online Viewer

1. Go to https://mermaid.live
2. Copy any mermaid code block from this file
3. Paste into the editor
4. View interactive diagram
5. Export as PNG/SVG if desired

### Alternative: GitHub

1. Push this file to a GitHub repository
2. View on GitHub (automatic Mermaid rendering)

## Next Steps

Now that you understand authentication:

1. ✅ Understand registration flow
2. ✅ Understand authentication flow
3. ⬜ Design database schema
4. ⬜ Implement backend endpoints
5. ⬜ Build frontend components
6. ⬜ Test the complete flow

Ready to start implementing? Let me know!
