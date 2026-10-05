.PHONY: start build test test-storage mock

start:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

# -race is not optional: background jobs run goroutines that outlive a request.
test:
	go test ./... -race

# Same suite, with the storage tests run against TEST_POSTGRES_DSN (a database whose name ends in _test);
# -p 1 because they share that database and empty it per test.
test-storage:
	go test ./... -race -count=1 -p 1

# Each interface file in internal/domain/interface/ carries its own directive:
#   //go:generate go tool mockgen -source=i_xxx.go -destination=mocks/mock_i_xxx.go -package=mocks
mock:
	go generate ./...
