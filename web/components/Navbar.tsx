"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "../context/AuthContext";

export default function Navbar() {
  const pathname = usePathname();
  const { userId, role } = useAuth();

  const links = [
    { href: "/", label: "Overview" },
    { href: "/dashboard", label: "User Backups" },
    { href: "/admin", label: "Cluster Topology" },
  ];

  return (
    <header style={{
      borderBottom: "1px solid var(--surface-border)",
      background: "rgba(17, 24, 39, 0.8)",
      backdropFilter: "blur(12px)",
      position: "sticky",
      top: 0,
      zIndex: 50,
      padding: "0.75rem 2rem",
    }}>
      <div style={{
        maxWidth: "1200px",
        margin: "0 auto",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
      }}>
        <div style={{ display: "flex", alignItems: "center", gap: "2rem" }}>
          <Link href="/" style={{
            fontSize: "1.25rem",
            fontWeight: "700",
            letterSpacing: "-0.5px",
            color: "var(--foreground)",
            display: "flex",
            alignItems: "center",
            gap: "0.5rem",
          }}>
            <span style={{
              width: "10px",
              height: "10px",
              borderRadius: "50%",
              backgroundColor: "var(--primary)",
              display: "inline-block",
            }}></span>
            FUDOU
          </Link>

          <nav style={{ display: "flex", gap: "1rem" }}>
            {links.map((link) => {
              const active = pathname === link.href;
              return (
                <Link
                  key={link.href}
                  href={link.href}
                  style={{
                    fontSize: "0.9rem",
                    fontWeight: active ? "600" : "400",
                    color: active ? "var(--foreground)" : "var(--text-muted)",
                    padding: "0.4rem 0.8rem",
                    borderRadius: "6px",
                    background: active ? "rgba(59, 130, 246, 0.1)" : "transparent",
                    transition: "all 0.15s ease",
                  }}
                >
                  {link.label}
                </Link>
              );
            })}
          </nav>
        </div>

        <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
          <span style={{
            fontSize: "0.8rem",
            padding: "0.25rem 0.6rem",
            borderRadius: "9999px",
            background: "rgba(16, 185, 129, 0.15)",
            color: "var(--accent)",
            border: "1px solid rgba(16, 185, 129, 0.3)",
          }}>
            Node Quorum Online
          </span>
          <span style={{
            fontSize: "0.85rem",
            color: "var(--text-muted)",
          }}>
            {userId} ({role})
          </span>
        </div>
      </div>
    </header>
  );
}
