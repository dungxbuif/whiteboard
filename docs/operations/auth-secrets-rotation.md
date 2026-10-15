# Authentication Secret Rotation

Google OAuth and JWT configuration are required at runtime. Before deploying
the rewritten app:

1. Create a new Google OAuth web client and set its callback to
   `GOOGLE_REDIRECT_URL` (for example,
   `https://whiteboard.example.com/api/v1/auth/google/callback`).
2. Revoke the old Google client secret and store the replacement as
   `GOOGLE_CLIENT_SECRET`.
3. Generate a new high-entropy `JWT_SECRET` and update every backend instance
   atomically. Existing JWTs will expire and users must sign in again.
4. Set `FRONTEND_URL` to the canonical HTTPS SPA origin and verify the Google
   client allow-list contains that callback.
5. Restart the backend, complete a real Google sign-in, and verify `/users/me`
   and one workspace request with the new session.

Do not commit `src/.env` or paste secret values into tickets, logs, or browser
URLs. The OAuth callback only places the short-lived JWT in the frontend hash;
the SPA removes the hash after storing the token.
