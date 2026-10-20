# L2 native library verification

Status: implementation and root integration complete in `dc9f3a1`. Production
provider/storage and public browser UAT remain release gates.

Required proof:

- strict v2 validation rejects unsupported, oversized, unsafe or unresolved data;
- every element/group/file/frame/container/binding reference is freshly remapped;
- immutable revisions and asset pins commit atomically under workspace locks;
- source and target authorization are rechecked for every insert and replay;
- one expected-sequence collaboration write produces one durable receipt;
- uncertain exact retries do not duplicate elements or staged assets;
- import -> insert -> reload -> checkpoint -> export preserves native content;
- revoked/cross-workspace access and asset pin races fail closed.

Focused evidence on 2026-10-20:

- full Go suite: 345 passed across 14 packages;
- full diagram-kit suite: 43 passed;
- full collaboration runtime suite: 210 passed with the guarded loopback
  database (including recovery and durable library insertion/replay);
- full frontend unit suite: 199 passed;
- focused Go library use-case tests: 5 passed;
- Go handler boundary test: 1 passed;
- guarded PostgreSQL migration and asset-stage race tests: passed against a
  loopback disposable `*_test` database;
- frontend library API/panel tests: 3 passed;
- diagram-kit and collaboration TypeScript builds: passed;
- frontend production build: passed.

The browser real-stack case is compiled and remains gated on integration wiring.
Fixture evidence is not public storage UAT.
