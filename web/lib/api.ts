export interface FileRecord {
  id: string;
  user_id: string;
  filename: string;
  mime_type: string;
  size: number;
  checksum: string;
  chunk_count: number;
  created_at: string;
  updated_at: string;
}

export interface NodeRecord {
  id: string;
  address: string;
  status: string;
  capacity: number;
  used_bytes: number;
  last_seen: string;
}

export interface ClusterMetrics {
  total_files: number;
  total_bytes: number;
  total_capacity: number;
  total_used: number;
  active_nodes: number;
  replication_factor: number;
}

export interface BackupResult {
  FileID: string;
  Filename: string;
  TotalSize: number;
  Checksum: string;
  ChunkCount: number;
  KeyHex: string;
}

const API_BASE = process.env.NEXT_PUBLIC_COORDINATOR_URL || "http://localhost:8080";

export async function fetchFiles(userId?: string): Promise<FileRecord[]> {
  const url = userId ? `${API_BASE}/api/files?user_id=${encodeURIComponent(userId)}` : `${API_BASE}/api/files`;
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`Failed to fetch files: ${res.statusText}`);
  }
  return res.json();
}

export async function uploadFile(file: File, userId: string): Promise<BackupResult> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("user_id", userId);

  const res = await fetch(`${API_BASE}/api/files`, {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Upload failed: ${text || res.statusText}`);
  }

  return res.json();
}

export async function deleteFile(fileId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/files/${fileId}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    throw new Error(`Delete failed: ${res.statusText}`);
  }
}

export function getDownloadUrl(fileId: string, keyHex: string): string {
  return `${API_BASE}/api/files/${fileId}/download?key=${encodeURIComponent(keyHex)}`;
}

export async function fetchClusterNodes(): Promise<NodeRecord[]> {
  const res = await fetch(`${API_BASE}/api/admin/nodes`);
  if (!res.ok) {
    throw new Error(`Failed to fetch nodes: ${res.statusText}`);
  }
  return res.json();
}

export async function fetchClusterMetrics(): Promise<ClusterMetrics> {
  const res = await fetch(`${API_BASE}/api/admin/metrics`);
  if (!res.ok) {
    throw new Error(`Failed to fetch metrics: ${res.statusText}`);
  }
  return res.json();
}
