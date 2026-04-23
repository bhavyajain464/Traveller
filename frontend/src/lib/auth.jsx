import { createContext, useContext, useEffect, useState } from "react";
import { apiRequest } from "./api";

const STORAGE_KEY = "traveller.auth.user";
const AuthContext = createContext(null);

function readStoredUser() {
  try {
    const value = window.localStorage.getItem(STORAGE_KEY);
    return value ? JSON.parse(value) : null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => readStoredUser());

  useEffect(() => {
    if (!user) {
      window.localStorage.removeItem(STORAGE_KEY);
      return;
    }

    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(user));
  }, [user]);

  const login = async ({ name, phone }) => {
    const normalizedPhone = phone.trim();
    const trimmedName = name.trim();
    let nextUser;

    try {
      const existing = await apiRequest(`/users/phone/${encodeURIComponent(normalizedPhone)}`);
      nextUser = {
        id: existing.user.id,
        name: existing.user.name,
        phone: existing.user.phone_number
      };
    } catch (error) {
      const created = await apiRequest("/users", {
        method: "POST",
        body: JSON.stringify({
          phone_number: normalizedPhone,
          name: trimmedName
        })
      });
      nextUser = {
        id: created.user.id,
        name: created.user.name,
        phone: created.user.phone_number
      };
    }

    setUser(nextUser);
    return nextUser;
  };

  const logout = () => {
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: Boolean(user), login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return value;
}
