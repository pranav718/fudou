.PHONY: build test run-coordinator run-node docker-up docker-down clean

build:
	go build -o bin/coordinator ./cmd/coordinator
	go build -o bin/node ./cmd/node

test:
	go test -v ./...

run-coordinator:
	go run ./cmd/coordinator

run-node:
	go run ./cmd/node

docker-up:
	docker compose -f deploy/docker-compose.yml up --build

docker-down:
	docker compose -f deploy/docker-compose.yml down

clean:
	rm -rf bin/
