# Project AGENT: Excalidraw Transformation

Welcome to the Excalidraw Transformation project. This repository uses the **repository-harness** to manage agent workflows.

## 1. Quick Links
- **Product Spec:** [docs/product/SPEC.md](docs/product/SPEC.md)
- **Architecture:** [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- **Current Backlog:** [docs/stories/backlog.md](docs/stories/backlog.md)
- **Phase 1 Plans:** [docs/stories/phase-01/](docs/stories/phase-01/)

## 2. Current State
We have completed planning for Phase 1-5. 
- **Phase 1:** Core Dashboard & BE Rewrite (Implementation Ready).
- **Phase 2:** Presentation Mode.
- **Phase 3:** Shareable Production.
- **Phase 4:** Real-time Collaboration.
- **Phase 5:** AI Features.

## 3. Working with this Repo
1. Always start by reading [docs/HARNESS.md](docs/HARNESS.md). When working on any ticket, always remember and adhere strictly to the constraints and workflows in [docs/HARNESS.md](docs/HARNESS.md).
2. Follow the [Clean Architecture](docs/ARCHITECTURE.md) mandates for Go.
3. Use **Zustand** for React state management.
4. Refer to `docs/stories/phase-01/` for executable plans.
5. **Branch Checkout**: Always checkout a new/appropriate branch before implementing code.
6. **Tests & Verification**: When reporting completion, you must write tests and self-verify the completed task (including running linter, ensuring builds pass, and verifying tests).

## 4. Project Engineering Standards
This section mandates the architectural, code quality, and verification standards for the project. All agents must adhere to these rules strictly.

### 4.1. Code Quality & Refactoring
- **No Magic Values**: Never use magic strings or magic numbers. All literals (timeouts, URLs, header names, context keys, etc.) must be defined as named constants in `internal/domain/constants.go` or localized constants within the package.
- **Strict Configuration**: Environment variable loading must be defensive. Use a `loadConfig` pattern that **exits the application immediately** with a clear error message if a required variable is missing or invalid. Never use insecure fallbacks for credentials or security-critical settings.
- **No Redundant Comments**: Do not add comments for obvious logic. Use comments only to explain complex algorithms, business domain reasoning, or as part of formal documentation (e.g., Go doc comments).

### 4.2. Frontend Verification
- **Automated Verification**: Every ticket involving frontend changes **MUST** be verified using automated tools.
- **Playwright/MCP Requirement**: Use the `mcp-playwright-browser` tools to perform end-to-end verification of UI components, user flows, and state changes. Capturing screenshots and comparing them against design concepts is mandated for high-fidelity work.

### 4.3. Backend Architecture
- **Clean Architecture**: Follow the `Repository -> Usecase -> Handler` pattern.
- **Validation**: All DTOs and input structures must be validated using `go-playground/validator/v10`.
- **Database**: Use **GORM** for ORM operations and **Casbin v3** for authorization logic.

### 4.4. Documentation
- **Story Status**: Always update the story's status in its dedicated markdown file (`docs/stories/phase-XX/EXC-XXX.md`) upon completion.
- **Detail Design**: High-risk or complex features require a detailed design document approved by a reviewer before implementation starts.

---
*Reference Research:* [docs/research/](docs/research/)
