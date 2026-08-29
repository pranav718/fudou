package config

type CoordinatorConfig struct {
	Port              int
	MetadataDBPath    string
	ReplicationFactor int
	AuthSecret        string
}

type NodeConfig struct {
	NodeID          string
	Port            int
	StorageDirPath  string
	CoordinatorURL  string
	HeartbeatSec    int
}

func DefaultCoordinatorConfig() CoordinatorConfig {
	return CoordinatorConfig{
		Port:              8080,
		MetadataDBPath:    "./data/metadata.json",
		ReplicationFactor: 3,
		AuthSecret:        "fudou-secret-key-32bytes-secure-00",
	}
}

func DefaultNodeConfig() NodeConfig {
	return NodeConfig{
		NodeID:         "node-1",
		Port:           9001,
		StorageDirPath: "./data/chunks",
		CoordinatorURL: "http://localhost:8080",
		HeartbeatSec:   5,
	}
}
