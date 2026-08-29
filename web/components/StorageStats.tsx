"use client";

import React from "react";
import { ClusterMetrics } from "../lib/api";

interface StorageStatsProps {
  metrics: ClusterMetrics | null;
}

export default function StorageStats({ metrics }: StorageStatsProps) {
  const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const statCards = [
    {
      title: "Total Backups",
      value: metrics ? metrics.total_files.toString() : "0",
      subtitle: "Unique files registered in metadata",
    },
    {
      title: "Logical Data Volume",
      value: metrics ? formatBytes(metrics.total_bytes) : "0 B",
      subtitle: "Uncompressed payload size",
    },
    {
      title: "Cluster Physical Storage",
      value: metrics ? `${formatBytes(Number(metrics.total_used))} / ${formatBytes(Number(metrics.total_capacity))}` : "0 B / 0 B",
      subtitle: "Distributed across storage nodes",
    },
    {
      title: "Replication Factor",
      value: metrics ? `${metrics.replication_factor}x` : "3x",
      subtitle: "Fault tolerance redundancy level",
    },
  ];

  return (
    <div style={{
      display: "grid",
      gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))",
      gap: "1rem",
      marginBottom: "2rem",
    }}>
      {statCards.map((card, idx) => (
        <div
          key={idx}
          style={{
            background: "var(--surface)",
            border: "1px solid var(--surface-border)",
            borderRadius: "10px",
            padding: "1.5rem",
            display: "flex",
            flexDirection: "column",
            gap: "0.5rem",
          }}
        >
          <span style={{ fontSize: "0.85rem", color: "var(--text-muted)", fontWeight: "500" }}>
            {card.title}
          </span>
          <span style={{ fontSize: "1.75rem", fontWeight: "700", color: "var(--foreground)" }}>
            {card.value}
          </span>
          <span style={{ fontSize: "0.75rem", color: "var(--text-muted)" }}>
            {card.subtitle}
          </span>
        </div>
      ))}
    </div>
  );
}
