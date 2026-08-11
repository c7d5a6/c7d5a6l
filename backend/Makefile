.PHONY: server dev test

server:
	go run ./cmd/server

# Same as server, with verbose [debug] logs (C7D5A6L_DEBUG=1).
dev:
	C7D5A6L_DEBUG=1 go run ./cmd/server

test:
	go test ./...
