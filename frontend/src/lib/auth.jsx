import { createContext, useContext, useEffect, useState } from "react";
import { apiRequest, clearAuthToken, getAuthToken, setAuthToken } from "./api";

const AuthContext = createContext(null);

function mapUser(user) {
  if (!user) {
    return null;
  }

  return {
    id: user.id,
    name: user.name,
    email: user.email || "",
    phone: user.phone_number || "",
    avatarUrl: user.avatar_url || "",
    authProvider: user.auth_provider || "unknown"
  };
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [isLoading, setIsLoading] = useState(() => Boolean(getAuthToken()));

  useEffect(() => {
    let ignore = false;

    async function hydrateUser() {
      const token = getAuthToken();
      if (!token) {
        setIsLoading(false);
        return;
      }

      try {
        const payload = await apiRequest("/auth/me");
        if (!ignore) {
          setUser(mapUser(payload.user));
        }
      } catch {
        clearAuthToken();
        if (!ignore) {
          setUser(null);
        }
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    hydrateUser();
    return () => {
      ignore = true;
    };
  }, []);

  const loginWithGoogle = async (credential) => {
    const payload = await apiRequest("/auth/google", {
      method: "POST",
      body: JSON.stringify({ credential })
    });

    setAuthToken(payload.token);
    const nextUser = mapUser(payload.user);
    setUser(nextUser);
    setIsLoading(false);
    return nextUser;
  };

  const logout = async () => {
    try {
      if (getAuthToken()) {
        await apiRequest("/auth/logout", { method: "POST" });
      }
    } catch {
      // Clear local auth even if the server-side session is already gone.
    } finally {
      clearAuthToken();
      setUser(null);
      setIsLoading(false);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: Boolean(user),
        isLoading,
        loginWithGoogle,
        logout
      }}
    >
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
