.PHONY: test test-race test-coverage test-integration

test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-integration:
	go test ./...

