import Link from "next/link";

export default function DashboardPage() {
  return (
    <div style={{ padding: "2rem", maxWidth: "1200px", margin: "0 auto" }}>
      <nav style={{ marginBottom: "2rem" }}>
        <Link href="/" style={{ color: "var(--primary)", fontSize: "0.9rem" }}>
          &larr; Back to Home
        </Link>
      </nav>
      <h1 style={{ fontSize: "2rem", marginBottom: "1rem" }}>User Backup Dashboard</h1>
      <p style={{ color: "var(--text-muted)", marginBottom: "2rem" }}>
        Manage backups, initiate file chunking, and view encryption status.
      </p>
      <div style={{ background: "var(--surface)", border: "1px solid var(--surface-border)", borderRadius: "8px", padding: "2rem" }}>
        <p style={{ color: "var(--text-muted)" }}>No backups uploaded yet.</p>
      </div>
    </div>
  );
}
