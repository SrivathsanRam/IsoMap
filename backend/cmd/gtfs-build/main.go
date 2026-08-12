package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	gtfsdata "github.com/SrivathsanRam/IsoMap/internal/isochrone/gtfs"
)

func main() {
	log.SetFlags(0)
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(arguments []string, output io.Writer) error {
	flags := flag.NewFlagSet("gtfs-build", flag.ContinueOnError)
	flags.SetOutput(output)
	inputPath := flags.String("input", "../gtfs/raw/singapore-gtfs.zip", "path to the source GTFS ZIP")
	graphPath := flags.String("output", "../gtfs/generated/singapore.graph", "path for the generated graph")
	manifestPath := flags.String("manifest", "../gtfs/generated/manifest.json", "path for the generated JSON manifest")
	validateOnly := flags.Bool("validate-only", false, "parse and validate without writing graph files")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	feed, err := gtfsdata.ParseArchive(*inputPath)
	if err != nil {
		return err
	}
	if err := feed.Validate(); err != nil {
		return err
	}
	fmt.Fprintf(
		output,
		"validated %d agencies, %d stops, %d routes, %d services, %d trips, %d stop times, %d frequencies, and %d transfers\n",
		len(feed.Agencies),
		len(feed.Stops),
		len(feed.Routes),
		len(feed.Services),
		len(feed.Trips),
		len(feed.StopTimes),
		len(feed.Frequencies),
		len(feed.Transfers),
	)
	if *validateOnly {
		return nil
	}

	digest, err := graph.SHA256File(*inputPath)
	if err != nil {
		return err
	}
	artifact, err := graph.Build(feed, graph.Source{
		Filename: filepath.Base(*inputPath),
		SHA256:   digest,
	})
	if err != nil {
		return fmt.Errorf("build graph: %w", err)
	}
	if err := graph.WriteArtifact(*graphPath, artifact); err != nil {
		return fmt.Errorf("write graph artifact: %w", err)
	}
	if err := graph.WriteManifest(*manifestPath, artifact.Manifest); err != nil {
		return fmt.Errorf("write graph manifest: %w", err)
	}

	fmt.Fprintf(
		output,
		"wrote %d nodes and %d edges to %s\nmanifest: %s\nsource SHA-256: %s\n",
		artifact.Manifest.NodeCount,
		artifact.Manifest.EdgeCount,
		*graphPath,
		*manifestPath,
		artifact.Manifest.SourceSHA256,
	)
	return nil
}
