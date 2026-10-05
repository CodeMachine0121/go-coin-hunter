.PHONY: start build test mock

start:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

# -race is not optional: background jobs run goroutines that outlive a request.
test:
	go test ./... -race

# Each interface file in internal/domain/interface/ carries its own directive:
#   //go:generate go tool mockgen -source=i_xxx.go -destination=mocks/mock_i_xxx.go -package=mocks
mock:
	go generate ./...
