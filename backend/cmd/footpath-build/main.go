package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/walking"
)

func main() {
	log.SetFlags(0)
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(arguments []string, output io.Writer) error {
	flags := flag.NewFlagSet("footpath-build", flag.ContinueOnError)
	flags.SetOutput(output)
	inputPath := flags.String("input", "../gtfs/raw/Footpath_Mar2026.zip", "path to the source footpath ZIP")
	artifactPath := flags.String("output", "../gtfs/generated/singapore.walk", "path for the generated pedestrian raster")
	manifestPath := flags.String("manifest", "../gtfs/generated/footpath-manifest.json", "path for the generated JSON manifest")
	cellSize := flags.Float64("cell-size", walking.DefaultCellSizeMeters, "raster cell size in SVY21 metres")
	validateOnly := flags.Bool("validate-only", false, "parse and validate without writing artifact files")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	config := walking.DefaultBuildConfig()
	config.CellSizeMeters = *cellSize
	artifact, err := walking.BuildArchive(context.Background(), *inputPath, config)
	if err != nil {
		return err
	}
	manifest := artifact.Manifest
	fmt.Fprintf(
		output,
		"validated %d PolyLine records, %d parts, %d points, and %d segments\n"+
			"raster: %dx%d at %.3fm, %d walkable cells, %d connected components, largest %d (%.2f%%)\n"+
			"source SHA-256: %s\n",
		manifest.PolylineRecordCount,
		manifest.PartCount,
		manifest.PointCount,
		manifest.SegmentCount,
		manifest.Columns,
		manifest.Rows,
		manifest.CellSizeMeters,
		manifest.WalkableCellCount,
		manifest.ConnectedComponentCount,
		manifest.LargestComponentCells,
		100*float64(manifest.LargestComponentCells)/float64(manifest.WalkableCellCount),
		manifest.SourceSHA256,
	)
	if *validateOnly {
		return nil
	}
	if err := walking.WriteArtifact(*artifactPath, artifact); err != nil {
		return fmt.Errorf("write footpath artifact: %w", err)
	}
	if err := walking.WriteManifest(*manifestPath, manifest); err != nil {
		return fmt.Errorf("write footpath manifest: %w", err)
	}
	fmt.Fprintf(output, "artifact: %s\nmanifest: %s\n", *artifactPath, *manifestPath)
	return nil
}
