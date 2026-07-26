import { useState, useEffect, useCallback } from "react";
import { useAuth } from "@/context/AuthContext";
import { api } from "@/services/api";
import type { AddressSearch, AddressRequest } from "@/types";

export function useRecentAddresses(limit = 5) {
  const { user, isGuest } = useAuth();
  const [recents, setRecents] = useState<AddressSearch[]>([]);

  const refresh = useCallback(() => {
    if (!user || isGuest) {
      setRecents([]);
      return;
    }
    api.getRecentAddresses(user.id, limit)
      .then(setRecents)
      .catch(() => setRecents([]));
  }, [user, isGuest, limit]);

  useEffect(() => { refresh(); }, [refresh]);

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