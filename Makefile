build:
	go build -o bin/arp ./cmd/

run: build
	./bin/arp
