"use client";

import React, { useState } from "react";
import { FileRecord, deleteFile } from "../lib/api";
import RestoreModal from "./RestoreModal";

interface FileListProps {
  files: FileRecord[];
  onRefresh: () => void;
}

export default function FileList({ files, onRefresh }: FileListProps) {
  const [selectedFile, setSelectedFile] = useState<FileRecord | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleDelete = async (fileId: string) => {
    if (!confirm("Are you sure you want to delete this backup from all storage nodes?")) return;

    setDeletingId(fileId);
    try {
      await deleteFile(fileId);
      onRefresh();
    } catch (err: any) {
      alert(err.message || "Failed to delete file");
    } finally {
      setDeletingId(null);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  return (
    <div style={{
      background: "var(--surface)",
      border: "1px solid var(--surface-border)",
      borderRadius: "10px",
      padding: "1.5rem",
    }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
        <h2 style={{ fontSize: "1.2rem", fontWeight: "600" }}>Active Backups ({files.length})</h2>
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
          Refresh
        </button>
      </div>

      {files.length === 0 ? (
        <p style={{ color: "var(--text-muted)", fontSize: "0.9rem", padding: "1.5rem 0", textAlign: "center" }}>
          No backup archives present in cluster. Upload a file above to begin.
        </p>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: "0.75rem" }}>
          {files.map((file) => (
            <div
              key={file.id}
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                padding: "1rem",
                background: "rgba(0, 0, 0, 0.2)",
                border: "1px solid var(--surface-border)",
                borderRadius: "8px",
              }}
            >
              <div>
                <h4 style={{ fontSize: "0.95rem", fontWeight: "600", marginBottom: "0.25rem" }}>
                  {file.filename}
                </h4>
                <p style={{ fontSize: "0.8rem", color: "var(--text-muted)" }}>
                  Size: {formatBytes(file.size)} | Chunks: {file.chunk_count} | Checksum: {file.checksum.substring(0, 12)}...
                </p>
              </div>

              <div style={{ display: "flex", gap: "0.5rem" }}>
                <button
                  onClick={() => setSelectedFile(file)}
                  style={{
                    padding: "0.4rem 0.9rem",
                    background: "rgba(59, 130, 246, 0.15)",
                    border: "1px solid rgba(59, 130, 246, 0.3)",
                    color: "var(--primary)",
                    borderRadius: "6px",
                    fontSize: "0.85rem",
                    fontWeight: "600",
                    cursor: "pointer",
                  }}
                >
                  Restore
                </button>
                <button
                  onClick={() => handleDelete(file.id)}
                  disabled={deletingId === file.id}
                  style={{
                    padding: "0.4rem 0.9rem",
                    background: "rgba(239, 68, 68, 0.15)",
                    border: "1px solid rgba(239, 68, 68, 0.3)",
                    color: "var(--danger)",
                    borderRadius: "6px",
                    fontSize: "0.85rem",
                    fontWeight: "600",
                    cursor: deletingId === file.id ? "not-allowed" : "pointer",
                  }}
                >
                  {deletingId === file.id ? "Deleting..." : "Delete"}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {selectedFile && (
        <RestoreModal file={selectedFile} onClose={() => setSelectedFile(null)} />
      )}
    </div>
  );
}
