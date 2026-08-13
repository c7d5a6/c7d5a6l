.PHONY: server dev test build deploy ensure-env

# Source backend/.env in the same shell as the command (missing → copy from example).
ENV_LOAD = set -a && . ./.env && set +a

ensure-env:
	@test -f .env || cp .env.example .env

server: ensure-env
	$(ENV_LOAD) && go run ./cmd/server

# Same as server, with verbose [debug] logs (overrides C7D5A6L_DEBUG in .env).
dev: ensure-env
	$(ENV_LOAD) && C7D5A6L_DEBUG=1 go run ./cmd/server

test:
	go test ./...

build:
	go build -o bin/server ./cmd/server

# Cross-compile, rsync to foundry, PM2 restart. Working tree must be clean.
deploy:
	./scripts/deploy.sh
