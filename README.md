# fudou

a high-performance, fault-tolerant distributed backup system, content-addressable storage engine, and real-time cluster telemetry dashboard built in raw go from first principles. client-side aes-256-gcm authenticated encryption, sha-256 integrity verification, stream chunking, n-way redundant replication, and automated self-healing failover across storage nodes.

---

## contents

- [features](#features)
- [architecture](#architecture)
- [technologies and stack](#technologies-and-stack)
- [live cluster dashboard and web ui](#live-cluster-dashboard-and-web-ui)
  - [dashboard capabilities](#dashboard-capabilities)
  - [how to run the web dashboard](#how-to-run-the-web-dashboard)
  - [using the web dashboard](#using-the-web-dashboard)
- [quickstart and cli reference](#quickstart-and-cli-reference)
  - [1. build binaries from source](#1-build-binaries-from-source)
  - [2. run the coordinator service](#2-run-the-coordinator-service)
  - [3. run storage node daemons](#3-run-storage-node-daemons)
  - [4. launch multi-node cluster with docker compose](#4-launch-multi-node-cluster-with-docker-compose)
  - [5. environment variables and configuration reference](#5-environment-variables-and-configuration-reference)
- [api reference and curl examples](#api-reference-and-curl-examples)
  - [generate authentication token](#generate-authentication-token)
  - [upload and backup a file](#upload-and-backup-a-file)
  - [list stored files](#list-stored-files)
  - [download and restore a file](#download-and-restore-a-file)
  - [delete a file](#delete-a-file)
  - [query cluster topology and telemetry](#query-cluster-topology-and-telemetry)
  - [query aggregate cluster metrics](#query-aggregate-cluster-metrics)
- [project structure](#project-structure)
- [how distributed backup works in fudou](#how-distributed-backup-works-in-fudou)
  - [stream chunking and segmentation](#stream-chunking-and-segmentation)
  - [aes-256-gcm authenticated encryption](#aes-256-gcm-authenticated-encryption)
  - [least-loaded n-way replication](#least-loaded-n-way-replication)
  - [automated self-healing engine](#automated-self-healing-engine)
- [testing](#testing)
- [license](#license)

---

## features

- zero external dependencies in core backend: pure go standard library implementation utilizing standard net/http, crypto/aes, crypto/cipher, crypto/sha256, and sync primitives.
- stream chunking: fixed-size stream segmentation (default 1MB chunks) allowing arbitrary file sizes to be streamed, hashed, and distributed without high memory overhead.
- zero-knowledge client encryption: each upload generates a cryptographically random 256-bit encryption key and unique 96-bit nonces per chunk using aes-256-gcm authenticated encryption.
- tamper-evident integrity verification: sha-256 checksums are calculated on raw byte streams before encryption and verified post-reassembly upon restore.
- n-way redundant replication: distributor allocates each chunk across multiple storage nodes (configurable replication factor, default 3x) targeting least-loaded nodes.
- automated self-healing failover: background monitor polls node health every 10 seconds, identifies under-replicated chunks when nodes go offline, and replicates missing chunks from healthy nodes to active nodes.
- independent storage node daemons: lightweight node daemons maintain isolated disk stores, enforce capacity quotas, and beacon periodic heartbeats to the coordinator.
- token authentication: hmac-sha256 signed bearer token service for secure access control.
- full-stack modern web dashboard: next.js 15 app with real-time cluster health topology, capacity gauges, drag-and-drop file uploader, decryption key modal, and one-click restore.
- multi-node docker compose orchestration: single-command deployment spinning up the coordinator alongside 3 isolated storage nodes.

---

## architecture

```text
[ Client / Web Dashboard / CLI ]
               |
               v (HTTP / REST API)
     +-------------------+
     |    COORDINATOR    | <----+ Background Self-Healing Worker
     |      (:8080)      | <----+ In-Memory / File Metadata Store
     +-------------------+
        |       |       |
   Chunk 0  Chunk 1  Chunk 2  (N-way Replication)
        |       |       |
        v       v       v
     +-----+ +-----+ +-----+
     |Node1| |Node2| |Node3|
     |:9001| |:9002| |:9003|
     +-----+ +-----+ +-----+
```

the coordinator acts as the central ingress gateway, metadata manager, and replication orchestrator:

1. backup pipeline: receives incoming multipart file streams, calculates overall sha-256 checksums, segments the payload into fixed chunks, encrypts each chunk with aes-256-gcm, selects least-loaded active nodes, and replicates chunks in parallel.
2. restore pipeline: looks up chunk metadata, queries healthy storage nodes in parallel, decrypts chunk payloads, reassembles chunks in deterministic order, verifies the sha-256 checksum, and streams the restored file to the client.
3. self-healing monitor: runs continuously in the background, checks node heartbeat freshness, detects dropped nodes, calculates replica deficits, and rebalances chunks to surviving nodes.
4. storage nodes: run independently on distinct ports or hosts, persist encrypted chunks to dedicated disk paths, and beacon status payloads every 5 seconds.

---

## technologies and stack

- language: go 1.24+ (standard library: net/http, crypto/aes, crypto/cipher, crypto/rand, crypto/sha256, crypto/hmac, sync, os, io)
- frontend: next.js 15, react 19, typescript, lucide-react, vanilla css tokens
- storage engine: content-addressable local disk storage with atomic write operations
- containers: docker, docker compose
- build tool: gnu make

---

## live cluster dashboard and web ui

fudou includes a responsive management dashboard built with next.js 15 and react 19 for visualizing cluster topology and executing zero-knowledge backup and restore operations.

### dashboard capabilities

- real-time cluster metrics: displays total files backed up, total bytes stored, aggregate cluster disk capacity, active node count, and default replication factor.
- cluster topology view: visualizes every registered storage node, including node identifier, network address, status badge (online/offline), used storage bytes, and total capacity progress bars.
- drag-and-drop backup interface: upload files of any type with automatic chunking and aes-256-gcm key generation.
- client-side key modal: exposes the generated 64-character hex encryption key upon backup completion, emphasizing zero-knowledge architecture.
- file catalog: tabular view displaying file name, mime type, size, sha-256 checksum preview, total chunk count, and upload timestamp.
- secure restore flow: download prompt requiring the user to provide their 64-character hex decryption key to authorize streaming decryption and checksum validation.
- distributed deletion: deletes file metadata from the coordinator and cascades chunk deletion requests to all storing nodes.

### how to run the web dashboard

```bash
cd web
npm install
npm run dev
```

the web interface is accessible at `http://localhost:3000`.

to point the frontend to a custom coordinator host or port, configure the environment variable:

```bash
NEXT_PUBLIC_COORDINATOR_URL=http://localhost:8080 npm run dev
```

### using the web dashboard

1. navigate to `http://localhost:3000/dashboard` to access file backup and catalog management.
2. drag a file into the upload zone or click to select a file from your system.
3. upon upload, copy and safely store the generated 64-character hex key shown in the success dialog.
4. to restore a file, click the download icon next to the record and paste your hex key.
5. navigate to `http://localhost:3000/admin` to inspect real-time storage node allocations, node health, and cluster storage utilization.

---

## quickstart and cli reference

### 1. build binaries from source

compile both the coordinator and storage node binaries into the local `bin/` directory:

```bash
make build
```

or build directly using the go compiler:

```bash
go build -o bin/coordinator ./cmd/coordinator
go build -o bin/node ./cmd/node
```

### 2. run the coordinator service

start the coordinator daemon listening on port 8080:

```bash
make run-coordinator
```

or start manually with custom parameters:

```bash
PORT=8080 REPLICATION_FACTOR=3 METADATA_PATH=./data/coordinator/meta.json AUTH_SECRET=dev-secret go run ./cmd/coordinator
```

### 3. run storage node daemons

in separate terminal sessions, start three independent storage node daemons:

```bash
make run-node1
```

```bash
make run-node2
```

```bash
make run-node3
```

or start individual nodes manually with environment variables:

```bash
NODE_ID=node-1 PORT=9001 STORAGE_DIR=./data/node1 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node
NODE_ID=node-2 PORT=9002 STORAGE_DIR=./data/node2 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node
NODE_ID=node-3 PORT=9003 STORAGE_DIR=./data/node3 COORDINATOR_URL=http://localhost:8080 go run ./cmd/node
```

each node registers with the coordinator immediately and dispatches recurring heartbeats.

### 4. launch multi-node cluster with docker compose

to spin up the entire cluster (coordinator and 3 storage nodes) in isolated containers with a single command:

```bash
make docker-up
```

or using docker compose directly:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

to stop and remove all cluster containers:

```bash
make docker-down
```

### 5. environment variables and configuration reference

#### coordinator options

| variable | type | default | description |
| :--- | :--- | :--- | :--- |
| `PORT` | integer | `8080` | tcp port for the coordinator http server |
| `REPLICATION_FACTOR` | integer | `3` | number of healthy storage nodes to replicate each chunk to |
| `METADATA_PATH` | string | `./data/metadata.json` | file path for the atomic persistent json metadata store |
| `AUTH_SECRET` | string | `default-fudou-secret-key-change-me` | hmac secret used to sign and verify api tokens |

#### storage node options

| variable | type | default | description |
| :--- | :--- | :--- | :--- |
| `NODE_ID` | string | `node-<timestamp>` | unique identifier for the storage node daemon |
| `PORT` | integer | `9001` | tcp port for the storage node daemon http server |
| `STORAGE_DIR` | string | `./data/chunks` | local filesystem directory for storing encrypted chunks |
| `COORDINATOR_URL` | string | `http://localhost:8080` | base url of the active coordinator instance |
| `HEARTBEAT_SEC` | integer | `5` | interval in seconds between outbound heartbeat signals |

#### web dashboard options

| variable | type | default | description |
| :--- | :--- | :--- | :--- |
| `NEXT_PUBLIC_COORDINATOR_URL` | string | `http://localhost:8080` | target coordinator url for browser api requests |

---

## api reference and curl examples

### generate authentication token

```bash
curl -X POST http://localhost:8080/api/auth/token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-001", "role": "admin"}'
```

sample response:

```json
{
  "role": "admin",
  "token": "d98a21f7c3...",
  "user_id": "user-001"
}
```

### upload and backup a file

```bash
curl -X POST http://localhost:8080/api/files \
  -F "file=@document.pdf" \
  -F "user_id=user-001"
```

sample response:

```json
{
  "FileID": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "Filename": "document.pdf",
  "TotalSize": 2097152,
  "Checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "ChunkCount": 2,
  "KeyHex": "a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0"
}
```

### list stored files

```bash
curl -X GET "http://localhost:8080/api/files?user_id=user-001"
```

sample response:

```json
[
  {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "user_id": "user-001",
    "filename": "document.pdf",
    "mime_type": "application/pdf",
    "size": 2097152,
    "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "chunk_count": 2,
    "created_at": "2026-09-14T10:00:00Z",
    "updated_at": "2026-09-14T10:00:00Z"
  }
]
```

### download and restore a file

pass the file identifier and the encryption key in the query parameter to stream and decrypt the restored file:

```bash
curl -X GET "http://localhost:8080/api/files/3fa85f64-5717-4562-b3fc-2c963f66afa6/download?key=a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0" \
  --output restored_document.pdf
```

### delete a file

```bash
curl -X DELETE http://localhost:8080/api/files/3fa85f64-5717-4562-b3fc-2c963f66afa6
```

### query cluster topology and telemetry

```bash
curl -X GET http://localhost:8080/api/admin/nodes
```

sample response:

```json
[
  {
    "id": "node-1",
    "address": "http://localhost:9001",
    "status": "online",
    "capacity": 10737418240,
    "used_bytes": 2097152,
    "last_seen": "2026-09-14T10:05:00Z"
  }
]
```

### query aggregate cluster metrics

```bash
curl -X GET http://localhost:8080/api/admin/metrics
```

sample response:

```json
{
  "total_files": 12,
  "total_bytes": 45097152,
  "total_capacity": 32212254720,
  "total_used": 135291456,
  "active_nodes": 3,
  "replication_factor": 3
}
```

---

## project structure

```text
fudou/
├── cmd/
│   ├── coordinator/
│   │   └── main.go               coordinator server entry point
│   └── node/
│       └── main.go               storage node daemon entry point
├── deploy/
│   ├── coordinator.Dockerfile    multi-stage alpine container for coordinator
│   ├── node.Dockerfile           multi-stage alpine container for storage node
│   └── docker-compose.yml        multi-node cluster orchestration
├── internal/
│   ├── api/
│   │   ├── handlers.go           http api endpoints for backup, restore, nodes, auth
│   │   ├── handlers_test.go      comprehensive api integration test suite
│   │   ├── metrics.go            cluster telemetry aggregation structures
│   │   └── middleware.go         cors headers and request access logging
│   ├── auth/
│   │   ├── auth.go               hmac-sha256 token generation and validation
│   │   └── auth_test.go          token tamper resistance and expiry tests
│   ├── chunker/
│   │   ├── chunker.go            chunk and chunker interface definitions
│   │   ├── fixed_chunker.go      fixed-size stream chunk segmentation engine
│   │   ├── fixed_chunker_test.go boundary, empty, and large stream tests
│   │   ├── reassembler.go        ordered chunk reassembly and integrity pipeline
│   │   └── reassembler_test.go   out-of-order and partial chunk reassembly tests
│   ├── config/
│   │   └── config.go             default parameters and environment loaders
│   ├── coordinator/
│   │   ├── backup.go             distributed backup coordination pipeline
│   │   ├── backup_test.go        end-to-end backup pipeline verification
│   │   ├── delete.go             distributed chunk deletion engine
│   │   ├── distributor.go        least-loaded node placement algorithm
│   │   ├── distributor_test.go   distribution and allocation tests
│   │   ├── healing.go            background self-healing and replication engine
│   │   ├── healing_test.go       node dropout detection and rebalancing tests
│   │   ├── node_client.go        http client for node chunk operations
│   │   ├── restore.go            distributed restore pipeline and verification
│   │   ├── restore_test.go       end-to-end restore pipeline verification
│   │   ├── transfer.go           concurrent chunk transfer engine
│   │   └── transfer_test.go      parallel chunk transfer and error handling tests
│   ├── crypto/
│   │   ├── aes_gcm.go            aes-256-gcm authenticated encryption and decryption
│   │   ├── aes_gcm_test.go       ciphertext verification and tamper failure tests
│   │   ├── crypto.go             encryptor and hasher interface contracts
│   │   ├── hasher.go             sha-256 stream hasher implementation
│   │   ├── hasher_test.go        stream checksum validation tests
│   │   ├── keys.go               secure 256-bit key and 96-bit nonce generators
│   │   └── keys_test.go          entropy and uniqueness tests
│   ├── metadata/
│   │   ├── file_store.go         atomic json metadata persistence implementation
│   │   ├── file_store_test.go    concurrent read/write and crash safety tests
│   │   ├── memory_store.go       in-memory thread-safe metadata store
│   │   ├── memory_store_test.go  metadata crud operations test suite
│   │   └── store.go              store interface and domain record definitions
│   └── node/
│       ├── disk_store.go         local disk storage engine with atomic writes
│       ├── disk_store_test.go    disk store io and quota boundary tests
│       ├── handler.go            http handlers for chunk store, read, delete
│       ├── handler_test.go       node http handler verification tests
│       ├── heartbeat.go          outbound heartbeat beacon service
│       ├── heartbeat_test.go     heartbeat interval and payload tests
│       ├── server.go             storage node daemon lifecycle and router
│       ├── server_test.go        graceful startup and shutdown tests
│       └── store.go              node storage interface definition
├── web/
│   ├── app/
│   │   ├── admin/
│   │   │   └── page.tsx          cluster topology, node inspection, and telemetry
│   │   ├── dashboard/
│   │   │   └── page.tsx          file upload, key display modal, and file catalog
│   │   ├── globals.css           sleek dark mode styling and typography
│   │   ├── layout.tsx            root next.js layout wrapper
│   │   └── page.tsx              landing page with architecture overview
│   ├── components/
│   │   ├── ClusterTopology.tsx   node cards with live health indicators and quotas
│   │   ├── FileList.tsx          file table with delete and restore actions
│   │   ├── FileUpload.tsx        drag-and-drop file upload zone
│   │   ├── Navbar.tsx            top navigation bar with live status indicators
│   │   ├── RestoreModal.tsx      decryption key prompt modal for downloads
│   │   └── StorageStats.tsx      metrics cards for total storage and nodes
│   ├── context/
│   │   └── AuthContext.tsx       react auth state and session token provider
│   ├── lib/
│   │   └── api.ts                type-safe api client methods for web frontend
│   ├── package.json              next.js, react, and dependencies manifest
│   └── tsconfig.json             typescript compiler configuration
├── go.mod                        go module definition
└── Makefile                      automation recipes for build, test, and run
```

---

## how distributed backup works in fudou

### stream chunking and segmentation

files are processed as streams rather than buffered wholly in memory. the `FixedChunker` reads standard `io.Reader` streams in chunks of 1MB (1,048,576 bytes). each chunk receives a sequential index, an independent byte count, and a deterministic content-addressable chunk identifier derived from its sequential position and content.

### aes-256-gcm authenticated encryption

before leaving the coordinator, every chunk is encrypted using standard galois/counter mode (gcm) with 256-bit keys:

1. a unique 256-bit encryption key is generated for the upload session using cryptographically secure random bytes from `crypto/rand`.
2. for each chunk, a unique 96-bit (12-byte) initialization vector (nonce) is generated.
3. the chunk payload is encrypted and sealed with an authenticated 128-bit tag. the nonce is prepended directly to the ciphertext chunk payload, ensuring each chunk is self-describing for decryption when the master key is provided.
4. raw content is never stored unencrypted on any storage node.

### least-loaded n-way replication

the coordinator tracks the storage usage of all active nodes through their periodic heartbeats:

1. for every chunk, the `Distributor` sorts all healthy, online storage nodes by `used_bytes` in ascending order.
2. the top `N` nodes (where `N` is the configured `REPLICATION_FACTOR`, default 3) are assigned as replica targets for that chunk.
3. the `ChunkTransferEngine` dispatches concurrent http put requests to all selected nodes via goroutines.
4. if any target node fails, the upload aborts or re-routes to maintain the target replication factor.

### automated self-healing engine

node failures are handled autonomously without human intervention:

1. every 5 seconds, each storage node posts a heartbeat to `/api/nodes/heartbeat` with its current byte count and status.
2. the coordinator records the timestamp of each heartbeat.
3. a background self-healing worker runs every 10 seconds. if a node has not reported within 15 seconds, it is marked as offline.
4. the self-healing worker scans all stored file chunks. any chunk residing on an offline node experiences a replica deficit.
5. the engine fetches the chunk from a surviving healthy replica node and transmits a copy to another healthy online node that does not yet hold the chunk.
6. the file metadata is updated atomically on disk to reflect the new replica locations.

---

## testing

the codebase includes unit and integration tests across all internal packages covering crypto verification, chunk boundary conditions, out-of-order reassembly, concurrent transfers, heartbeat beaconing, and self-healing.

to execute the complete test suite:

```bash
make test
```

or run tests directly with coverage details:

```bash
go test -v -race ./...
```

---

## license

mit license. inspect `LICENSE` or repository headers for copyright terms.
