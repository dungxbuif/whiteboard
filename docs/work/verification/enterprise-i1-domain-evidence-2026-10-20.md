# I1 domain verification evidence — 2026-10-20

## Automated fixture evidence

- Go use-case tests cover IDNA normalization, the exact `_whiteboard-verification` lookup name, 32-byte challenge construction, 24-hour expiry, global exact-domain conflict, consumed challenge replay, explicit capture with `dataMoved: false`, and owner denial.
- DNS adapter test forces a resolver deadline and confirms timeout failure.
- The full backend suite passed with 259 tests across 13 packages after root router/server integration.
- Frontend component tests passed for one-time TXT display and explicit post-verification capture.
- The TypeScript 7 production build passed after the complete concurrent frontend changes settled.
- A domain real-stack scenario covers migration persistence, owner denial, the explicit UI capture, and user-free discovery. Local execution is environment-blocked before setup because PostgreSQL role `test` is absent; it does not count as passed evidence.

These DNS results use an injected resolver fixture. They are not evidence of a public authoritative DNS publication.

## Operator-dependent production UAT

After backend route wiring and deployment, an owner must create a challenge for a controlled production domain, publish the exact TXT record in public DNS, verify it through the UI, explicitly capture it, then revoke it. Record the authoritative DNS response and HTTP request IDs in the deployment evidence. This step remains open because public DNS publication is external operator input.
