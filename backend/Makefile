.PHONY: server dev test build deploy

server:
	go run ./cmd/server

# Same as server, with verbose [debug] logs (C7D5A6L_DEBUG=1).
dev:
	C7D5A6L_DEBUG=1 go run ./cmd/server

test:
	go test ./...

build:
	go build -o bin/server ./cmd/server

# Cross-compile, rsync to foundry, PM2 restart. Working tree must be clean.
deploy:
	./scripts/deploy.sh
