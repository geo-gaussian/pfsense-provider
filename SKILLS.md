# 🧠 SKILLS.md — Technical Patterns & Playbooks

## 🔌 Skill 1: Adding a New Resource via `tfplugingen-openapi`

1. Locate the target `POST`, `GET`, `PUT`, and `DELETE` routes in `docs/nexus.openapi.yaml`.
2. Add the resource mapping under `resources:` in `generator_config.yml`:
   ```yaml
   resources:
     <resource_name>:
       create: { path: /api/v1/..., method: POST }
       read:   { path: /api/v1/.../{id}, method: GET }
       update: { path: /api/v1/.../{id}, method: PUT }
       delete: { path: /api/v1/.../{id}, method: DELETE }
   ```
3. Run:
   ```sh
   tfplugingen-openapi generate --config generator_config.yml --output provider_code_spec.json docs/nexus.openapi.yaml
   tfplugingen-framework generate all --input provider_code_spec.json --output internal/provider
   ```
4. Register `New<ResourceName>Resource` in `internal/provider/provider.go` under `Resources()`.

## 📥 Skill 2: Implementing `ImportState` for Brownfield HCL Generation

Every resource must implement `resource.ResourceWithImportState` so `terraform plan -generate-config-out=...` works:

```go
func (r *exampleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

_(Replace `"id"` with `"name"` or `"vlanif"` if the Nexus API uses a natural key in its `GET` path.)_

## 🔒 Skill 3: Preventing Secret Wipeouts on `Read()`

When Netgate's API omits write-only secrets (like WireGuard `privatekey`) from `GET` responses, preserve prior state:

```go
if apiResp.PrivateKey != "" {
	state.PrivateKey = types.StringValue(apiResp.PrivateKey)
}
// Otherwise keep state.PrivateKey untouched from req.State.Get(ctx, &state)
```

## 🧪 Skill 4: Offline HTTP Unit Testing with `httptest`

Always test JSON serialization, Bearer headers, and HTTP error handling without needing a live router:

```go
server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/login" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"mock-jwt-token"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
}))
defer server.Close()
```

## 🖥️ Skill 5: Booting pfSense 26.10 in GitHub Actions (QEMU/KVM)

On `ubuntu-latest` runners, forward host port `8443` to the FreeBSD guest so `TF_ACC=1` tests run against `https://127.0.0.1:8443`:

```sh
qemu-system-x86_64 \
  -enable-kvm -m 2048 -smp 2 \
  -drive file=pfsense-26.10.qcow2,format=qcow2 \
  -netdev user,id=net0,hostfwd=tcp:127.0.0.1:8443-:8443 \
  -device virtio-net-pci,netdev=net0 \
  -display none -daemonize
```
