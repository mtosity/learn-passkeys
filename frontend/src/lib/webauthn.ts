import { startRegistration, startAuthentication } from '@simplewebauthn/browser';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export async function registerUser(username: string) {
  // Step 1: Begin registration - get challenge from server
  const beginResponse = await fetch(`${API_URL}/register/begin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username }),
  });

  if (!beginResponse.ok) {
    const error = await beginResponse.text();
    throw new Error(`Registration failed: ${error}`);
  }

  const options = await beginResponse.json();

  // Step 2: Use WebAuthn to create credential
  const credential = await startRegistration(options);

  // Step 3: Send credential to server for verification
  const finishResponse = await fetch(`${API_URL}/register/finish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credential),
  });

  if (!finishResponse.ok) {
    const error = await finishResponse.text();
    throw new Error(`Registration verification failed: ${error}`);
  }

  return await finishResponse.json();
}

export async function loginUser(username: string) {
  // Step 1: Begin login - get challenge from server
  const beginResponse = await fetch(`${API_URL}/login/begin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username }),
  });

  if (!beginResponse.ok) {
    const error = await beginResponse.text();
    throw new Error(`Login failed: ${error}`);
  }

  const options = await beginResponse.json();

  // Step 2: Use WebAuthn to sign challenge
  const assertion = await startAuthentication(options);

  // Step 3: Send assertion to server for verification
  const finishResponse = await fetch(`${API_URL}/login/finish`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(assertion),
  });

  if (!finishResponse.ok) {
    const error = await finishResponse.text();
    throw new Error(`Login verification failed: ${error}`);
  }

  return await finishResponse.json();
}
