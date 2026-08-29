"use client";

import React, { useState } from "react";
import { FileRecord, getDownloadUrl } from "../lib/api";

interface RestoreModalProps {
  file: FileRecord;
  onClose: () => void;
}

export default function RestoreModal({ file, onClose }: RestoreModalProps) {
  const [keyHex, setKeyHex] = useState("");

  const handleDownload = (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyHex.trim()) return;

    const downloadUrl = getDownloadUrl(file.id, keyHex.trim());
    window.open(downloadUrl, "_blank");
    onClose();
  };

  return (
    <div style={{
      position: "fixed",
      inset: 0,
      background: "rgba(0, 0, 0, 0.75)",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      zIndex: 100,
      backdropFilter: "blur(4px)",
    }}>
      <div style={{
        background: "var(--surface)",
        border: "1px solid var(--surface-border)",
        borderRadius: "12px",
        padding: "2rem",
        maxWidth: "500px",
        width: "90%",
      }}>
        <h3 style={{ fontSize: "1.25rem", marginBottom: "0.5rem" }}>
          Restore and Decrypt File
        </h3>
        <p style={{ fontSize: "0.9rem", color: "var(--text-muted)", marginBottom: "1.5rem" }}>
          Reconstructing <strong>{file.filename}</strong> from distributed chunks across healthy storage nodes.
        </p>

        <form onSubmit={handleDownload} style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
          <label style={{ fontSize: "0.85rem", color: "var(--text-muted)" }}>
            Enter 64-character AES-256 Hex Decryption Key:
          </label>
          <input
            type="text"
            placeholder="e.g. 3a7f9b01c..."
            value={keyHex}
            onChange={(e) => setKeyHex(e.target.value)}
            style={{
              padding: "0.75rem",
              background: "rgba(0, 0, 0, 0.3)",
              border: "1px solid var(--surface-border)",
              borderRadius: "6px",
              color: "var(--foreground)",
              fontFamily: "monospace",
              fontSize: "0.85rem",
            }}
          />

          <div style={{ display: "flex", justifyContent: "flex-end", gap: "0.75rem", marginTop: "1rem" }}>
            <button
              type="button"
              onClick={onClose}
              style={{
                padding: "0.6rem 1.2rem",
                background: "transparent",
                border: "1px solid var(--surface-border)",
                color: "var(--foreground)",
                borderRadius: "6px",
                cursor: "pointer",
              }}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!keyHex.trim()}
              style={{
                padding: "0.6rem 1.2rem",
                background: keyHex.trim() ? "var(--primary)" : "var(--surface-border)",
                color: "#fff",
                border: "none",
                borderRadius: "6px",
                fontWeight: "600",
                cursor: keyHex.trim() ? "pointer" : "not-allowed",
              }}
            >
              Fetch & Decrypt
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
