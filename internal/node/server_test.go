package node

import (
	"os"
	"testing"

	"github.com/pranav718/fudou/internal/config"
)

func TestNewServerAddressConfiguration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fudou-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewDiskStore(tempDir, 1024*1024)
	if err != nil {
		t.Fatalf("failed to create disk store: %v", err)
	}

	cfgDefault := config.DefaultNodeConfig()
	cfgDefault.Port = 9015
	srvDefault := NewServer(cfgDefault, store)
	if srvDefault.heartbeat.address != "http://localhost:9015" {
		t.Fatalf("expected fallback address http://localhost:9015, got %s", srvDefault.heartbeat.address)
	}

	cfgCustom := config.DefaultNodeConfig()
	cfgCustom.Port = 9016
	cfgCustom.Address = "http://storage-node-1:9016"
	srvCustom := NewServer(cfgCustom, store)
	if srvCustom.heartbeat.address != "http://storage-node-1:9016" {
		t.Fatalf("expected custom address http://storage-node-1:9016, got %s", srvCustom.heartbeat.address)
	}
}
