# VM100 Compose tmpfs compatibility — 2026-10-19

Status: implementation pending.

During the first collaboration-v2 production create, Docker Compose v5.6.0 rejected `mode=1777` as an independent mount path. The YAML flow-sequence form `tmpfs: [/tmp:size=64m,mode=1777]` treats the comma as a list delimiter, producing two tmpfs entries even though `docker compose config --quiet` accepts the file. Backend creation completed first; collab/MCP creation stopped before either service ran.

The intended runtime contract remains one bounded tmpfs mounted at `/tmp` for each read-only application container. Change the three flow-sequence declarations to block-sequence scalar declarations so the entire `/tmp:size=...,mode=1777` value reaches Docker as one entry. Apply the same correction to the Homelab VM100 template.

Verification must first retain the minimal failing reproduction, then prove the block scalar creates a container whose `HostConfig.Tmpfs` contains exactly `/tmp` with the requested size and mode. Run Compose config validation, create the real collab/MCP/frontend services, inspect their tmpfs maps, bring the stack healthy and repeat public health checks. The change affects packaging only; API, schema, permissions and user-facing behavior do not change, so public product and agent API documentation require no behavioral update.
