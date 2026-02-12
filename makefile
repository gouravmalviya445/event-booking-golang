dev:
	@go run ./cmd/server/main.go -config ./config/config.yaml

build:
	@go build -tags netgo -ldflags '-s -w' -o app ./cmd/server/main.go