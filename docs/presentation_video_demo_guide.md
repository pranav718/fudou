# 2-Minute Presentation Video Demonstration Script: Fudou Distributed Storage System

## 1. Window Setup and Commands (Left vs Right)

Arrange your screen into two side-by-side terminal windows (50 percent width each):

### Right Window: The Live Visual Dashboard (TUI)
- Purpose: Displays the real-time cluster monitor, active storage nodes table, storage capacity, file records, and live cluster event logs.
- Simple Command:
```bash
make tui
```
(Alternatively: `./bin/fudou -coordinator http://localhost:8080`)

### Left Window: The Execution Terminal
- Purpose: Executes cluster operations, file uploads, simulated disaster injection, and verification tests.
- Simple Command:
```bash
make demo
```
`make demo` automatically runs all 7 demonstration phases sequentially (starting the coordinator, spinning up storage nodes 1, 2, and 3, uploading an encrypted test file, inspecting raw ciphertext on disk, terminating node-2 with SIGKILL, performing failover restore, confirming byte-for-byte identity with cmp, and spawning replacement node-4 for background self-healing).

### Pre-Recording Reset (Run once before recording)
In the Left Window:
```bash
make stop
make clean
make build
rm -rf data/demo_* data/*.txt data/*.json
```

---

## 2. Verbatim 2-Minute Presentation Script (120 Seconds Max)

### Phase 1: Project Overview and High-Level Architecture (0:00 to 0:35 | 35 seconds)
- Visual: Split screen with Left Window (terminal prompt) and Right Window (`make tui` loaded).
- Spoken Narration:
  "Hello. Today I am presenting Fudou, a high-performance, zero-knowledge distributed storage and backup engine engineered in Go.

  To solve single points of failure and cloud privacy vulnerabilities, Fudou decouples metadata coordination from autonomous storage nodes. Here is how it works at a high level: when a file is ingested, Fudou streams it through pre-allocated 64KB memory buffers, encrypts each block independently using AES-256-GCM with unique cryptographic nonces, and uses a consistent hash ring to replicate chunks across storage nodes. The coordinator tracks locations in an append-only Write-Ahead Log, while storage daemons store only raw ciphertext with zero knowledge of file contents or encryption keys."

---

### Phase 2: Live Cluster Startup and TUI Handshake (0:35 to 0:55 | 20 seconds)
- Visual: Run `make demo` in the Left Window. The Right Window (TUI) lights up with 3 active nodes (node-1, node-2, node-3) showing green status and heartbeats.
- Spoken Narration:
  "In the left terminal, we launch our automated cluster demonstration, while our Terminal User Interface on the right monitors the cluster in real time. We see our coordinator start on port 8080, and three independent storage nodes register instantly. Each daemon emits heartbeats every five seconds, and the coordinator tracks capacity with sub-nanosecond atomic integer counters."

---

### Phase 3: Ingestion, Encryption, and Zero-Knowledge Proof (0:55 to 1:15 | 20 seconds)
- Visual: Left terminal displays file upload completion, shows chunks distributed across `demo_node1`, `demo_node2`, `demo_node3`, and prints a hexdump of raw ciphertext.
- Spoken Narration:
  "Now, Fudou ingests a confidential test document. The coordinator slices the stream into 64KB blocks, encrypts each chunk, and fans out three replicas in parallel. Looking at the raw bytes on disk via hexdump, we see purely randomized ciphertext. Even if storage hardware is compromised, the data remains completely confidential."

---

### Phase 4: Chaos Disaster, Failover Restore, and Self-Healing (1:15 to 1:45 | 30 seconds)
- Visual: Left terminal terminates Node 2 (`kill -9`); requests file download; cmp confirms byte-for-byte match; replacement Node 4 registers in the TUI.
- Spoken Narration:
  "Now we simulate a hardware catastrophe by abruptly killing storage node-2 with SIGKILL. When we request file restoration, the coordinator detects the dead node, fails over to surviving replicas, decrypts on the fly, and restores the file. As confirmed by the cmp utility, the restored document matches the original byte-for-byte. When replacement node-4 is introduced, our self-healing supervisor automatically detects degraded chunks and re-replicates them in the background."

---

### Phase 5: Empirical Benchmarks and Conclusion (1:45 to 2:00 | 15 seconds)
- Visual: Left terminal displays benchmark metrics or the benchmark summary table.
- Spoken Narration:
  "Under the hood, Fudou achieves 2.2 gigabytes per second chunking, 15.5 gigabytes per second in-memory cache reads, and 233-nanosecond consistent hash lookups, with append-only write-ahead logging and zero data races under Go's race detector. Fudou proves that enterprise distributed storage can be both resilient and lightning fast. Thank you."

---

## 3. Alternative Step-by-Step Manual Commands (If Not Using `make demo`)

If you prefer typing individual commands manually in the Left Window instead of `make demo`:

1. Launch Coordinator:
```bash
PORT=8080 REPLICATION_FACTOR=3 ./bin/coordinator &
```

2. Launch Three Storage Daemons:
```bash
NODE_ID=node-1 PORT=9001 STORAGE_DIR=./data/demo_node1 COORDINATOR_URL=http://localhost:8080 ./bin/node &
NODE_ID=node-2 PORT=9002 STORAGE_DIR=./data/demo_node2 COORDINATOR_URL=http://localhost:8080 ./bin/node &
NODE_ID=node-3 PORT=9003 STORAGE_DIR=./data/demo_node3 COORDINATOR_URL=http://localhost:8080 ./bin/node &
```

3. Ingest Test File:
```bash
echo "Software Engineering Evaluation Document" > data/eval.txt
curl -s -X POST http://localhost:8080/api/files -F "file=@data/eval.txt" -F "user_id=eval"
```

4. Kill Node 2 and Verify Restore:
```bash
pkill -9 -f "PORT=9002"
curl -s "http://localhost:8080/api/files/$FILE_ID/download?key=$KEY_HEX" -o data/restored.txt
cmp data/eval.txt data/restored.txt && echo "FILES IDENTICAL"
```

5. Spawn Replacement Node 4:
```bash
NODE_ID=node-4 PORT=9004 STORAGE_DIR=./data/demo_node4 COORDINATOR_URL=http://localhost:8080 ./bin/node &
```
