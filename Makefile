.PHONY: lint test install-hooks

lint:
	pnpm run lint

test:
	pnpm run test

install-hooks:
	pnpm run setup-hooks
