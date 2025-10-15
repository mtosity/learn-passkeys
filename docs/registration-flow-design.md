# Registration Flow Design Decision

## The Question

When should we create the user record in the database?

1. **During `BeginRegistration`** (before passkey is created)?
2. **During `FinishRegistration`** (after passkey is verified)?

## Current Implementation: Approach 1 (Create User Early)

### Flow:
```
BeginRegistration:
  1. User submits username
  2. Create user record in database
  3. Generate WebAuthn challenge
  4. Store challenge (linked to user_id)
  5. Return options to client

FinishRegistration:
  1. Verify credential
  2. Save credential (linked to user_id)
```

### Pros:
- Simpler implementation
- User ID immediately available for foreign key relationships
- Challenge can reference user_id directly

### Cons:
- **Orphaned users**: Users created but never complete registration
  - User abandons registration flow
  - Registration fails on client side
  - Browser doesn't support WebAuthn
  - User closes tab before completing
- Database pollution over time
- Need cleanup mechanism for abandoned users

### Database Schema (Current):
```sql
CREATE TABLE challenges (
    challenge BYTEA PRIMARY KEY,
    user_id UUID REFERENCES users(id),  -- User must exist
    type VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP NOT NULL
);
```

---

## Alternative: Approach 2 (Create User After Success)

### Flow:
```
BeginRegistration:
  1. User submits username
  2. Check username availability (don't create user yet)
  3. Generate WebAuthn challenge
  4. Store challenge with username (not user_id)
  5. Return options to client

FinishRegistration:
  1. Verify credential
  2. Create user record
  3. Save credential (linked to new user_id)
```

### Pros:
- **No orphaned users**: Only creates users who successfully register
- Cleaner database
- No need for cleanup mechanisms
- Better data integrity

### Cons:
- Slightly more complex implementation
- Need to handle both username and user_id in challenges table
- Username collision possible between begin/finish (race condition)

### Database Schema (Modified):
```sql
CREATE TABLE challenges (
    challenge BYTEA PRIMARY KEY,
    user_id UUID REFERENCES users(id),      -- Nullable, for login
    username VARCHAR(255),                   -- For registration
    type VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    CHECK (
        (type = 'registration' AND username IS NOT NULL) OR
        (type = 'authentication' AND user_id IS NOT NULL)
    )
);
```

---

## Recommendation for Production

**Use Approach 2** (create user after success) because:

1. **Data Quality**: Only complete registrations create records
2. **Scale**: As traffic grows, abandoned registrations become a significant issue
3. **Privacy**: Don't store user information until they complete registration
4. **Compliance**: Some regulations require minimizing data collection

---

## Implementation Notes

### Current Implementation Issues:
- ✅ Works functionally
- ⚠️ Will accumulate incomplete registrations
- 🔧 Need periodic cleanup job to delete users without credentials

### Migration Path to Approach 2:

1. Modify challenges table:
   ```sql
   ALTER TABLE challenges
     ALTER COLUMN user_id DROP NOT NULL,
     ADD COLUMN username VARCHAR(255);
   ```

2. Update `BeginRegistration`:
   - Check username availability (SELECT to verify not taken)
   - Don't create user record
   - Store username in challenges table

3. Update `FinishRegistration`:
   - Retrieve challenge by challenge value (get username)
   - Create user record with username
   - Save credential with new user_id
   - Handle race condition (username taken between begin/finish)

### Race Condition Handling:
If username is taken between begin and finish:
- Option A: Return error, user must start over with new username
- Option B: Generate unique username variant (username_123)
- Option C: Use transaction to check + create atomically

---

## For This Learning Project

**Current approach (Approach 1) is acceptable** because:
- Simpler to understand and implement
- Low traffic (local development)
- Focus is on learning WebAuthn, not production optimization
- Easy to refactor later if desired

But it's important to understand the trade-offs for real-world applications.
