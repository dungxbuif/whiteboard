# I1 domain verification evidence — 2026-10-20

## Automated fixture evidence

- Go use-case tests cover IDNA normalization, the exact `_whiteboard-verification` lookup name, 32-byte challenge construction, 24-hour expiry, global exact-domain conflict, consumed challenge replay, capture-time TXT revalidation, expiry at capture, explicit capture with `dataMoved: false`, ownership rechecks inside every mutation lock, lost-response create replay with the same TXT challenge, and rejection when one request ID is rebound to another principal, action, or canonical body. The focused run passed 16 tests.
- DNS adapter test forces a resolver deadline and confirms timeout failure.
- Strict HTTP body tests passed six malformed-body cases covering unknown fields, trailing values, and the 8 KiB limit without invoking the use case. A handler retry test also proves the unchanged client request ID reaches the use case and yields an identical response at the HTTP boundary.
- Guarded PostgreSQL tests passed against the disposable local `devtool_a1_test` database, including migration up/down and receipt-column checks, active/verified/captured protection, expired-pending transfer, a two-workspace reclaim race with exactly one winner, and a durable encrypted response receipt. The receipt test simulates timeout after commit, proves callback suppression and identical replay, rejects principal/action/body rebinding, and confirms the plaintext TXT secret is absent from ciphertext.
- The full backend suite passed across all packages with the disposable PostgreSQL integration database enabled.
- Frontend component tests passed for one-time TXT display, explicit post-verification capture, retaining the request ID after an uncertain timeout, and retiring it after a definitive response.
- The TypeScript 7 production build passed after the complete concurrent frontend changes settled.
- The domain real-stack scenario passed against the disposable database. It proves migration persistence, owner denial, fail-closed capture when public TXT is absent, unchanged verified state after that failure, captured-state UI shape, and user-free discovery. Its captured-state setup is explicitly a database fixture; successful current-TXT capture remains injected-resolver evidence rather than public-DNS evidence.

Successful DNS verification and capture use an injected resolver fixture. They are not evidence of a public authoritative DNS publication.

## Operator-dependent production UAT

After backend route wiring and deployment, an owner must create a challenge for a controlled production domain, publish the exact TXT record in public DNS, verify it through the UI, explicitly capture it, then revoke it. Record the authoritative DNS response and HTTP request IDs in the deployment evidence. This step remains open because public DNS publication is external operator input.
