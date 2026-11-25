.PHONY: generate lint test install-hooks

generate:
	go run github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@v0.3.0 generate --config generator_config.yaml --output provider_code_spec.json docs/nexus.openapi.yaml
	go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@v0.4.1 generate resources --input provider_code_spec.json --output internal/provider
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.2 -config oapi-codegen.yaml -o internal/api/client_gen.go docs/nexus.openapi.yaml

lint:
	pnpm run lint

test:
	pnpm run test

install-hooks:
	pnpm run setup-hooks
