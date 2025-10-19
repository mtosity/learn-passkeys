# Quick Start Guide

This guide will help you get the passkey authentication app running locally.

## Prerequisites

- Docker and Docker Compose (for PostgreSQL)
- Go 1.21+ (for backend)
- Node.js 18+ (for frontend)

## Step 1: Start the Database

```bash
docker-compose up -d
```

Verify it's running:
```bash
docker-compose ps
```

## Step 2: Start the Backend

```bash
cd backend
go run main.go
```

The backend will be available at http://localhost:8080

## Step 3: Start the Frontend

In a new terminal:

```bash
cd frontend
npm install
npm run dev
```

The frontend will be available at http://localhost:5173

## Step 4: Test the Application

1. Open http://localhost:5173 in your browser
2. Click "Register" to create a new account with a passkey
3. Enter a username and click "Register with Passkey"
4. Your browser will prompt you to create a passkey (use Touch ID, Face ID, or security key)
5. After registration, you'll be redirected to login
6. Enter your username and click "Login with Passkey"
7. Authenticate with your passkey
8. You should see the secret message: "O Captain! My Captain!"

## Configuration for Production

### Backend

Create a `.env` file in the `backend/` directory (see `.env.example`):

```bash
# For production
RP_ID=yourdomain.com
RP_ORIGINS=https://yourdomain.com
ALLOWED_ORIGINS=https://yourdomain.com
```

### Frontend

Update `.env` in the `frontend/` directory:

```bash
VITE_API_URL=https://api.yourdomain.com
```

## Troubleshooting

### CORS Issues
- Make sure the backend `ALLOWED_ORIGINS` includes your frontend URL
- Make sure the backend `RP_ORIGINS` matches your frontend domain

### WebAuthn Not Working
- WebAuthn requires HTTPS in production (localhost is exempt)
- Check browser console for error messages
- Ensure your authenticator supports WebAuthn (most modern devices do)

### Database Connection Issues
- Verify PostgreSQL is running: `docker-compose ps`
- Check database credentials in backend code match docker-compose.yml
