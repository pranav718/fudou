import "./globals.css";
import React from "react";

export const metadata = {
  title: "Fudou Distributed Backup",
  description: "Fault-Tolerant Distributed Backup and Storage System",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
