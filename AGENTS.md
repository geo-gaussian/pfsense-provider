# 🤖 AGENTS.md — AI Coding Agent Guidelines

## 🎯 Project Context

This repository implements an experimental Terraform provider in Go, published
as `geo-gaussian/pfsense-nexus`. It wraps the Nexus API described in
`docs/nexus.openapi.yaml`. Alias listing has been tested on one appliance;
write operations and compatibility with other versions remain unverified.

## 📐 Architectural Rules & Constraints

1. **Never Bypass `NexusClient.DoRequest`:** All HTTP calls to pfSense MUST route through `internal/client.NexusClient.DoRequest()`. Its mutex serializes HTTP requests per configured client, not across separate clients or Terraform processes.
2. **Preserve Generated Files vs. Custom Logic:**
   - Files emitted by `tfplugingen-framework` into `internal/provider/` define schemas and models. When regenerating from `generator_config.yml`, verify that custom `Create`/`Read`/`Update`/`Delete` wiring and `ImportState` methods are preserved.
3. **Explicit Apply / Reload Lifecycle:**
   - If an endpoint in `docs/nexus.openapi.yaml` stages changes without immediately activating them, trigger its corresponding apply or reload operation after `Create`, `Update`, and `Delete`. The alias resource initiates a filter reload but does not confirm completion.
4. **Strict Secret Handling:**
   - Any field holding keys, passwords, PSKs, or tokens (`private_key`, `preshared_key`, `password`, `tls_key`) MUST have `Sensitive: true` in its `schema.StringAttribute`.
   - If a `GET` endpoint masks or omits a secret value, `Read()` MUST retain the existing `state` value rather than overwriting it with an empty string.
5. **Graceful 404 Handling in `Read()`:**
   - When the Nexus API returns `HTTP 404 Not Found` during `Read()`, call `resp.State.RemoveResource(ctx)` and return without adding an error diagnostic so Terraform detects out-of-band deletion cleanly.

## 🔄 Standard Development Workflow

Use pnpm for JavaScript-based repository tooling. The repository `.npmrc`
disables package installation scripts; do not override this setting. Run
required project scripts explicitly with `pnpm run`.

```mermaid
sequenceDiagram
    participant Dev as Agent / Developer
    participant Gen as tfplugingen-openapi / framework
    participant Client as internal/client (Mutex + Auth)
    participant Test as go test (httptest / TF_ACC)

    Dev->>Gen: 1. Update generator_config.yml & regenerate schemas
    Dev->>Client: 2. Map JSON DTOs & apply endpoints in CRUD methods
    Dev->>Test: 3. Write httptest unit test + resource.Test acceptance test
    Test-->>Dev: 4. Verify pass with `go test -race ./...`
```

## ✅ Definition of Done for Any New Resource

Before marking a resource complete, verify:

- [ ] Endpoint paths and JSON field tags match `docs/nexus.openapi.yaml`.
- [ ] `Create`, `Read`, `Update`, `Delete`, and `ImportState` are all implemented.
- [ ] Sensitive fields are marked `Sensitive: true`.
- [ ] Offline unit test in `internal/client` or `internal/provider` passes under `go test -race ./...`.
- [ ] No hardcoded credentials, IPs, or private keys appear in code or examples.
