dev:
	@go run ./cmd/server/main.go -config ./config/config.yaml

build:
	@go build -o bin/server ./cmd/server/main.go