# Authentication Secret Rotation

JWT configuration is required at runtime. Before deploying the rewritten app:

1. Generate a new high-entropy `JWT_SECRET` and update every backend instance
   atomically. Existing JWTs will expire and users must sign in again.
2. Set `FRONTEND_URL` to the canonical HTTPS SPA origin.
3. Restart the backend, provision or rotate a user with `userctl`, and verify
   `/api/v1/auth/login`, `/users/me`, and one workspace request with the new session.
4. Rotate the Coturn `TURN_AUTH_SECRET` together with the VPS Coturn
   `static-auth-secret`, restart the backend, and complete a voice allocation
   from an external network. Existing TURN credentials expire within one hour.

Do not commit `src/.env` or paste secret values into tickets, logs, or browser
URLs. The CLI accepts passwords only through its process invocation and stores
only bcrypt hashes in `users.password_hash`.
