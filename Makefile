.PHONY: build test run-coordinator run-node1 run-node2 run-node3 dev-web docker-up docker-down clean

build:
	go build -o bin/coordinator ./cmd/coordinator
	go build -o bin/node ./cmd/node

test:
	go test -v ./...

run-coordinator:
	PORT=8080 REPLICATION_FACTOR=3 go run ./cmd/coordinator

run-node1:
	NODE_ID=node-1 PORT=9001 STORAGE_DIR=./data/node1 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node

run-node2:
	NODE_ID=node-2 PORT=9002 STORAGE_DIR=./data/node2 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node

run-node3:
	NODE_ID=node-3 PORT=9003 STORAGE_DIR=./data/node3 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node

dev-web:
	cd web && npm run dev

docker-up:
	docker compose -f deploy/docker-compose.yml up --build

docker-down:
	docker compose -f deploy/docker-compose.yml down

clean:
	rm -rf bin/ data/
