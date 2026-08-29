"use client";

import React, { useState, useEffect } from "react";
import StorageStats from "../../components/StorageStats";
import ClusterTopology from "../../components/ClusterTopology";
import { fetchClusterNodes, fetchClusterMetrics, NodeRecord, ClusterMetrics } from "../../lib/api";

export default function AdminPage() {
  const [nodes, setNodes] = useState<NodeRecord[]>([]);
  const [metrics, setMetrics] = useState<ClusterMetrics | null>(null);

  const loadClusterData = async () => {
    try {
      const [nodesData, metricsData] = await Promise.all([
        fetchClusterNodes().catch(() => []),
        fetchClusterMetrics().catch(() => null),
      ]);
      setNodes(nodesData);
      setMetrics(metricsData);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadClusterData();
    const interval = setInterval(loadClusterData, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div style={{ maxWidth: "1200px", margin: "0 auto", padding: "2rem" }}>
      <header style={{ marginBottom: "2rem" }}>
        <h1 style={{ fontSize: "2rem", fontWeight: "700", marginBottom: "0.5rem" }}>
          Cluster Administration & Telemetry
        </h1>
        <p style={{ color: "var(--text-muted)" }}>
          Real-time health monitoring, node heartbeat tracking, and automatic self-healing metrics.
        </p>
      </header>

      <StorageStats metrics={metrics} />
      <ClusterTopology nodes={nodes} onRefresh={loadClusterData} />
    </div>
  );
}
