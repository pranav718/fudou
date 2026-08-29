"use client";

import React, { useState, useEffect } from "react";
import FileUpload from "../../components/FileUpload";
import FileList from "../../components/FileList";
import { fetchFiles, FileRecord } from "../../lib/api";
import { useAuth } from "../../context/AuthContext";

export default function DashboardPage() {
  const { userId } = useAuth();
  const [files, setFiles] = useState<FileRecord[]>([]);
  const [loading, setLoading] = useState(true);

  const loadUserFiles = async () => {
    try {
      setLoading(true);
      const data = await fetchFiles(userId);
      setFiles(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadUserFiles();
  }, [userId]);

  return (
    <div style={{ maxWidth: "1200px", margin: "0 auto", padding: "2rem" }}>
      <header style={{ marginBottom: "2rem" }}>
        <h1 style={{ fontSize: "2rem", fontWeight: "700", marginBottom: "0.5rem" }}>
          User Backups
        </h1>
        <p style={{ color: "var(--text-muted)" }}>
          Manage your distributed file archives, trigger chunking, and decrypt restored streams.
        </p>
      </header>

      <FileUpload onUploadSuccess={loadUserFiles} />

      {loading ? (
        <div style={{ textAlign: "center", padding: "2rem", color: "var(--text-muted)" }}>
          Loading backup registry...
        </div>
      ) : (
        <FileList files={files} onRefresh={loadUserFiles} />
      )}
    </div>
  );
}
