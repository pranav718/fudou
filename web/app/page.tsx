import Link from "next/link";

export default function HomePage() {
  return (
    <main style={{ padding: "4rem 2rem", maxWidth: "1200px", margin: "0 auto" }}>
      <header style={{ marginBottom: "3.5rem" }}>
        <h1 style={{ fontSize: "2.8rem", fontWeight: "800", letterSpacing: "-1px", marginBottom: "0.75rem" }}>
          Fault-Tolerant Distributed Backup
        </h1>
        <p style={{ color: "var(--text-muted)", fontSize: "1.2rem", maxWidth: "700px", lineHeight: "1.6" }}>
          Fudou fragments files into verifiable chunks, secures each chunk with AES-256-GCM encryption, and replicates data redundantly across independent storage nodes with automatic self-healing.
        </p>
      </header>

      <section style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))", gap: "1.5rem" }}>
        <div style={{
          background: "var(--surface)",
          border: "1px solid var(--surface-border)",
          borderRadius: "12px",
          padding: "2rem",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
        }}>
          <div>
            <h2 style={{ fontSize: "1.4rem", fontWeight: "700", marginBottom: "0.75rem" }}>
              User Backup Workspace
            </h2>
            <p style={{ color: "var(--text-muted)", fontSize: "0.95rem", lineHeight: "1.5", marginBottom: "1.5rem" }}>
              Upload any file size to trigger chunk distribution, inspect generated AES-256 keys, and restore files seamlessly with checksum validation.
            </p>
          </div>
          <Link
            href="/dashboard"
            style={{
              display: "inline-block",
              padding: "0.75rem 1.25rem",
              background: "var(--primary)",
              color: "#fff",
              borderRadius: "6px",
              fontWeight: "600",
              textAlign: "center",
            }}
          >
            Launch Backup Workspace &rarr;
          </Link>
        </div>

        <div style={{
          background: "var(--surface)",
          border: "1px solid var(--surface-border)",
          borderRadius: "12px",
          padding: "2rem",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
        }}>
          <div>
            <h2 style={{ fontSize: "1.4rem", fontWeight: "700", marginBottom: "0.75rem" }}>
              Admin Cluster Topology
            </h2>
            <p style={{ color: "var(--text-muted)", fontSize: "0.95rem", lineHeight: "1.5", marginBottom: "1.5rem" }}>
              Real-time telemetry, storage node heartbeat monitor, cluster storage allocation, and self-healing status metrics.
            </p>
          </div>
          <Link
            href="/admin"
            style={{
              display: "inline-block",
              padding: "0.75rem 1.25rem",
              background: "rgba(16, 185, 129, 0.15)",
              border: "1px solid rgba(16, 185, 129, 0.3)",
              color: "var(--accent)",
              borderRadius: "6px",
              fontWeight: "600",
              textAlign: "center",
            }}
          >
            Open Cluster Console &rarr;
          </Link>
        </div>
      </section>
    </main>
  );
}
