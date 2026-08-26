import Link from "next/link";

export default function HomePage() {
  return (
    <main style={{ padding: "4rem 2rem", maxWidth: "1200px", margin: "0 auto" }}>
      <header style={{ marginBottom: "3rem" }}>
        <h1 style={{ fontSize: "2.5rem", fontWeight: "700", marginBottom: "0.5rem" }}>
          Fudou Distributed Backup System
        </h1>
        <p style={{ color: "var(--text-muted)", fontSize: "1.1rem" }}>
          High-performance, fault-tolerant distributed storage and backup engine.
        </p>
      </header>

      <section style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))", gap: "1.5rem" }}>
        <div style={{ background: "var(--surface)", border: "1px solid var(--surface-border)", borderRadius: "8px", padding: "1.5rem" }}>
          <h2 style={{ fontSize: "1.25rem", marginBottom: "0.5rem" }}>User Dashboard</h2>
          <p style={{ color: "var(--text-muted)", marginBottom: "1rem" }}>
            Upload files, manage encrypted backups, and restore distributed data chunks.
          </p>
          <Link href="/dashboard" style={{ color: "var(--primary)", fontWeight: "600" }}>
            Open Dashboard &rarr;
          </Link>
        </div>

        <div style={{ background: "var(--surface)", border: "1px solid var(--surface-border)", borderRadius: "8px", padding: "1.5rem" }}>
          <h2 style={{ fontSize: "1.25rem", marginBottom: "0.5rem" }}>Admin Console</h2>
          <p style={{ color: "var(--text-muted)", marginBottom: "1rem" }}>
            Monitor storage nodes health, cluster replication factor, and node heartbeats.
          </p>
          <Link href="/admin" style={{ color: "var(--accent)", fontWeight: "600" }}>
            Open Admin &rarr;
          </Link>
        </div>
      </section>
    </main>
  );
}
