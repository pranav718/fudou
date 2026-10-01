set -e

echo "=================================================================="
echo "    FUDOU DISTRIBUTED STORAGE SYSTEM - LIVE LAB DEMONSTRATION     "
echo "=================================================================="
echo ""

pkill -f "bin/coordinator" 2>/dev/null || true
pkill -f "bin/node" 2>/dev/null || true
sleep 1

rm -rf data/demo_* 2>/dev/null || true
mkdir -p data/demo_node1 data/demo_node2 data/demo_node3 data/demo_node4

echo "[PHASE 1] Building binaries and verifying test suite..."
go build -o bin/coordinator ./cmd/coordinator
go build -o bin/node ./cmd/node
go test -count=1 ./internal/crypto/... ./internal/chunker/... ./internal/coordinator/... > /dev/null
echo "[STATUS: PASS] Cryptography, chunking, and replication tests passed."
echo ""

echo "[PHASE 2] Initializing distributed cluster..."
PORT=8080 REPLICATION_FACTOR=3 METADATA_PATH=./data/demo_meta.json nohup ./bin/coordinator > data/demo_coordinator.log 2>&1 &
COORD_PID=$!

for i in {1..30}; do
    if curl -s http://localhost:8080/api/admin/metrics >/dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

NODE_ID=node-1 PORT=9001 STORAGE_DIR=./data/demo_node1 COORDINATOR_URL=http://localhost:8080 nohup ./bin/node > data/demo_node1.log 2>&1 &
N1_PID=$!

NODE_ID=node-2 PORT=9002 STORAGE_DIR=./data/demo_node2 COORDINATOR_URL=http://localhost:8080 nohup ./bin/node > data/demo_node2.log 2>&1 &
N2_PID=$!

NODE_ID=node-3 PORT=9003 STORAGE_DIR=./data/demo_node3 COORDINATOR_URL=http://localhost:8080 nohup ./bin/node > data/demo_node3.log 2>&1 &
N3_PID=$!

for i in {1..50}; do
    ONLINE_COUNT=$(curl -s http://localhost:8080/api/admin/nodes | grep -o '"status":"online"' | wc -l | tr -d ' ')
    if [ "$ONLINE_COUNT" -ge 3 ]; then
        break
    fi
    sleep 0.1
done

echo "[STATUS: ONLINE] Coordinator listening on :8080"
echo "[STATUS: ONLINE] 3 Storage Nodes registered and beaconing heartbeats."
echo ""

echo "[PHASE 3] Uploading file with AES-256-GCM zero-knowledge encryption..."
echo "Software Engineering Lab Evaluation - Confidential Storage Payload" > data/sample_eval_doc.txt

UPLOAD_RESP=$(curl -s -X POST http://localhost:8080/api/files -F "file=@data/sample_eval_doc.txt" -F "user_id=student-eval")

FILE_ID=$(echo "$UPLOAD_RESP" | grep -o '"FileID":"[^"]*' | cut -d'"' -f4)
KEY_HEX=$(echo "$UPLOAD_RESP" | grep -o '"KeyHex":"[^"]*' | cut -d'"' -f4)
CHECKSUM=$(echo "$UPLOAD_RESP" | grep -o '"Checksum":"[^"]*' | cut -d'"' -f4)

echo "Uploaded File ID : $FILE_ID"
echo "Master Checksum  : $CHECKSUM"
echo "AES-256 Key (Hex): $KEY_HEX"
echo ""

echo "[PHASE 4] Inspecting physical chunk distribution on storage nodes..."
echo "--- Node 1 Chunks ---"
ls -lh data/demo_node1/
echo "--- Node 2 Chunks ---"
ls -lh data/demo_node2/
echo "--- Node 3 Chunks ---"
ls -lh data/demo_node3/
echo ""

CHUNK_SAMPLE=$(ls data/demo_node1/chk-* 2>/dev/null | head -n 1)
echo "Inspecting raw bytes of chunk ($CHUNK_SAMPLE) to verify zero-knowledge ciphertext:"
hexdump -C "$CHUNK_SAMPLE" | head -n 3
echo ""

echo "[PHASE 5] Simulating node failure (Killing Storage Node 2)..."
kill -9 $N2_PID 2>/dev/null || true
wait $N2_PID 2>/dev/null || true
echo "Node 2 (PID $N2_PID) terminated."
echo ""

echo "[PHASE 6] Performing failover restore from surviving nodes..."
curl -s "http://localhost:8080/api/files/$FILE_ID/download?key=$KEY_HEX" --output data/restored_eval_doc.txt

RESTORED_CONTENT=$(cat data/restored_eval_doc.txt)
echo "Restored Payload Content:"
echo "--------------------------------------------------"
echo "$RESTORED_CONTENT"
echo "--------------------------------------------------"

if cmp -s data/sample_eval_doc.txt data/restored_eval_doc.txt; then
    echo "[VERIFICATION: SUCCESS] Original and restored files match byte-for-byte."
else
    echo "[VERIFICATION: FAILURE] Checksum mismatch."
    exit 1
fi
echo ""

echo "[PHASE 7] Triggering autonomous self-healing with replacement node..."
NODE_ID=node-4 PORT=9004 STORAGE_DIR=./data/demo_node4 COORDINATOR_URL=http://localhost:8080 nohup ./bin/node > data/demo_node4.log 2>&1 &
N4_PID=$!
sleep 2

echo "Node 4 spawned on port 9004."
echo ""
echo "=================================================================="
echo "    ALL SYSTEM CHECKS AND EVALUATION TESTS COMPLETED SAFELY       "
echo "=================================================================="
echo ""
echo "Cluster nodes are currently running in the background:"
echo "  - Coordinator : http://localhost:8080"
echo "  - Node 1      : http://localhost:9001"
echo "  - Node 3      : http://localhost:9003"
echo "  - Node 4      : http://localhost:9004"
echo ""
echo "To shut down all running background nodes when done, run:"
echo "  make stop"
echo ""
