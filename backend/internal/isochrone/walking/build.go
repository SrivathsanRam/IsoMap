package walking

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const projectionName = "SVY21 / EPSG:3414"

// BuildArchive parses Footpath_Mar2026.zip directly and rasterizes its single
// PolyLine shapefile. The source archive is not extracted to disk.
func BuildArchive(ctx context.Context, path string, config BuildConfig) (*Artifact, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var err error
	config, err = normalizeBuildConfig(config)
	if err != nil {
		return nil, err
	}

	sourceSHA, err := hashFile(ctx, path)
	if err != nil {
		return nil, err
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open footpath ZIP: %w", err)
	}
	defer archive.Close()

	shapeFile, projectionFile, err := locateArchiveMembers(archive.File)
	if err != nil {
		return nil, err
	}
	if shapeFile.UncompressedSize64 > uint64(config.MaximumShapefileSize) {
		return nil, fmt.Errorf("shapefile is %d bytes, exceeding configured maximum of %d", shapeFile.UncompressedSize64, config.MaximumShapefileSize)
	}
	if err := readAndValidateProjection(ctx, projectionFile); err != nil {
		return nil, err
	}

	shapeReader, err := shapeFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open shapefile %q: %w", shapeFile.Name, err)
	}
	network, summary, err := parseAndRasterizeShapefile(ctx, shapeReader, shapeFile.UncompressedSize64, config)
	closeErr := shapeReader.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close shapefile %q: %w", shapeFile.Name, closeErr)
	}

	components, largest, err := connectedComponentStats(ctx, &network)
	if err != nil {
		return nil, err
	}
	walkable := countWalkable(&network)
	if walkable == 0 || components == 0 || largest == 0 {
		return nil, fmt.Errorf("footpath raster contains no connected walkable cells")
	}

	artifact := &Artifact{
		Manifest: Manifest{
			Format:                  FormatName,
			SchemaVersion:           SchemaVersion,
			SourceFile:              filepath.Base(path),
			SourceSHA256:            sourceSHA,
			Projection:              projectionName,
			CellSizeMeters:          network.CellSizeMeters,
			Columns:                 network.Columns,
			Rows:                    network.Rows,
			ShapeRecordCount:        summary.shapeRecords,
			PolylineRecordCount:     summary.polylineRecords,
			PartCount:               summary.parts,
			PointCount:              summary.points,
			SegmentCount:            summary.segments,
			RasterizedSampleCount:   summary.samples,
			WalkableCellCount:       walkable,
			ConnectedComponentCount: components,
			LargestComponentCells:   largest,
			Bounds:                  summary.bounds,
		},
		Network: network,
	}
	if err := validateArtifact(artifact, false); err != nil {
		return nil, fmt.Errorf("validate built footpath artifact: %w", err)
	}
	return artifact, nil
}

func normalizeBuildConfig(config BuildConfig) (BuildConfig, error) {
	defaults := DefaultBuildConfig()
	if config.CellSizeMeters == 0 {
		config.CellSizeMeters = defaults.CellSizeMeters
	}
	if config.MaximumGridCells == 0 {
		config.MaximumGridCells = defaults.MaximumGridCells
	}
	if config.MaximumShapefileSize == 0 {
		config.MaximumShapefileSize = defaults.MaximumShapefileSize
	}
	if !finite(config.CellSizeMeters) || config.CellSizeMeters <= 0 {
		return config, fmt.Errorf("footpath cell size must be a finite positive number")
	}
	if config.MaximumGridCells <= 0 {
		return config, fmt.Errorf("maximum footpath grid cells must be positive")
	}
	if config.MaximumShapefileSize < 100 {
		return config, fmt.Errorf("maximum shapefile size must be at least 100 bytes")
	}
	return config, nil
}

func locateArchiveMembers(files []*zip.File) (*zip.File, *zip.File, error) {
	var shapes, projections []*zip.File
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(strings.ReplaceAll(file.Name, "\\", "/"))) {
		case ".shp":
			shapes = append(shapes, file)
		case ".prj":
			projections = append(projections, file)
		}
	}
	if len(shapes) != 1 {
		return nil, nil, fmt.Errorf("footpath ZIP must contain exactly one .shp file (found %d)", len(shapes))
	}
	if len(projections) != 1 {
		return nil, nil, fmt.Errorf("footpath ZIP must contain exactly one .prj file (found %d)", len(projections))
	}
	return shapes[0], projections[0], nil
}

func hashFile(ctx context.Context, path string) (string, error) {
	file, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve footpath source path: %w", err)
	}
	reader, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("open footpath source for hashing: %w", err)
	}
	defer reader.Close()

	digest := sha256.New()
	buffer := make([]byte, 128*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		count, readErr := reader.Read(buffer)
		if count > 0 {
			if _, err := digest.Write(buffer[:count]); err != nil {
				return "", fmt.Errorf("hash footpath source: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("hash footpath source: %w", readErr)
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func markWalkable(network *Network, index int) {
	network.WalkableBits[index>>6] |= uint64(1) << uint(index&63)
}

func rasterizeSegment(ctx context.Context, network *Network, from, to shapePoint) (int64, error) {
	distance := math.Hypot(to.x-from.x, to.y-from.y)
	stepValue := math.Ceil(distance / (network.CellSizeMeters / 2))
	if !finite(stepValue) || stepValue > float64(maxInt()-1) {
		return 0, fmt.Errorf("footpath segment is too long to rasterize")
	}
	steps := int(stepValue)
	if steps < 1 {
		steps = 1
	}
	for step := 0; step <= steps; step++ {
		if step&4095 == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}
		fraction := float64(step) / float64(steps)
		easting := from.x + (to.x-from.x)*fraction
		northing := from.y + (to.y-from.y)*fraction
		if index, ok := network.CellAt(easting, northing); ok {
			markWalkable(network, index)
		}
	}
	return int64(steps + 1), nil
}
