package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/SrivathsanRam/IsoMap/internal/database"
	isochroneHandler "github.com/SrivathsanRam/IsoMap/internal/handlers/isochrone"
	isochroneEngine "github.com/SrivathsanRam/IsoMap/internal/isochrone"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/contour"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/footpath"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	gtfsdata "github.com/SrivathsanRam/IsoMap/internal/isochrone/gtfs"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/walking"
	"github.com/SrivathsanRam/IsoMap/internal/router"
	"github.com/SrivathsanRam/IsoMap/internal/routes"
)

var errFootpathUnavailable = errors.New("footpath data is unavailable")

func main() {
	loadEnv()

	log.Println("Connecting to database...")
	db, err := database.GetDB()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Database connection ready; running migrations...")

	if err := db.Migrate(); err != nil {
		log.Fatalln(err)
	}
	log.Println("Database migrations complete; loading isochrone graph...")

	artifact, source, err := loadIsochroneArtifact()
	if err != nil {
		log.Fatalln(err)
	}
	isochroneConfig := isochroneEngine.DefaultConfig()
	isochrone, walkingSource, err := buildIsochroneEngine(artifact, isochroneConfig)
	if err != nil {
		log.Fatalln(err)
	}
	isochroneHTTP, err := isochroneHandler.New(isochrone, artifact.Manifest)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf(
		"Isochrone graph ready from %s: %d stops, %d nodes, %d edges",
		source,
		artifact.Manifest.StopCount,
		artifact.Manifest.NodeCount,
		artifact.Manifest.EdgeCount,
	)
	log.Printf("Isochrone walking model ready from %s", walkingSource)

	r := router.Setup(routes.Dependencies{Isochrone: isochroneHTTP})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	fmt.Printf("Listening on port %s\n", port)
	log.Fatalln(http.ListenAndServe(":"+port, r))
}

func buildIsochroneEngine(
	transitArtifact *graph.Artifact,
	config isochroneEngine.Config,
) (*isochroneEngine.Engine, string, error) {
	walkingArtifact, source, err := loadFootpathArtifact(context.Background())
	if err != nil {
		if errors.Is(err, errFootpathUnavailable) {
			log.Printf("Footpath network unavailable (%v); using radial walking fallback", err)
			engine, fallbackErr := isochroneEngine.New(transitArtifact, config)
			return engine, "radial fallback", fallbackErr
		}
		return nil, "", err
	}
	runtimeConfig := footpath.DefaultConfig(config.Walking)
	runtime, err := footpath.NewRuntime(walkingArtifact, transitArtifact.Graph.Stops, config.Access, runtimeConfig)
	if err != nil {
		return nil, "", fmt.Errorf("initialize footpath routing: %w", err)
	}
	engine, err := isochroneEngine.NewWithWalkingProviders(
		transitArtifact,
		runtime,
		runtime,
		contour.DefaultConfig(runtimeConfig.SurfaceCellSizeMeters),
		footpath.ModelName,
	)
	if err != nil {
		return nil, "", err
	}
	manifest := walkingArtifact.Manifest
	return engine, fmt.Sprintf(
		"%s (%d walkable cells, %d components; largest %d cells)",
		source,
		manifest.WalkableCellCount,
		manifest.ConnectedComponentCount,
		manifest.LargestComponentCells,
	), nil
}

func loadFootpathArtifact(ctx context.Context) (*walking.Artifact, string, error) {
	if configured := strings.TrimSpace(os.Getenv("FOOTPATH_GRAPH_PATH")); configured != "" {
		artifact, err := walking.LoadArtifact(configured)
		if err != nil {
			return nil, "", fmt.Errorf("load footpath graph from FOOTPATH_GRAPH_PATH: %w", err)
		}
		return artifact, configured, nil
	}

	graphCandidates := []string{
		filepath.Clean("../gtfs/generated/singapore.walk"),
		filepath.Clean("gtfs/generated/singapore.walk"),
	}
	for _, candidate := range graphCandidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		artifact, err := walking.LoadArtifact(candidate)
		if err != nil {
			return nil, "", fmt.Errorf("load generated footpath graph %s: %w", candidate, err)
		}
		return artifact, candidate, nil
	}

	sourcePath := strings.TrimSpace(os.Getenv("FOOTPATH_FEED_PATH"))
	if sourcePath == "" {
		sourceCandidates := []string{
			filepath.Clean("../gtfs/raw/Footpath_Mar2026.zip"),
			filepath.Clean("gtfs/raw/Footpath_Mar2026.zip"),
		}
		for _, candidate := range sourceCandidates {
			if _, err := os.Stat(candidate); err == nil {
				sourcePath = candidate
				break
			}
		}
	}
	if sourcePath == "" {
		return nil, "", fmt.Errorf("%w: no generated footpath graph or source archive was found", errFootpathUnavailable)
	}
	artifact, err := walking.BuildArchive(ctx, sourcePath, walking.DefaultBuildConfig())
	if err != nil {
		return nil, "", fmt.Errorf("build in-memory footpath graph: %w", err)
	}
	return artifact, sourcePath + " (built in memory)", nil
}

func loadIsochroneArtifact() (*graph.Artifact, string, error) {
	if configured := strings.TrimSpace(os.Getenv("GTFS_GRAPH_PATH")); configured != "" {
		artifact, err := graph.LoadArtifact(configured)
		if err != nil {
			return nil, "", fmt.Errorf("load GTFS graph from GTFS_GRAPH_PATH: %w", err)
		}
		return artifact, configured, nil
	}

	graphCandidates := []string{
		filepath.Clean("../gtfs/generated/singapore.graph"),
		filepath.Clean("gtfs/generated/singapore.graph"),
	}
	for _, candidate := range graphCandidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		artifact, err := graph.LoadArtifact(candidate)
		if err != nil {
			return nil, "", fmt.Errorf("load generated GTFS graph %s: %w", candidate, err)
		}
		return artifact, candidate, nil
	}

	feedPath := strings.TrimSpace(os.Getenv("GTFS_FEED_PATH"))
	if feedPath == "" {
		feedCandidates := []string{
			filepath.Clean("../gtfs/raw/singapore-gtfs.zip"),
			filepath.Clean("gtfs/raw/singapore-gtfs.zip"),
		}
		for _, candidate := range feedCandidates {
			if _, err := os.Stat(candidate); err == nil {
				feedPath = candidate
				break
			}
		}
	}
	if feedPath == "" {
		return nil, "", fmt.Errorf("GTFS graph is missing and no source feed was found; set GTFS_GRAPH_PATH or GTFS_FEED_PATH")
	}

	feed, err := gtfsdata.ParseArchive(feedPath)
	if err != nil {
		return nil, "", fmt.Errorf("parse fallback GTFS feed: %w", err)
	}
	if err := feed.Validate(); err != nil {
		return nil, "", err
	}
	digest, err := graph.SHA256File(feedPath)
	if err != nil {
		return nil, "", err
	}
	artifact, err := graph.Build(feed, graph.Source{
		Filename: filepath.Base(feedPath),
		SHA256:   digest,
	})
	if err != nil {
		return nil, "", fmt.Errorf("build in-memory GTFS graph: %w", err)
	}
	return artifact, feedPath + " (built in memory)", nil
}

func loadEnv() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "development"
	}
	databaseURLWasConfigured := strings.TrimSpace(os.Getenv("DATABASE_URL")) != ""
	loadEnvironment(env)

	// Database selection is independent of the application environment. This
	// lets local development use Neon without enabling production-only CORS and
	// cookie behavior. An explicitly supplied DATABASE_URL always wins.
	databaseEnv := strings.TrimSpace(os.Getenv("DATABASE_ENV"))
	if databaseURLWasConfigured || databaseEnv == "" || databaseEnv == env {
		return
	}
	if databaseURL, ok := environmentValue(databaseEnv, "DATABASE_URL"); ok {
		_ = os.Setenv("DATABASE_URL", databaseURL)
	}
}

func loadEnvironment(env string) {
	for _, path := range environmentPaths(env) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, ok := parseEnvironmentLine(line)
			if !ok || os.Getenv(key) != "" {
				continue
			}
			_ = os.Setenv(key, value)
		}
		return
	}
}

func environmentValue(env, wantedKey string) (string, bool) {
	for _, path := range environmentPaths(env) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, ok := parseEnvironmentLine(line)
			if ok && key == wantedKey {
				return value, true
			}
		}
		return "", false
	}
	return "", false
}

func environmentPaths(env string) []string {
	return []string{
		".env." + env,
		"backend/.env." + env,
		".env",
		"backend/.env",
	}
}

func parseEnvironmentLine(line string) (string, string, bool) {
	key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
	if !ok || key == "" || strings.HasPrefix(key, "#") {
		return "", "", false
	}
	return key, strings.Trim(strings.TrimSpace(value), `"'`), true
}
