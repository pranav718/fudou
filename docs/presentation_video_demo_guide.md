# 2-Minute Presentation Video Demonstration Script: Fudou Distributed Storage System

## 1. Quick Setup and Screen Layout (0:00 to 2:00)

### Screen Setup
- Left Window (50 percent width): Terminal for running commands.
- Right Window (50 percent width): Interactive Terminal User Interface (TUI) running `bin/fudou`.

### Reset Environment Before Hitting Record
```bash
make stop
make clean
make build
rm -rf data/demo_* data/*.txt data/*.json
```

### One-Command Demonstration Strategy
Launch the TUI in the right window:
```bash
./bin/fudou -coordinator http://localhost:8080
```
Then in the left window, run the automated evaluation runner while speaking the script:
```bash
make demo
```
`make demo` executes all 7 phases sequentially (cluster startup, upload, encryption inspection, killing node-2, failover restore, byte-for-byte verification, and self-healing with node-4). You can speak the 2-minute narration in sync without typing manual commands under time pressure.

---

## 2. Verbatim 2-Minute Presentation Script (120 Seconds Max)

### Phase 1: Problem Statement and Architecture (0:00 to 0:20 | 20 seconds)
- Visual: Split screen with terminal workspace on left and TUI on right.
- Spoken Narration:
  "Hello. Today I am presenting Fudou, a high-performance, zero-knowledge distributed object storage and backup engine engineered in Go. Fudou eliminates single points of failure by decoupling metadata coordination from autonomous storage nodes, applying AES-256-GCM authenticated encryption, and enforcing distributed replication across an autonomous cluster."

---

### Phase 2: Live Cluster Startup and TUI Monitoring (0:20 to 0:45 | 25 seconds)
- Visual: Coordinator starts on port 8080; storage nodes 1, 2, and 3 register in the TUI table.
- Spoken Narration:
  "On screen, we launch our coordinator on port 8080 and spin up three independent storage daemon instances. Notice our real-time Terminal User Interface on the right: each node registers atomically and beacons heartbeats every five seconds, with the coordinator monitoring storage health and node capacity using atomic sub-nanosecond integer counters."

---

### Phase 3: Zero-Alloc Chunking and Zero-Knowledge Storage (0:45 to 1:10 | 25 seconds)
- Visual: File upload completes; terminal displays chunk directories and hexdump of raw ciphertext.
- Spoken Narration:
  "Now, we ingest a confidential file. The pipeline streams data through an io.TeeReader into pooled 64KB buffers, encrypts each chunk using hardware-accelerated AES-GCM with unique nonces, and distributes three replicas across nodes. Inspecting the physical disk directories reveals purely randomized ciphertext: storage daemons hold zero knowledge of file names, plaintext, or user keys."

---

### Phase 4: Chaos Disaster, Failover Restore, and Self-Healing (1:10 to 1:40 | 30 seconds)
- Visual: Node 2 is killed (`kill -9`); file downloads seamlessly; cmp utility confirms byte-for-byte match; replacement Node 4 registers.
- Spoken Narration:
  "Next, we simulate production disaster by forcibly terminating Node 2 with SIGKILL. When we request file restoration, the coordinator catches the transport failure, fails over to surviving replicas, decrypts on the fly, and reconstructs the original file byte-for-byte. When replacement Node 4 joins, our self-healing supervisor automatically detects degraded chunks and re-replicates them in the background."

---

### Phase 5: Empirical Benchmarks and Engineering Rigor (1:40 to 2:00 | 20 seconds)
- Visual: Benchmark results table or terminal running `make bench-cache`.
- Spoken Narration:
  "Under the hood, Fudou achieves 2.2 gigabytes per second chunking throughput, 15.5 gigabytes per second in-memory cache reads, and 233-nanosecond consistent hash lookups, with append-only write-ahead logging and zero data races. Fudou proves that enterprise-grade distributed resilience can be both lightweight and exceptionally fast. Thank you."

---

## 3. Alternative Manual Command Sequence (If Not Using `make demo`)

If you prefer executing individual commands manually instead of `make demo`:

1. Start Coordinator:
```bash
PORT=8080 REPLICATION_FACTOR=3 ./bin/coordinator &
```

2. Start Storage Nodes:
```bash
NODE_ID=node-1 PORT=9001 STORAGE_DIR=./data/demo_node1 COORDINATOR_URL=http://localhost:8080 ./bin/node &
NODE_ID=node-2 PORT=9002 STORAGE_DIR=./data/demo_node2 COORDINATOR_URL=http://localhost:8080 ./bin/node &
NODE_ID=node-3 PORT=9003 STORAGE_DIR=./data/demo_node3 COORDINATOR_URL=http://localhost:8080 ./bin/node &
```

3. Upload File:
```bash
echo "Evaluation Secret Document" > data/test.txt
curl -s -X POST http://localhost:8080/api/files -F "file=@data/test.txt" -F "user_id=eval"
```

4. Kill Node 2 and Restore:
```bash
pkill -9 -f "PORT=9002"
curl -s "http://localhost:8080/api/files/$FILE_ID/download?key=$KEY_HEX" -o data/restored.txt
cmp data/test.txt data/restored.txt && echo "FILES IDENTICAL"
```

5. Spawn Node 4:
```bash
NODE_ID=node-4 PORT=9004 STORAGE_DIR=./data/demo_node4 COORDINATOR_URL=http://localhost:8080 ./bin/node &
```
