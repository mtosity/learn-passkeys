# WebAuthn Registration Flow - Mermaid Diagram

## Interactive Sequence Diagram

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Server
    participant Authenticator

    Note over User,Authenticator: Registration Ceremony Begins

    User->>Browser: 1. Clicks "Register" button
    Browser->>Server: 2. POST /register/begin (username)

    Note over Server: Generate random challenge (32 bytes)
    Note over Server: Create or lookup user in DB

    Server-->>Browser: 3. Registration options (JSON)
    Note right of Server: {challenge, rp, user, pubKeyCredParams}

    Browser->>Browser: 4. Parse options
    Browser->>Authenticator: 5. navigator.credentials.create()

    Note over Authenticator: Display prompt to user
    Authenticator->>User: 6. "Touch ID to continue..."
    User->>Authenticator: 7. Provides biometric/PIN

    Note over Authenticator: Verify user identity
    Note over Authenticator: Generate key pair<br/>Private key stays here!
    Note over Authenticator: Create attestation object

    Authenticator-->>Browser: 8. Return credential
    Note right of Authenticator: {id, rawId, response: {<br/>clientDataJSON,<br/>attestationObject}}

    Browser->>Server: 9. POST /register/finish (credential)

    Note over Server: Verify challenge matches
    Note over Server: Verify origin matches
    Note over Server: Parse attestation object
    Note over Server: Extract public key

    Server->>Server: 10. Store in database
    Note right of Server: - credential_id<br/>- public_key<br/>- counter<br/>- user_id

    Server-->>Browser: 11. Success response
    Browser->>User: 12. Show success message

    Note over User,Authenticator: Registration Complete!
```

## Key Steps Explained

### Steps 1-3: Initiation
- User wants to create an account
- Server generates a unique, random challenge
- Server sends registration options to browser

### Steps 4-7: User Verification
- Browser invokes WebAuthn API
- Authenticator prompts user (Touch ID, Face ID, PIN, etc.)
- User proves their identity

### Steps 8-9: Credential Creation
- Authenticator generates a brand new key pair
- **Private key NEVER leaves the authenticator**
- Public key is packaged with other data
- Browser sends credential to server

### Steps 10-12: Server Processing
- Server verifies the challenge (prevent replay attacks)
- Server verifies the origin (prevent phishing)
- Server extracts and stores the public key
- User is now registered!

## What Gets Stored

```mermaid
erDiagram
    USERS ||--o{ CREDENTIALS : has
    USERS {
        uuid id PK
        string username
        string display_name
        timestamp created_at
    }
    CREDENTIALS {
        uuid id PK
        uuid user_id FK
        bytea credential_id
        bytea public_key
        int counter
        text[] transports
        timestamp created_at
    }
```

## Security Highlights

```mermaid
graph TB
    A[Registration Request] --> B{Verify Challenge}
    B -->|Invalid| C[Reject ❌]
    B -->|Valid| D{Verify Origin}
    D -->|Wrong Origin| C
    D -->|Correct| E{Parse Attestation}
    E -->|Invalid Format| C
    E -->|Valid| F[Extract Public Key]
    F --> G[Store in Database]
    G --> H[Success ✅]

    style C fill:#ffcccc
    style H fill:#ccffcc
```

## Data Flow Visualization

```mermaid
flowchart LR
    A[Random<br/>Challenge] --> B[Browser]
    B --> C[Authenticator]
    C --> D[Key Pair<br/>Generated]
    D --> E[Private Key<br/>Stays on Device]
    D --> F[Public Key]
    F --> G[Server]
    G --> H[(Database)]

    style E fill:#ffcccc,stroke:#ff0000,stroke-width:3px
    style F fill:#ccffcc
    style H fill:#cce5ff
```

## Common Errors During Registration

```mermaid
graph TD
    Start[Registration Attempt] --> Q1{HTTPS or localhost?}
    Q1 -->|No| E1[Error: Secure context required]
    Q1 -->|Yes| Q2{Challenge valid?}
    Q2 -->|No| E2[Error: Challenge mismatch]
    Q2 -->|Yes| Q3{Origin matches RP ID?}
    Q3 -->|No| E3[Error: Origin mismatch]
    Q3 -->|Yes| Q4{User verified?}
    Q4 -->|No| E4[Error: User cancelled]
    Q4 -->|Yes| Q5{Authenticator available?}
    Q5 -->|No| E5[Error: No authenticator found]
    Q5 -->|Yes| Success[✅ Registration Successful]

    style E1 fill:#ffcccc
    style E2 fill:#ffcccc
    style E3 fill:#ffcccc
    style E4 fill:#ffcccc
    style E5 fill:#ffcccc
    style Success fill:#ccffcc
```

## How to View This Diagram

### Option 1: VS Code
1. Install extension: "Markdown Preview Mermaid Support"
2. Open this file
3. Press `Cmd+Shift+V` (Mac) or `Ctrl+Shift+V` (Windows/Linux)
4. See beautiful rendered diagrams!

### Option 2: GitHub
1. Push this file to GitHub
2. GitHub automatically renders Mermaid diagrams
3. View in browser

### Option 3: Online
1. Copy the mermaid code blocks
2. Go to https://mermaid.live
3. Paste and view/edit interactively
4. Export as PNG/SVG if desired

### Option 4: Obsidian
1. Open this file in Obsidian
2. Mermaid diagrams render automatically
3. Great for note-taking while learning!

## Next Steps

After understanding registration:
1. Read the authentication flow diagram
2. Understand how the stored public key is used
3. See how sessions are created after successful auth
4. Start implementing the backend!
