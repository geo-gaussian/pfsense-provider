# 🧠 Development Playbooks

These project playbooks supplement [AGENTS.md](AGENTS.md); they are not Spec
Kit agent skills. Spec Kit's installed skills live in `.github/skills/`.

## 🔌 Adding a New Resource via `tfplugingen-openapi`

1. Locate the target operations and methods in `docs/nexus.openapi.yaml`.
2. Add the mapping under `resources:` in `generator_config.yml`, and include
   the necessary operation IDs in `oapi-codegen.yml`. Use the existing alias
   mapping as an example; methods and paths vary by endpoint.
3. Run `make generate`, which uses the repository's pinned generator versions.
   Review generated changes and preserve custom resource lifecycle code.
4. Implement the lifecycle and import methods, register the resource in
   `internal/provider/provider.go`, and test its behaviour with mock servers.
   Trigger the appropriate reload or apply operation when required.

## 📥 Implementing `ImportState` for Brownfield HCL Generation

Every resource must implement `resource.ResourceWithImportState`. An import
block and `terraform plan -generate-config-out=...` can help generate
configuration for supported resources; review generated output for secrets:

```go
func (r *exampleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
  resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

Replace `"id"` with the actual state attribute used as the import identifier.
For example, the alias resource imports by `name`.

## 🔒 Preventing Secret Wipeouts on `Read()`

When Netgate's API omits write-only secrets (like WireGuard `privatekey`) from `GET` responses, preserve prior state:

```go
if apiResp.PrivateKey != "" {
  state.PrivateKey = types.StringValue(apiResp.PrivateKey)
}
// Otherwise keep state.PrivateKey untouched from req.State.Get(ctx, &state)
```

## 🧪 Offline HTTP Unit Testing with `httptest`

Test JSON serialization, authentication, and HTTP errors without needing a live
router. See `internal/client/client_test.go` and the existing resource and data
source tests for the current request paths and mock-server patterns.

## 🖥️ Planning live acceptance tests

Live acceptance tests and an appliance provisioning workflow are not yet
implemented. Use an isolated, disposable pfSense environment and test-only
credentials when adding them. Do not expose credentials or the appliance to
untrusted pull request code. Validate image availability, networking, and
runner support before choosing a virtualisation approach.
