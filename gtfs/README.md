# Singapore transport data

```text
raw/         Source ZIP files
fixtures/    Small test feeds
generated/   Runtime graph files
```

Build from `backend/`:

```powershell
go run ./cmd/gtfs-build
go run ./cmd/footpath-build
```

Production needs `singapore.graph` and `singapore.walk`. Generated files stay out of the frontend bundle.
