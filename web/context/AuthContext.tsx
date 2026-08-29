"use client";

import React, { createContext, useContext, useState, useEffect } from "react";

interface AuthState {
  userId: string;
  role: "user" | "admin";
  token: string | null;
  login: (userId: string, role: "user" | "admin", token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthState>({
  userId: "demo-user",
  role: "user",
  token: null,
  login: () => {},
  logout: () => {},
});

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
  const [userId, setUserId] = useState<string>("demo-user");
  const [role, setRole] = useState<"user" | "admin">("user");
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    const savedUser = localStorage.getItem("fudou_user");
    const savedRole = localStorage.getItem("fudou_role") as "user" | "admin";
    const savedToken = localStorage.getItem("fudou_token");
    if (savedUser) setUserId(savedUser);
    if (savedRole) setRole(savedRole);
    if (savedToken) setToken(savedToken);
  }, []);

  const login = (uid: string, r: "user" | "admin", t: string) => {
    setUserId(uid);
    setRole(r);
    setToken(t);
    localStorage.setItem("fudou_user", uid);
    localStorage.setItem("fudou_role", r);
    localStorage.setItem("fudou_token", t);
  };

  const logout = () => {
    setUserId("demo-user");
    setRole("user");
    setToken(null);
    localStorage.removeItem("fudou_user");
    localStorage.removeItem("fudou_role");
    localStorage.removeItem("fudou_token");
  };

  return (
    <AuthContext.Provider value={{ userId, role, token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
