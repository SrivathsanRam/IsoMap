package graph

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const artifactMagic = "ISOMAPGTFS\x00"

// SHA256File returns the lowercase SHA-256 digest used to tie a generated
// artifact to its source feed.
func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open source for hashing: %w", err)
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("hash source: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// WriteArtifact atomically writes a gzip-compressed, versioned graph. The
// encoded object contains only slices and scalar values so repeated builds
// from the same input are deterministic.
func WriteArtifact(path string, artifact *Artifact) error {
	if artifact == nil {
		return fmt.Errorf("graph artifact is nil")
	}
	return writeAtomically(path, func(writer io.Writer) error {
		if _, err := io.WriteString(writer, artifactMagic); err != nil {
			return err
		}
		compressed := gzip.NewWriter(writer)
		if err := gob.NewEncoder(compressed).Encode(artifact); err != nil {
			_ = compressed.Close()
			return fmt.Errorf("encode graph: %w", err)
		}
		if err := compressed.Close(); err != nil {
			return fmt.Errorf("finish graph compression: %w", err)
		}
		return nil
	})
}

// LoadArtifact reads an artifact and rejects incompatible or internally
// inconsistent graph formats before they reach the routing engine.
func LoadArtifact(path string) (*Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open graph artifact: %w", err)
	}
	defer file.Close()

	magic := make([]byte, len(artifactMagic))
	if _, err := io.ReadFull(file, magic); err != nil {
		return nil, fmt.Errorf("read graph header: %w", err)
	}
	if string(magic) != artifactMagic {
		return nil, fmt.Errorf("not an IsoMap GTFS graph artifact")
	}
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("open graph compression: %w", err)
	}
	defer compressed.Close()

	var artifact Artifact
	if err := gob.NewDecoder(compressed).Decode(&artifact); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}
	if err := validateArtifact(&artifact); err != nil {
		return nil, err
	}
	return &artifact, nil
}

func WriteManifest(path string, manifest Manifest) error {
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode graph manifest: %w", err)
	}
	contents = append(contents, '\n')
	return writeAtomically(path, func(writer io.Writer) error {
		_, err := writer.Write(contents)
		return err
	})
}

func validateArtifact(artifact *Artifact) error {
	if artifact.Manifest.Format != FormatName {
		return fmt.Errorf("unsupported graph format %q", artifact.Manifest.Format)
	}
	if artifact.Manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported graph schema version %d; expected %d", artifact.Manifest.SchemaVersion, SchemaVersion)
	}
	graph := &artifact.Graph
	if artifact.Manifest.AgencyCount != len(graph.Agencies) ||
		artifact.Manifest.StopCount != len(graph.Stops) ||
		artifact.Manifest.RouteCount != len(graph.Routes) ||
		artifact.Manifest.ServiceCount != len(graph.Services) ||
		artifact.Manifest.TripCount != len(graph.Trips) ||
		artifact.Manifest.NodeCount != len(graph.Nodes) ||
		artifact.Manifest.EdgeCount != len(graph.Edges) {
		return fmt.Errorf("graph counts do not match its manifest")
	}
	if len(graph.FirstOut) != len(graph.Nodes)+1 {
		return fmt.Errorf("graph adjacency index has invalid length")
	}
	if len(graph.FirstOut) > 0 && graph.FirstOut[len(graph.FirstOut)-1] != uint32(len(graph.Edges)) {
		return fmt.Errorf("graph adjacency index does not cover all edges")
	}
	for index := 1; index < len(graph.FirstOut); index++ {
		if graph.FirstOut[index] < graph.FirstOut[index-1] {
			return fmt.Errorf("graph adjacency index is not monotonic")
		}
	}
	for _, edge := range graph.Edges {
		if int(edge.To) >= len(graph.Nodes) {
			return fmt.Errorf("graph edge points to unknown node %d", edge.To)
		}
	}
	return nil
}

func writeAtomically(path string, write func(io.Writer) error) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := write(temporary); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}

	if err := os.Rename(temporaryPath, path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("install output: %w", err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace existing output: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("install output: %w", err)
	}
	return nil
}
