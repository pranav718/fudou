"use client";

import React, { useState } from "react";
import { uploadFile, BackupResult } from "../lib/api";
import { useAuth } from "../context/AuthContext";

interface FileUploadProps {
  onUploadSuccess: () => void;
}

export default function FileUpload({ onUploadSuccess }: FileUploadProps) {
  const { userId } = useAuth();
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<BackupResult | null>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setError(null);
      setResult(null);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) return;

    setUploading(true);
    setError(null);
    setResult(null);

    try {
      const res = await uploadFile(file, userId);
      setResult(res);
      setFile(null);
      onUploadSuccess();
    } catch (err: any) {
      setError(err.message || "Upload failed");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div style={{
      background: "var(--surface)",
      border: "1px solid var(--surface-border)",
      borderRadius: "10px",
      padding: "1.5rem",
      marginBottom: "2rem",
    }}>
      <h2 style={{ fontSize: "1.2rem", fontWeight: "600", marginBottom: "1rem" }}>
        Upload and Distributed Backup
      </h2>

      <form onSubmit={handleUpload} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
        <input
          type="file"
          onChange={handleFileChange}
          style={{
            padding: "0.8rem",
            background: "rgba(0,0,0,0.2)",
            border: "1px dashed var(--surface-border)",
            borderRadius: "6px",
            color: "var(--foreground)",
            cursor: "pointer",
          }}
        />

        <button
          type="submit"
          disabled={!file || uploading}
          style={{
            padding: "0.75rem 1.5rem",
            background: uploading ? "var(--surface-border)" : "var(--primary)",
            color: "#fff",
            border: "none",
            borderRadius: "6px",
            fontWeight: "600",
            cursor: uploading || !file ? "not-allowed" : "pointer",
            alignSelf: "flex-start",
            transition: "background 0.2s ease",
          }}
        >
          {uploading ? "Chunking and Encrypting..." : "Start Distributed Backup"}
        </button>
      </form>

      {error && (
        <div style={{
          marginTop: "1rem",
          padding: "0.75rem 1rem",
          background: "rgba(239, 68, 68, 0.15)",
          border: "1px solid rgba(239, 68, 68, 0.3)",
          color: "var(--danger)",
          borderRadius: "6px",
          fontSize: "0.9rem",
        }}>
          {error}
        </div>
      )}

      {result && (
        <div style={{
          marginTop: "1rem",
          padding: "1rem",
          background: "rgba(16, 185, 129, 0.1)",
          border: "1px solid rgba(16, 185, 129, 0.3)",
          borderRadius: "8px",
        }}>
          <h3 style={{ fontSize: "1rem", color: "var(--accent)", marginBottom: "0.5rem" }}>
            Backup Distributed Successfully
          </h3>
          <p style={{ fontSize: "0.85rem", color: "var(--text-muted)", marginBottom: "0.5rem" }}>
            File ID: <strong style={{ color: "var(--foreground)" }}>{result.FileID}</strong> | Chunks Created: <strong>{result.ChunkCount}</strong>
          </p>
          <div style={{
            background: "rgba(0, 0, 0, 0.4)",
            padding: "0.5rem 0.8rem",
            borderRadius: "4px",
            fontSize: "0.8rem",
            fontFamily: "monospace",
            wordBreak: "break-all",
          }}>
            AES-256 Decryption Key: {result.KeyHex}
          </div>
          <p style={{ fontSize: "0.75rem", color: "var(--text-muted)", marginTop: "0.5rem" }}>
            Save this decryption key. It is required to reconstruct and decrypt your file during restore.
          </p>
        </div>
      )}
    </div>
  );
}
