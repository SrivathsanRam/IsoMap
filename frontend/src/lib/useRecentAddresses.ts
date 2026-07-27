import { useState, useEffect, useCallback } from "react";
import { useAuth } from "@/context/AuthContext";
import { api } from "@/services/api";
import type { AddressSearch, AddressRequest } from "@/types";

export function useRecentAddresses(limit = 5) {
  const { user, isGuest } = useAuth();
  const [recents, setRecents] = useState<AddressSearch[]>([]);

  useEffect(() => {
    let isActive = true;

    async function loadRecents() {
      if (!user || isGuest) {
        setRecents([]);
        return;
      }
      try {
        const addresses = await api.getRecentAddresses(user.id, limit);
        if (isActive) {
          setRecents(addresses);
        }
      } catch {
        if (isActive) {
          setRecents([]);
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
      return;
    }
    try {
      setRecents(await api.getRecentAddresses(user.id, limit));
    } catch {
      setRecents([]);
    }
  }, [user, isGuest, limit]);

  const addRecent = useCallback(async (data: AddressRequest) => {
    if (!user || isGuest) return;
    try {
      await api.createRecentAddress(user.id, data);
      refresh();
    } catch {
      // search history is non-critical — fail silently
    }
  }, [user, isGuest, refresh]);

  return { recents, addRecent, isEnabled: !!user && !isGuest };
}
