"use client";

import React from "react";
import { NodeRecord } from "../lib/api";

interface ClusterTopologyProps {
  nodes: NodeRecord[];
  onRefresh: () => void;
}

export default function ClusterTopology({ nodes, onRefresh }: ClusterTopologyProps) {
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  return (
    <div style={{
      background: "var(--surface)",
      border: "1px solid var(--surface-border)",
      borderRadius: "10px",
      padding: "1.5rem",
      marginBottom: "2rem",
    }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1.5rem" }}>
        <div>
          <h2 style={{ fontSize: "1.2rem", fontWeight: "600" }}>Storage Node Topology</h2>
          <p style={{ fontSize: "0.85rem", color: "var(--text-muted)", marginTop: "0.2rem" }}>
            Real-time health, disk allocation, and heartbeat status across distributed nodes.
          </p>
        </div>
        <button
          onClick={onRefresh}
          style={{
            padding: "0.4rem 0.8rem",
            background: "rgba(255,255,255,0.05)",
            border: "1px solid var(--surface-border)",
            color: "var(--foreground)",
            borderRadius: "6px",
            fontSize: "0.85rem",
            cursor: "pointer",
          }}
        >
          Refresh Topology
        </button>
      </div>

      {nodes.length === 0 ? (
        <div style={{
          padding: "2rem",
          textAlign: "center",
          background: "rgba(0,0,0,0.2)",
          borderRadius: "8px",
          color: "var(--text-muted)",
        }}>
          No storage nodes currently registered. Launch storage nodes via Docker or CLI.
        </div>
      ) : (
        <div style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
          gap: "1rem",
        }}>
          {nodes.map((node) => {
            const isOnline = node.status === "online";
            const percent = node.capacity > 0 ? Math.min(100, Math.round((node.used_bytes / node.capacity) * 100)) : 0;

            return (
              <div
                key={node.id}
                style={{
                  background: "rgba(0, 0, 0, 0.2)",
                  border: `1px solid ${isOnline ? "var(--surface-border)" : "rgba(239, 68, 68, 0.4)"}`,
                  borderRadius: "8px",
                  padding: "1.25rem",
                  display: "flex",
                  flexDirection: "column",
                  gap: "0.75rem",
                }}
              >
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <span style={{ fontWeight: "700", fontSize: "1rem" }}>{node.id}</span>
                  <span style={{
                    fontSize: "0.75rem",
                    padding: "0.2rem 0.5rem",
                    borderRadius: "9999px",
                    background: isOnline ? "rgba(16, 185, 129, 0.15)" : "rgba(239, 68, 68, 0.15)",
                    color: isOnline ? "var(--accent)" : "var(--danger)",
                    border: `1px solid ${isOnline ? "rgba(16, 185, 129, 0.3)" : "rgba(239, 68, 68, 0.3)"}`,
                  }}>
                    {node.status.toUpperCase()}
                  </span>
                </div>

                <div style={{ fontSize: "0.85rem", color: "var(--text-muted)" }}>
                  Endpoint: <span style={{ color: "var(--foreground)", fontFamily: "monospace" }}>{node.address}</span>
                </div>

                <div>
                  <div style={{ display: "flex", justifyContent: "space-between", fontSize: "0.8rem", marginBottom: "0.25rem" }}>
                    <span style={{ color: "var(--text-muted)" }}>Storage Used</span>
                    <span>{formatBytes(node.used_bytes)} / {formatBytes(node.capacity)} ({percent}%)</span>
                  </div>
                  <div style={{
                    width: "100%",
                    height: "6px",
                    background: "rgba(255,255,255,0.1)",
                    borderRadius: "3px",
                    overflow: "hidden",
                  }}>
                    <div style={{
                      width: `${percent}%`,
                      height: "100%",
                      background: percent > 85 ? "var(--danger)" : "var(--primary)",
                      transition: "width 0.3s ease",
                    }} />
                  </div>
                </div>

                <div style={{ fontSize: "0.75rem", color: "var(--text-muted)" }}>
                  Last Heartbeat: {new Date(node.last_seen).toLocaleTimeString()}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
