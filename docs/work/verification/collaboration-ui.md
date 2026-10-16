# C2 browser access integration

Status: implementation in progress; production unchanged. [Plan](../../superpowers/plans/2026-10-16-collaboration-dashboard.md).

Before implementation: public link exchange/read uses an isolated HTTP client with no automatic login bearer or global 401 logout interceptor. Passwords are POST body only. A failed public password request must preserve the ordinary login token and user store. Existing legacy public read remains compatible through POST /api/v1/public/scenes/{token}; newer guest edit uses POST /api/v1/public/scene-session, then room-token and B3 provider. Never store scene credentials as auth_token.

Controller owns the initial public client/read-view change. Remaining C2 invite/member/settings/share/role/route UI follows A APIs and native scene styling; separate acceptance tests must cover complete flows. Planned regression: actual Axios adapter receives POST JSON without Authorization/password query, rejects401, and login token remains. Existing public view browser fixture moves to POST; public password challenge/retry is verified in Chromium. Backend A3 already enforces POST.

Controller regression observed RED in Chromium: initial401 cleared auth_token, requests usedGET/query and attached login bearer. After isolated publicApi + POST adoption, all three focused public viewer/password tests pass (2.3s). Fixture renders the actual frontend; API is mocked, so this is browser transport proof, not fullstack authorization proof. BackendA3 tests separately cover POST contract. No new unit test mirrors the one-line transport wrapper. Production frontend build passed (6.31s), with the existing chunk-size warning. Browser transport subset implemented; full C2 task and review remain pending.
