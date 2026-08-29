package api

type ClusterMetrics struct {
	TotalFiles     int   `json:"total_files"`
	TotalBytes     int   `json:"total_bytes"`
	TotalCapacity  int64 `json:"total_capacity"`
	TotalUsed      int64 `json:"total_used"`
	ActiveNodes    int   `json:"active_nodes"`
	ReplicationFactor int `json:"replication_factor"`
}
