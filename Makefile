build:
	go build -ldflags="-s -w" -o bin/arp ./cmd/

run: build
	./bin/arp

.PHONY: testupstream
testupstream:
	go run ./testupstream/main.go &

run-unit-test: testupstream
	go test ./... -v

test-e2e: testupstream
	ginkgo run --timeout=30m -v --race ./test/e2e
