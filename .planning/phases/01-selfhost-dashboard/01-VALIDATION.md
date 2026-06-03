# Phase 1 Validation Architecture: Self-hosted Dashboard Rewrite

## 1. Testing Strategy
This phase ensures the integrity of the rewrite by validating Go Clean Architecture, React Vite dashboard integration, and Multi-tenant database isolation.

## 2. Validation Layers

### A. Backend (Go)
- **Unit Tests:** Validate logic in `internal/domain` and `internal/usecase` using `testing` package.
- **Integration Tests:** Use `testcontainers-go` or a local Postgres instance to verify `internal/repository` SQL and Materialized Path logic.
- **Middleware Check:** Verify `WorkspaceID` injection into context.

### B. Frontend (React)
- **Component Tests:** Verify Shadcn ports and Dashboard layout.
- **Store Tests:** Validate Zustand state transitions for folders and whiteboard sync.
- **OAuth Mocking:** Verify redirection logic for Google login.

### C. System Integration
- **Docker Simulation:** Verify service discovery and Nginx routing as defined in `references/nginx/integrated.conf`.

## 3. Success Criteria (Final Check)
- [ ] Backend starts and connects to Postgres/S3.
- [ ] User can log in via Google OAuth.
- [ ] Workspace creation and switching work without data leakage.
- [ ] Nested folders can be created and moved (Materialized Path update).
- [ ] Whiteboard state is debounced-synced to DB and assets to S3.

