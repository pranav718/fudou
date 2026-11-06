# Video Demonstration and Presentation Guide: Fudou Distributed Storage System

## 1. Overview and Recording Specifications

This document provides a turnkey script, scene-by-scene timing, screen layout, and verbatim spoken narration for recording the project demonstration video required for the Software Engineering Lab project evaluation.

### Production Specifications

- Target Video Duration: 6 to 7 minutes
- Screen Resolution: 1920x1080 (1080p, 60fps)
- Audio: Clear voiceover without background noise
- Recommended Screen Layout: Split screen or dual terminal windows:
  - Left Window (50 percent width): Execution terminal for running cluster commands, inspecting logs, checking disk files, and executing the automated demo script.
  - Right Window (50 percent width): Interactive Terminal User Interface (TUI) running `bin/fudou` to visually track node status, live metrics, file records, and storage distribution in real time.

---

## 2. Pre-Recording Preparation Checklist

Execute these setup commands in your terminal before starting your screen recording software to ensure a pristine demonstration environment:

```bash
make stop
make clean
make build
rm -rf data/demo_* data/*.txt data/*.json
```

Verify that all necessary binaries are present:
- `bin/coordinator`
- `bin/node`
- `bin/fudou`

---

## 3. Scene-by-Scene Demonstration Script

### Scene 1: Introduction and Architecture Overview
- Timing: 0:00 to 1:00 (Duration: 60 seconds)
- Visual on Screen: Title slide or terminal workspace showing the project tree and architecture diagram.
- Spoken Narration:
  "Hello everyone and welcome to our demonstration of Fudou, a high-performance, fault-tolerant distributed object storage and backup engine written from the ground up in Go.
  
  Modern centralized backup platforms present two critical risks: single points of failure and security exposure from storing raw unencrypted data in centralized cloud repositories. Fudou solves this by decoupling metadata coordination from storage nodes, dividing files into deterministic fixed-size chunks, applying hardware-accelerated AES-256-GCM authenticated encryption, and distributing chunks with configurable replication across an autonomous cluster.
  
  In this demonstration, we will showcase the live distributed cluster, verify zero-knowledge encrypted chunk distribution, execute a real-time failover under simulated hardware disaster, observe autonomous self-healing, and review our empirical performance benchmarks."

---

### Scene 2: Cluster Initialization and Real-Time Dashboard
- Timing: 1:00 to 2:15 (Duration: 75 seconds)
- Action:
  1. Open Right Terminal and launch the interactive TUI:
     ```bash
     ./bin/fudou -coordinator http://localhost:8080
     ```
  2. Open Left Terminal and execute the first phase of the cluster:
     ```bash
     PORT=8080 REPLICATION_FACTOR=3 go run ./cmd/coordinator &
     ```
  3. Spin up three independent storage nodes on separate ports and storage directories:
     ```bash
     NODE_ID=node-1 PORT=9001 STORAGE_DIR=./data/demo_node1 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node &
     NODE_ID=node-2 PORT=9002 STORAGE_DIR=./data/demo_node2 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node &
     NODE_ID=node-3 PORT=9003 STORAGE_DIR=./data/demo_node3 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node &
     ```
- Spoken Narration:
  "Let us initialize the cluster. On the right, we have launched the Fudou Terminal User Interface built with the Elm architecture and Bubble Tea. On the left, we initialize our coordinator service on port 8080 with a replication factor of 3.
  
  Next, we spin up three independent storage daemon instances: node-1 on port 9001, node-2 on port 9002, and node-3 on port 9003, each managing an isolated disk storage directory.
  
  Notice the right pane: each node immediately registers itself with the coordinator via an atomic handshake and transmits periodic heartbeats. The coordinator monitors node health, tracking live capacity and active status with sub-millisecond atomic integer counters."

---

### Scene 3: Secure Chunking, Encryption, and Zero-Knowledge Storage
- Timing: 2:15 to 3:45 (Duration: 90 seconds)
- Action:
  1. Create a sample confidential file:
     ```bash
     echo "Manipal University Software Engineering Evaluation Payload 2026" > data/confidential.txt
     ```
  2. Upload the file to the cluster:
     ```bash
     curl -s -X POST http://localhost:8080/api/files -F "file=@data/confidential.txt" -F "user_id=student-eval" | jq .
     ```
  3. Inspect the storage directories of node-1, node-2, and node-3:
     ```bash
     ls -lh data/demo_node1/ data/demo_node2/ data/demo_node3/
     ```
  4. Hexdump one of the stored chunk files:
     ```bash
     hexdump -C data/demo_node1/chk-* | head -n 8
     ```
- Spoken Narration:
  "Now let us upload a confidential test document. The coordinator accepts the stream through an io.TeeReader, simultaneously computing a master SHA-256 checksum while feeding pre-allocated 64-kilobyte buffers recycled through a sync.Pool.
  
  The coordinator generates a high-entropy 256-bit AES key. Each chunk receives a cryptographic nonce and is sealed with AES-GCM authenticated encryption. The coordinator then fans out the encrypted ciphertext in parallel to our three storage nodes.
  
  As we inspect the physical filesystem across node-1, node-2, and node-3, we observe identical chunk distribution fulfilling our 3x replication policy. Furthermore, when we inspect the raw bytes on disk using hexdump, we see purely randomized ciphertext. Storage nodes hold zero knowledge of file contents, filenames, or user metadata, ensuring complete data confidentiality even if physical hardware is compromised."

---

### Scene 4: Simulated Hardware Crash and Byzantine Failover Restore
- Timing: 3:45 to 5:00 (Duration: 75 seconds)
- Action:
  1. Identify the PID of Storage Node 2 and forcibly kill it:
     ```bash
     pkill -9 -f "PORT=9002"
     ```
  2. Check the coordinator admin endpoint or observe the TUI dashboard:
     ```bash
     curl -s http://localhost:8080/api/admin/nodes | jq .
     ```
  3. Request file restoration using the master key:
     ```bash
     curl -s "http://localhost:8080/api/files/$FILE_ID/download?key=$KEY_HEX" -o data/restored.txt
     cat data/restored.txt
     cmp data/confidential.txt data/restored.txt && echo "FILES IDENTICAL"
     ```
- Spoken Narration:
  "Now let us simulate a severe hardware failure in production. We issue a SIGKILL to terminate storage node-2 abruptly.
  
  Looking at our dashboard, node-2 stops emitting heartbeats. Now, what happens when a client requests a file download?
  
  The restore engine attempts to stream chunks concurrently. When contact with node-2 fails, the coordinator's parallel fetcher dynamically catches the transport fault, consults metadata for alternate replica holders, and seamlessly pulls the verified chunks from node-1 and node-3.
  
  The stream is decrypted on the fly with AES-GCM and verified against the original SHA-256 checksum. As demonstrated by the cmp utility, the restored payload matches our original input byte for byte with zero data corruption and zero downtime for the user."

---

### Scene 5: Autonomous Self-Healing and Cluster Expansion
- Timing: 5:00 to 5:45 (Duration: 45 seconds)
- Action:
  1. Spawn a replacement storage node (Node 4):
     ```bash
     NODE_ID=node-4 PORT=9004 STORAGE_DIR=./data/demo_node4 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node &
     ```
  2. Show Node 4 registering in the TUI or via curl:
     ```bash
     curl -s http://localhost:8080/api/admin/nodes | jq .
     ```
- Spoken Narration:
  "To restore cluster fault tolerance, we introduce a new replacement storage node, node-4 on port 9004.
  
  Upon registration, the coordinator's self-healing supervisor scans the metadata ledger for under-replicated chunks whose replica count dropped below the target factor of 3. The coordinator orchestrates background node-to-node replication, copying degraded chunks to node-4 without client involvement. Fudou thus achieves self-healing high availability."

---

### Scene 6: Benchmarks, Testing Harness, and Conclusion
- Timing: 5:45 to 6:45 (Duration: 60 seconds)
- Action:
  1. Run the benchmark suite:
     ```bash
     make bench-crypto
     make bench-chunker
     ```
  2. Highlight the race condition tests:
     ```bash
     go test -count=1 -race ./internal/...
     ```
- Spoken Narration:
  "Finally, let us review our empirical performance and testing harness. Our chunking engine sustains over 2.2 gigabytes per second throughput, while AES-GCM hardware-accelerated encryption exceeds 5.0 gigabytes per second on Apple Silicon. Furthermore, our concurrent disk store achieves 7.1 gigabytes per second read speed with O(1) atomic quota evaluations in 0.48 nanoseconds.
  
  Our test harness features native Go fuzzing, deterministic network chaos injection, and 100 percent pass rates under Go's race detector.
  
  Thank you for watching our demonstration of Fudou."

---

## 4. Troubleshooting and Contingency Guide

- If ports 8080 or 9001 through 9004 are occupied before recording:
  ```bash
  make stop
  lsof -ti:8080,9001,9002,9003,9004 | xargs kill -9 2>/dev/null || true
  ```
- If you prefer an entirely automated end-to-end run during the video:
  ```bash
  make demo
  ```
  The script handles all 7 phases, prints color-coded diagnostic banners, and leaves background daemons active for TUI inspection.

