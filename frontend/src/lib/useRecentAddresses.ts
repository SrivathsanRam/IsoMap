import { useState, useEffect, useCallback } from "react";
import { useAuth } from "@/context/AuthContext";
import { api } from "@/services/api";
import type { AddressSearch, AddressRequest } from "@/types";

export function useRecentAddresses(limit = 5) {
  const { user, isGuest } = useAuth();
  const [recents, setRecents] = useState<AddressSearch[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    let isActive = true;

    async function loadRecents() {
      if (!user || isGuest) {
        setRecents([]);
        setError("");
        return;
      }
      try {
        const addresses = await api.getRecentAddresses(user.id, limit);
        if (isActive) {
          setRecents(addresses);
          setError("");
        }
      } catch {
        if (isActive) {
          setRecents([]);
          setError("Could not load recent searches");
        }
      }
    }

    void loadRecents();
    return () => {
      isActive = false;
    };
  }, [user, isGuest, limit]);

  const refresh = useCallback(async () => {
    if (!user || isGuest) {
      setRecents([]);
      setError("");
      return;
    }
    try {
      setRecents(await api.getRecentAddresses(user.id, limit));
      setError("");
    } catch {
      setRecents([]);
      setError("Could not load recent searches");
    }
  }, [user, isGuest, limit]);

  const addRecent = useCallback(async (data: AddressRequest) => {
    if (!user || isGuest) return;
    try {
      await api.createRecentAddress(user.id, data);
      await refresh();
    } catch {
      setError("Could not save recent search");
    }
  }, [user, isGuest, refresh]);

  return { recents, addRecent, error, isEnabled: !!user && !isGuest };
}
