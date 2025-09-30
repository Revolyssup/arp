build:
	go build -o bin/arp ./cmd/

run: build
	./bin/arp

.PHONY: testupstream
testupstream:
	go run ./testupstream/main.go

run-unit-test: testupstream
	go test ./... -v
