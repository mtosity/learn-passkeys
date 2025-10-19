# Passkeys Frontend

React frontend for the passkey authentication learning project.

## Tech Stack

- React 18 with TypeScript
- Vite for build tooling
- shadcn/ui for UI components
- TanStack Query (React Query) for data fetching
- React Router for routing
- @simplewebauthn/browser for WebAuthn client

## Getting Started

### Prerequisites

- Node.js 18+ installed
- Backend server running on http://localhost:8080

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The app will be available at http://localhost:5173

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

- `VITE_API_URL` - Backend API URL (default: http://localhost:8080)

## Pages

- `/login` - Login with passkey
- `/register` - Register new user with passkey
- `/secret` - Protected page showing secret message (requires authentication)

## Building for Production

```bash
npm run build
```

For production deployment, make sure to:
1. Set `VITE_API_URL` to your production backend URL
2. Configure CORS in the backend to allow your frontend domain
3. Ensure HTTPS is enabled (required for WebAuthn in production)
