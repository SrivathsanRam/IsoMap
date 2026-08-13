package walking

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

const artifactMagic = "ISOMAPWALK\x00"

func WriteArtifact(path string, artifact *Artifact) error {
	if err := validateArtifact(artifact, true); err != nil {
		return err
	}
	return writeAtomically(path, func(writer io.Writer) error {
		if _, err := io.WriteString(writer, artifactMagic); err != nil {
			return fmt.Errorf("write footpath artifact header: %w", err)
		}
		compressed := gzip.NewWriter(writer)
		if err := gob.NewEncoder(compressed).Encode(artifact); err != nil {
			_ = compressed.Close()
			return fmt.Errorf("encode footpath artifact: %w", err)
		}
		if err := compressed.Close(); err != nil {
			return fmt.Errorf("finish footpath artifact compression: %w", err)
		}
		return nil
	})
}

func LoadArtifact(path string) (*Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open footpath artifact: %w", err)
	}
	defer file.Close()
	magic := make([]byte, len(artifactMagic))
	if _, err := io.ReadFull(file, magic); err != nil {
		return nil, fmt.Errorf("read footpath artifact header: %w", err)
	}
	if string(magic) != artifactMagic {
		return nil, fmt.Errorf("not an IsoMap footpath raster artifact")
	}
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("open footpath artifact compression: %w", err)
	}
	decoder := gob.NewDecoder(compressed)
	var artifact Artifact
	if err := decoder.Decode(&artifact); err != nil {
		_ = compressed.Close()
		return nil, fmt.Errorf("decode footpath artifact: %w", err)
	}
	// Reading to EOF validates the gzip checksum instead of accepting a valid
	// gob prefix followed by a corrupt compressed tail.
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		_ = compressed.Close()
		if err == nil {
			return nil, fmt.Errorf("footpath artifact contains trailing gob data")
		}
		return nil, fmt.Errorf("finish decoding footpath artifact: %w", err)
	}
	if err := compressed.Close(); err != nil {
		return nil, fmt.Errorf("close footpath artifact compression: %w", err)
	}
	if err := validateArtifact(&artifact, true); err != nil {
		return nil, err
	}
	return &artifact, nil
}

func WriteManifest(path string, manifest Manifest) error {
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode footpath manifest: %w", err)
	}
	contents = append(contents, '\n')
	return writeAtomically(path, func(writer io.Writer) error {
		_, err := writer.Write(contents)
		return err
	})
}

func validateArtifact(artifact *Artifact, verifyComponents bool) error {
	if artifact == nil {
		return fmt.Errorf("footpath artifact is nil")
	}
	manifest, network := &artifact.Manifest, &artifact.Network
	if manifest.Format != FormatName {
		return fmt.Errorf("unsupported footpath format %q", manifest.Format)
	}
	if manifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported footpath schema version %d; expected %d", manifest.SchemaVersion, SchemaVersion)
	}
	if manifest.Projection != projectionName {
		return fmt.Errorf("unsupported footpath projection %q", manifest.Projection)
	}
	if manifest.SourceFile == "" {
		return fmt.Errorf("footpath source filename is empty")
	}
	if len(manifest.SourceSHA256) != sha256HexLength {
		return fmt.Errorf("footpath source SHA-256 is invalid")
	}
	if _, err := hex.DecodeString(manifest.SourceSHA256); err != nil {
		return fmt.Errorf("footpath source SHA-256 is invalid")
	}
	if !finite(network.MinEasting) || !finite(network.MinNorthing) || !finite(network.CellSizeMeters) || network.CellSizeMeters <= 0 {
		return fmt.Errorf("footpath network origin or cell size is invalid")
	}
	if network.Columns <= 0 || network.Rows <= 0 || int64(network.Columns) > math.MaxInt64/int64(network.Rows) {
		return fmt.Errorf("footpath network dimensions are invalid")
	}
	cellCount := int64(network.Columns) * int64(network.Rows)
	if cellCount > int64(maxInt()) {
		return fmt.Errorf("footpath network is too large for this platform")
	}
	wantWords := (cellCount + 63) / 64
	if int64(len(network.WalkableBits)) != wantWords {
		return fmt.Errorf("footpath bitset has %d words; expected %d", len(network.WalkableBits), wantWords)
	}
	if remainder := uint(cellCount & 63); remainder != 0 && network.WalkableBits[len(network.WalkableBits)-1]>>remainder != 0 {
		return fmt.Errorf("footpath bitset has set bits beyond the raster bounds")
	}
	if manifest.Columns != network.Columns || manifest.Rows != network.Rows || manifest.CellSizeMeters != network.CellSizeMeters {
		return fmt.Errorf("footpath network dimensions do not match its manifest")
	}
	if err := validateBounds(manifest.Bounds, "footpath manifest"); err != nil {
		return err
	}
	gridMaxEasting := network.MinEasting + float64(network.Columns)*network.CellSizeMeters
	gridMaxNorthing := network.MinNorthing + float64(network.Rows)*network.CellSizeMeters
	if manifest.Bounds.MinEasting < network.MinEasting-1e-6 || manifest.Bounds.MinNorthing < network.MinNorthing-1e-6 ||
		manifest.Bounds.MaxEasting > gridMaxEasting+1e-6 || manifest.Bounds.MaxNorthing > gridMaxNorthing+1e-6 {
		return fmt.Errorf("footpath source bounds fall outside its raster")
	}
	walkable := countWalkable(network)
	if walkable == 0 || manifest.WalkableCellCount != walkable {
		return fmt.Errorf("footpath walkable-cell count does not match its manifest")
	}
	if manifest.ShapeRecordCount < manifest.PolylineRecordCount || manifest.PolylineRecordCount <= 0 ||
		manifest.PartCount <= 0 || manifest.PointCount <= 0 || manifest.SegmentCount <= 0 || manifest.RasterizedSampleCount <= 0 {
		return fmt.Errorf("footpath geometry counts are invalid")
	}
	if manifest.ConnectedComponentCount <= 0 || manifest.LargestComponentCells <= 0 || manifest.LargestComponentCells > walkable {
		return fmt.Errorf("footpath connectivity counts are invalid")
	}
	if verifyComponents {
		components, largest, err := connectedComponentStats(context.Background(), network)
		if err != nil {
			return err
		}
		if components != manifest.ConnectedComponentCount || largest != manifest.LargestComponentCells {
			return fmt.Errorf("footpath connectivity counts do not match its manifest")
		}
	}
	return nil
}

const sha256HexLength = 64

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
	} else if _, statErr := os.Stat(path); statErr != nil && errors.Is(statErr, os.ErrNotExist) {
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
