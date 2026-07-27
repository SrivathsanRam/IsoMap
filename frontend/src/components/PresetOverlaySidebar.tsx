import { useMemo, useState } from "react";
import { Plus, ThumbsDown, ThumbsUp, X } from "lucide-react";
import cdcCafes from "../assets/cdc_cafes_singapore.json";
import famousPlaces from "../assets/famous_places_singapore.json";
import historicalLandmarks from "../assets/historical_landmarks_singapore.json";
import malls from "../assets/malls_singapore.json";
import { Place, PlaceInput } from "./SearchBar";

const communityPresetKey = "isomap-community-presets";
const presetVoteKey = "isomap-community-preset-votes";

export type PresetLocation = {
  name: string;
  address: string;
  latitude: number;
  longitude: number;
};

export type PresetOverlay = {
  id: string;
  name: string;
  source: "provided" | "community";
  locations: PresetLocation[];
  upvotes?: number;
  downvotes?: number;
};

type PresetOverlaySidebarProps = {
  activePresetIDs: string[];
  onTogglePreset: (presetID: string) => void;
  onPresetsChange: (presets: PresetOverlay[]) => void;
};

const providedPresets: PresetOverlay[] = [
  makeProvidedPreset("provided-famous", "Famous Places", famousPlaces),
  makeProvidedPreset("provided-cafes", "CDC Cafes", cdcCafes),
  makeProvidedPreset("provided-landmarks", "Historical Landmarks", historicalLandmarks),
  makeProvidedPreset("provided-malls", "Shopping Malls", malls),
].filter((preset) => preset.locations.length > 0);

const initialCommunityPresets: PresetOverlay[] = [
  {
    id: "community-study-spots",
    name: "Community Study Spots",
    source: "community",
    upvotes: 12,
    downvotes: 2,
    locations: [
      {
        name: "NUS Central Library",
        address: "12 Kent Ridge Crescent, Singapore",
        latitude: 1.2966,
        longitude: 103.7738,
      },
      {
        name: "library@orchard",
        address: "277 Orchard Road, Singapore",
        latitude: 1.3015,
        longitude: 103.8374,
      },
    ],
  },
  {
    id: "community-weekend-food",
    name: "Weekend Food Trail",
    source: "community",
    upvotes: 8,
    downvotes: 1,
    locations: [
      {
        name: "Lau Pa Sat",
        address: "18 Raffles Quay, Singapore",
        latitude: 1.2807,
        longitude: 103.8504,
      },
      {
        name: "Tiong Bahru Market",
        address: "30 Seng Poh Road, Singapore",
        latitude: 1.2852,
        longitude: 103.8321,
      },
    ],
  },
];

export function PresetOverlaySidebar({
  activePresetIDs,
  onTogglePreset,
  onPresetsChange,
}: PresetOverlaySidebarProps) {
  const [communityPresets, setCommunityPresets] = useState(loadCommunityPresets);
  const [votes, setVotes] = useState(loadVotes);
  const [query, setQuery] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  const presets = useMemo(
    () => [...providedPresets, ...communityPresets],
    [communityPresets],
  );
  const filteredPresets = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    if (!normalizedQuery) {
      return presets;
    }
    return presets.filter((preset) => {
      const haystack = [
        preset.name,
        preset.source,
        ...preset.locations.flatMap((location) => [location.name, location.address]),
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(normalizedQuery);
    });
  }, [presets, query]);

  function submitPreset(preset: PresetOverlay) {
    const nextPresets = [preset, ...communityPresets];
    setCommunityPresets(nextPresets);
    saveCommunityPresets(nextPresets);
    onPresetsChange([...providedPresets, ...nextPresets]);
    setIsCreateOpen(false);
  }

  function vote(presetID: string, direction: "up" | "down") {
    const current = votes[presetID];
    const nextVotes = {
      ...votes,
      [presetID]: current === direction ? undefined : direction,
    };
    setVotes(nextVotes);
    saveVotes(nextVotes);
  }

  return (
    <>
      <aside className="preset-sidebar">
        <div className="preset-sidebar-header">
          <h2>Preset Overlays</h2>
          <button type="button" onClick={() => setIsCreateOpen(true)}>
            <Plus size={16} />
            Create
          </button>
        </div>

        <input
          className="preset-search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search presets"
        />

        <div className="preset-list">
          {filteredPresets.map((preset) => (
            <PresetRow
              key={preset.id}
              preset={preset}
              isActive={activePresetIDs.includes(preset.id)}
              vote={votes[preset.id]}
              onToggle={() => onTogglePreset(preset.id)}
              onVote={(direction) => vote(preset.id, direction)}
            />
          ))}
          {filteredPresets.length === 0 && (
            <p className="preset-empty">No matching presets.</p>
          )}
        </div>
      </aside>

      {isCreateOpen && (
        <CreatePresetModal
          onClose={() => setIsCreateOpen(false)}
          onSubmit={submitPreset}
        />
      )}
    </>
  );
}

function PresetRow({
  preset,
  isActive,
  vote,
  onToggle,
  onVote,
}: {
  preset: PresetOverlay;
  isActive: boolean;
  vote?: "up" | "down";
  onToggle: () => void;
  onVote: (direction: "up" | "down") => void;
}) {
  const upvotes = (preset.upvotes ?? 0) + (vote === "up" ? 1 : 0);
  const downvotes = (preset.downvotes ?? 0) + (vote === "down" ? 1 : 0);

  return (
    <article className={isActive ? "preset-row active" : "preset-row"}>
      <button type="button" className="preset-main" onClick={onToggle}>
        <span>{preset.name}</span>
        <small>
          {preset.source === "provided" ? "Provided" : "Community"} · {preset.locations.length} places
        </small>
      </button>
      {preset.source === "community" && (
        <div className="preset-votes" aria-label={`${preset.name} votes`}>
          <button
            type="button"
            className={vote === "up" ? "active" : ""}
            onClick={() => onVote("up")}
            title="Upvote"
          >
            <ThumbsUp size={14} />
            {upvotes}
          </button>
          <button
            type="button"
            className={vote === "down" ? "active" : ""}
            onClick={() => onVote("down")}
            title="Downvote"
          >
            <ThumbsDown size={14} />
            {downvotes}
          </button>
        </div>
      )}
    </article>
  );
}

function CreatePresetModal({
  onClose,
  onSubmit,
}: {
  onClose: () => void;
  onSubmit: (preset: PresetOverlay) => void;
}) {
  const [name, setName] = useState("");
  const [pendingPlace, setPendingPlace] = useState<Place | null>(null);
  const [locations, setLocations] = useState<PresetLocation[]>([]);

  function addPendingPlace() {
    if (!pendingPlace) {
      return;
    }
    const location = placeToPresetLocation(pendingPlace);
    setLocations((current) => [...current, location]);
    setPendingPlace(null);
  }

  function removeLocation(index: number) {
    setLocations((current) => current.filter((_, currentIndex) => currentIndex !== index));
  }

  function submit() {
    const trimmedName = name.trim();
    if (!trimmedName || locations.length === 0) {
      return;
    }
    onSubmit({
      id: `community-${crypto.randomUUID()}`,
      name: trimmedName,
      source: "community",
      upvotes: 0,
      downvotes: 0,
      locations,
    });
  }

  return (
    <div className="preset-modal-backdrop" role="presentation">
      <section className="preset-modal" role="dialog" aria-modal="true" aria-labelledby="create-preset-title">
        <div className="preset-modal-header">
          <h2 id="create-preset-title">Create Preset</h2>
          <button type="button" onClick={onClose} title="Close">
            <X size={18} />
          </button>
        </div>

        <label className="preset-field">
          <span>Name</span>
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="e.g. Quiet study route"
          />
        </label>

        <div className="preset-location-add">
          <PlaceInput placeholder="Type an address or location" onSelect={setPendingPlace} />
          <button type="button" onClick={addPendingPlace} disabled={!pendingPlace} title="Add location">
            <Plus size={18} />
          </button>
        </div>

        <div className="preset-location-list">
          {locations.map((location, index) => (
            <div key={`${location.name}-${location.latitude}-${location.longitude}`}>
              <span>
                <strong>{location.name}</strong>
                <small>{location.address}</small>
              </span>
              <button type="button" onClick={() => removeLocation(index)} title="Remove location">
                <X size={14} />
              </button>
            </div>
          ))}
          {locations.length === 0 && (
            <p className="preset-empty">Add at least one location.</p>
          )}
        </div>

        <div className="preset-modal-actions">
          <button type="button" className="preset-secondary" onClick={onClose}>
            Cancel
          </button>
          <button type="button" className="preset-primary" onClick={submit} disabled={!name.trim() || locations.length === 0}>
            Submit overlay
          </button>
        </div>
      </section>
    </div>
  );
}

function makeProvidedPreset(
  id: string,
  name: string,
  rawLocations: { name: string; address: string; latitude: number; longitude: number }[],
): PresetOverlay {
  return {
    id,
    name,
    source: "provided",
    locations: rawLocations.filter(isValidPresetLocation),
  };
}

function isValidPresetLocation(location: PresetLocation) {
  return Number.isFinite(location.latitude) && Number.isFinite(location.longitude);
}

function placeToPresetLocation(place: Place): PresetLocation {
  return {
    name: place.display_name,
    address: place.display_name,
    latitude: Number(place.lat),
    longitude: Number(place.lon),
  };
}

function loadCommunityPresets() {
  try {
    const raw = localStorage.getItem(communityPresetKey);
    if (!raw) {
      return initialCommunityPresets;
    }
    const parsed = JSON.parse(raw) as PresetOverlay[];
    return parsed.length > 0 ? parsed : initialCommunityPresets;
  } catch {
    return initialCommunityPresets;
  }
}

function saveCommunityPresets(presets: PresetOverlay[]) {
  localStorage.setItem(communityPresetKey, JSON.stringify(presets));
}

function loadVotes() {
  try {
    const raw = localStorage.getItem(presetVoteKey);
    return raw ? (JSON.parse(raw) as Record<string, "up" | "down" | undefined>) : {};
  } catch {
    return {};
  }
}

function saveVotes(votes: Record<string, "up" | "down" | undefined>) {
  localStorage.setItem(presetVoteKey, JSON.stringify(votes));
}
