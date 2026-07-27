import { useState, useEffect, useCallback } from "react";
import { useAuth } from "../../context/AuthContext";
import { api } from "../../services/api";
import type { SavedAddress, AddressRequest } from "../../types";

export function useSavedAddresses() {
  const { user, isGuest } = useAuth();
  const [saved, setSaved] = useState<SavedAddress[]>([]);

  useEffect(() => {
    let isActive = true;

    async function loadSaved() {
      if (!user || isGuest) {
        setSaved([]);
        return;
      }
      try {
        const addresses = await api.getSavedAddresses(user.id);
        if (isActive) {
          setSaved(addresses);
        }
      } catch {
        if (isActive) {
          setSaved([]);
        }
      }
    }

    void loadSaved();
    return () => {
      isActive = false;
    };
  }, [user, isGuest]);

  const refresh = useCallback(async () => {
    if (!user || isGuest) {
      setSaved([]);
      return;
    }
    try {
      setSaved(await api.getSavedAddresses(user.id));
    } catch {
      setSaved([]);
    }
  }, [user, isGuest]);

  const savePlace = useCallback(async (data: AddressRequest) => {
    if (!user || isGuest) return;
    await api.createSavedAddress(user.id, data);
    refresh();
  }, [user, isGuest, refresh]);

  // No reliable place_id from Nominatim, so match on coords + name
  const isSaved = useCallback((lat: number, lon: number, formattedAddress: string) => {
    return saved.some(
      (s) =>
        Math.abs(s.address.latitude - lat) < 1e-5 &&
        Math.abs(s.address.longitude - lon) < 1e-5 &&
        s.address.formatted_address === formattedAddress
    );
  }, [saved]);

  return { saved, savePlace, isSaved, isEnabled: !!user && !isGuest };
}
