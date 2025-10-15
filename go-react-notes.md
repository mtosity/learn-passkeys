# Go + React Setup Notes

## Important Concepts for Go + React + WebAuthn

### CORS Configuration
Since your frontend (React on port 5173) and backend (Go on port 8080) run on different ports, you MUST configure CORS properly:

**In your Go backend:**
```go
// Enable CORS with credentials
w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
w.Header().Set("Access-Control-Allow-Credentials", "true")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
```

**In your React frontend (fetch calls):**
```javascript
// Include credentials in fetch requests
fetch('http://localhost:8080/api/register/begin', {
  method: 'POST',
  credentials: 'include',  // CRITICAL for sessions
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(data)
})
```

### Go WebAuthn Library Requirements

The `go-webauthn/webauthn` library requires your User model to implement the `webauthn.User` interface:

```go
type User interface {
    WebAuthnID() []byte
    WebAuthnName() string
    WebAuthnDisplayName() string
    WebAuthnIcon() string
    WebAuthnCredentials() []Credential
}
```

Your User struct must have methods that satisfy this interface.

### Session Management in Go

You'll need to store challenges temporarily during registration/authentication. Options:
1. **In-memory map** (simple but doesn't scale, fine for learning)
2. **gorilla/sessions** (cookie-based sessions)
3. **PostgreSQL** (store challenges in a table with expiration)

Sessions are CRITICAL because:
- Registration: Store challenge between `/register/begin` and `/register/finish`
- Authentication: Store challenge between `/login/begin` and `/login/finish`

### React + Vite Proxy (Optional Alternative to CORS)

Instead of dealing with CORS, you can proxy API requests in development:

**In `vite.config.js`:**
```javascript
export default {
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
}
```

Then in React, you call `/api/register/begin` instead of `http://localhost:8080/api/register/begin`

### Typical API Endpoints Structure

**Registration Flow:**
- `POST /api/register/begin` - Input: `{username}`, Output: WebAuthn creation options
- `POST /api/register/finish` - Input: credential from browser, Output: success/failure

**Authentication Flow:**
- `POST /api/login/begin` - Input: `{username}`, Output: WebAuthn request options
- `POST /api/login/finish` - Input: assertion from browser, Output: success/failure + session

**Protected Routes:**
- `GET /api/secret` - Requires valid session, returns secret message
- `POST /api/logout` - Clears session

**Health Check:**
- `GET /api/health` - Simple endpoint to test if backend is running

### React Component Structure

**Suggested components:**
- `App.jsx` - Router and main layout
- `Register.jsx` - Registration form and WebAuthn flow
- `Login.jsx` - Login form and WebAuthn flow
- `Secret.jsx` - Protected page showing secret message
- `webauthn.js` - Helper functions for calling WebAuthn APIs

### Data Flow Example

**Registration:**
1. User enters username in React form
2. React calls `POST /api/register/begin` with username
3. Go generates challenge, stores in session, returns options
4. React calls `navigator.credentials.create()` with options
5. Browser prompts for biometric/PIN
6. React sends credential to `POST /api/register/finish`
7. Go verifies, stores credential in PostgreSQL
8. React shows success message

**Login:**
1. User enters username in React form
2. React calls `POST /api/login/begin` with username
3. Go generates challenge, retrieves user's credentials, stores challenge in session
4. React calls `navigator.credentials.get()` with options
5. Browser prompts for biometric/PIN
6. React sends assertion to `POST /api/login/finish`
7. Go verifies signature, creates session
8. React redirects to secret page

### Common Go Gotchas

1. **Byte slices vs strings**: WebAuthn works with `[]byte`, but PostgreSQL and JSON often use strings. You'll do lots of base64 encoding/decoding.

2. **Struct tags**: Use JSON tags for API responses:
   ```go
   type User struct {
       ID       int    `json:"id"`
       Username string `json:"username"`
   }
   ```

3. **Error handling**: Go requires explicit error checking. Don't ignore errors:
   ```go
   if err != nil {
       http.Error(w, err.Error(), http.StatusInternalServerError)
       return
   }
   ```

4. **Pointer receivers**: Methods implementing the webauthn.User interface should use pointer receivers:
   ```go
   func (u *User) WebAuthnID() []byte {
       return []byte(u.ID)
   }
   ```

### Common React Gotchas

1. **Base64URL encoding**: The `@simplewebauthn/browser` library handles this for you. Don't manually encode/decode.

2. **State management**: Use `useState` for form data and authentication status:
   ```javascript
   const [username, setUsername] = useState('')
   const [isAuthenticated, setIsAuthenticated] = useState(false)
   ```

3. **Effect for auth check**: Use `useEffect` to check if user is already authenticated on page load:
   ```javascript
   useEffect(() => {
     // Check if session exists
     fetch('/api/check-auth', { credentials: 'include' })
       .then(res => res.json())
       .then(data => setIsAuthenticated(data.authenticated))
   }, [])
   ```

4. **Error handling**: WebAuthn can fail in many ways (user cancels, no authenticator, etc.). Handle gracefully.

### Development Workflow

1. Start PostgreSQL: `docker-compose up -d`
2. Start Go backend: `cd backend && go run main.go`
3. Start React frontend: `cd frontend && npm run dev`
4. Open http://localhost:5173 in browser
5. Check both terminal outputs for errors

### Testing Tips

1. **Use Chrome DevTools**: Console shows WebAuthn errors clearly
2. **Check Network tab**: See request/response payloads
3. **Test with different authenticators**: Touch ID, security key, etc.
4. **Clear browser data**: Sometimes credentials get cached
5. **Check PostgreSQL**: Verify data is being stored correctly

### PostgreSQL Schema Considerations

**Users table:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Credentials table:**
```sql
CREATE TABLE credentials (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA UNIQUE NOT NULL,  -- Store as binary
    public_key BYTEA NOT NULL,            -- Store as binary
    counter BIGINT DEFAULT 0,
    transports TEXT[],                    -- Array of strings
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Sessions/Challenges table (optional):**
```sql
CREATE TABLE challenges (
    id SERIAL PRIMARY KEY,
    challenge BYTEA NOT NULL,
    user_id INTEGER REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Security Checklist

- [ ] CORS configured correctly (specific origin, not "*")
- [ ] Credentials: true in fetch calls
- [ ] HTTPS in production (localhost is OK for development)
- [ ] Challenges are cryptographically random
- [ ] Challenges expire (5 minutes max)
- [ ] Challenges are single-use (delete after verification)
- [ ] Origin validation in WebAuthn verification
- [ ] SQL injection prevention (use parameterized queries)
- [ ] Session security (httpOnly cookies if using cookies)
