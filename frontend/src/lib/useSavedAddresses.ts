import { useState, useEffect, useCallback } from "react";
import { useAuth } from "../../context/AuthContext";
import { api } from "../../services/api";
import type { SavedAddress, AddressRequest } from "../../types";

export function useSavedAddresses() {
  const { user, isGuest } = useAuth();
  const [saved, setSaved] = useState<SavedAddress[]>([]);

  const refresh = useCallback(() => {
    if (!user || isGuest) {
      setSaved([]);
      return;
    }
    api.getSavedAddresses(user.id)
      .then(setSaved)
      .catch(() => setSaved([]));
  }, [user, isGuest]);

  useEffect(() => { refresh(); }, [refresh]);

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