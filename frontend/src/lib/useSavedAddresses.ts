import { useState, useEffect, useCallback } from "react";
import { useAuth } from "../../context/AuthContext";
import { api } from "../../services/api";
import type { SavedAddress, AddressRequest } from "../../types";

export function useSavedAddresses() {
  const { user, isGuest } = useAuth();
  const [saved, setSaved] = useState<SavedAddress[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let isActive = true;

    async function loadSaved() {
      if (!user || isGuest) {
        setSaved([]);
        setError("");
        return;
      }
      try {
        const addresses = await api.getSavedAddresses(user.id);
        if (isActive) {
          setSaved(addresses);
          setError("");
        }
      } catch {
        if (isActive) {
          setSaved([]);
          setError("Could not load saved places");
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
      setError("");
      return;
    }
    try {
      setSaved(await api.getSavedAddresses(user.id));
      setError("");
    } catch {
      setSaved([]);
      setError("Could not load saved places");
    }
  }, [user, isGuest]);

  const savePlace = useCallback(async (data: AddressRequest) => {
    if (!user || isGuest) return;
    setIsSaving(true);
    setError("");
    try {
      await api.createSavedAddress(user.id, data);
      await refresh();
    } catch {
      setError("Could not save place");
    } finally {
      setIsSaving(false);
    }
  }, [user, isGuest, refresh]);

  const isSaved = useCallback((lat: number, lon: number, formattedAddress: string) => {
    return saved.some(
      (savedAddress) =>
        Math.abs(savedAddress.address.latitude - lat) < 1e-5 &&
        Math.abs(savedAddress.address.longitude - lon) < 1e-5 &&
        savedAddress.address.formatted_address === formattedAddress,
    );
  }, [saved]);

  return { saved, savePlace, isSaved, isSaving, error, isEnabled: !!user && !isGuest };
}
