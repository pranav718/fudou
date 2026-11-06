package coordinator

import (
	"fmt"
	"testing"
)

func BenchmarkHashRing_GetNodes(b *testing.B) {
	ring := NewHashRing(128)
	for i := 0; i < 10; i++ {
		ring.AddNode(fmt.Sprintf("node-%d", i))
	}

	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("chunk-%d", i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%1000]
		nodes, err := ring.GetNodes(key, 3)
		if err != nil || len(nodes) != 3 {
			b.Fatalf("get nodes failed: %v", err)
		}
	}
}

func BenchmarkHashRing_AddRemoveNode(b *testing.B) {
	ring := NewHashRing(64)
	for i := 0; i < 8; i++ {
		ring.AddNode(fmt.Sprintf("node-%d", i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		nodeID := fmt.Sprintf("dyn-node-%d", i%10)
		ring.AddNode(nodeID)
		ring.RemoveNode(nodeID)
	}
}
