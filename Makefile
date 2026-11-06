.PHONY: build test run-coordinator run-node1 run-node2 run-node3 tui docker-up docker-down clean demo status stop bench bench-crypto bench-chunker bench-node bench-pipeline bench-profile bench-wal bench-cache bench-ring

build:
	go build -o bin/coordinator ./cmd/coordinator
	go build -o bin/node ./cmd/node
	go build -o bin/fudou ./cmd/fudou

test:
	go test -v ./...

bench:
	go test -run=^$$ -bench=. -benchmem ./...

bench-crypto:
	go test -run=^$$ -bench=BenchmarkAESGCM -benchmem ./internal/crypto/...

bench-chunker:
	go test -run=^$$ -bench=BenchmarkFixedChunker -benchmem ./internal/chunker/...

bench-node:
	go test -run=^$$ -bench=BenchmarkDiskStore -benchmem ./internal/node/...

bench-pipeline:
	go test -run=^$$ -bench=BenchmarkPipeline -benchmem ./internal/coordinator/...

bench-profile:
	go test -run=^$$ -bench=BenchmarkPipeline_Backup/4MB -benchmem -cpuprofile=cpu.pprof -memprofile=mem.pprof ./internal/coordinator/...

bench-wal:
	go test -run=^$$ -bench=BenchmarkWAL -benchmem ./internal/metadata/...

bench-cache:
	go test -run=^$$ -bench=BenchmarkLRUCache -benchmem ./internal/node/...

bench-ring:
	go test -run=^$$ -bench=BenchmarkHashRing -benchmem ./internal/coordinator/...

demo:
	bash scripts/demo.sh

status:
	curl -s http://localhost:8080/api/admin/nodes

stop:
	pkill -f "bin/coordinator" 2>/dev/null || true
	pkill -f "bin/node" 2>/dev/null || true
	echo "All cluster services stopped."

run-coordinator:
	PORT=8080 REPLICATION_FACTOR=3 go run ./cmd/coordinator

run-node1:
	NODE_ID=node-1 PORT=9001 STORAGE_DIR=./data/node1 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node

run-node2:
	NODE_ID=node-2 PORT=9002 STORAGE_DIR=./data/node2 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node

run-node3:
	NODE_ID=node-3 PORT=9003 STORAGE_DIR=./data/node3 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node

tui:
	go run ./cmd/fudou

docker-up:
	docker compose -f deploy/docker-compose.yml up --build

docker-down:
	docker compose -f deploy/docker-compose.yml down

clean:
	rm -rf bin/ data/ cpu.pprof mem.pprof
