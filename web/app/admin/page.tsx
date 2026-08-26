import Link from "next/link";

export default function AdminPage() {
  return (
    <div style={{ padding: "2rem", maxWidth: "1200px", margin: "0 auto" }}>
      <nav style={{ marginBottom: "2rem" }}>
        <Link href="/" style={{ color: "var(--primary)", fontSize: "0.9rem" }}>
          &larr; Back to Home
        </Link>
      </nav>
      <h1 style={{ fontSize: "2rem", marginBottom: "1rem" }}>Cluster Administration</h1>
      <p style={{ color: "var(--text-muted)", marginBottom: "2rem" }}>
        Real-time telemetry, node availability, and cluster replication status.
      </p>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: "1rem" }}>
        <div style={{ background: "var(--surface)", border: "1px solid var(--surface-border)", borderRadius: "8px", padding: "1.5rem" }}>
          <h3 style={{ fontSize: "0.9rem", color: "var(--text-muted)" }}>Cluster Status</h3>
          <p style={{ fontSize: "1.5rem", fontWeight: "700", color: "var(--accent)", marginTop: "0.5rem" }}>Online</p>
        </div>
        <div style={{ background: "var(--surface)", border: "1px solid var(--surface-border)", borderRadius: "8px", padding: "1.5rem" }}>
          <h3 style={{ fontSize: "0.9rem", color: "var(--text-muted)" }}>Active Nodes</h3>
          <p style={{ fontSize: "1.5rem", fontWeight: "700", marginTop: "0.5rem" }}>0</p>
        </div>
        <div style={{ background: "var(--surface)", border: "1px solid var(--surface-border)", borderRadius: "8px", padding: "1.5rem" }}>
          <h3 style={{ fontSize: "0.9rem", color: "var(--text-muted)" }}>Replication Factor</h3>
          <p style={{ fontSize: "1.5rem", fontWeight: "700", marginTop: "0.5rem" }}>3x</p>
        </div>
      </div>
    </div>
  );
}
