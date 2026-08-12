package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/SrivathsanRam/IsoMap/internal/database"
	isochroneHandler "github.com/SrivathsanRam/IsoMap/internal/handlers/isochrone"
	isochroneEngine "github.com/SrivathsanRam/IsoMap/internal/isochrone"
	"github.com/SrivathsanRam/IsoMap/internal/isochrone/graph"
	gtfsdata "github.com/SrivathsanRam/IsoMap/internal/isochrone/gtfs"
	"github.com/SrivathsanRam/IsoMap/internal/router"
	"github.com/SrivathsanRam/IsoMap/internal/routes"
)

func main() {
	loadEnv()

	db, err := database.GetDB()
	if err != nil {
		log.Fatalln(err)
	}

	if err := db.Migrate(); err != nil {
		log.Fatalln(err)
	}

	artifact, source, err := loadIsochroneArtifact()
	if err != nil {
		log.Fatalln(err)
	}
	isochrone, err := isochroneEngine.New(artifact, isochroneEngine.DefaultConfig())
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

	r := router.Setup(routes.Dependencies{Isochrone: isochroneHTTP})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	fmt.Printf("Listening on port %s\n", port)
	log.Fatalln(http.ListenAndServe(":"+port, r))
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

	paths := []string{
		".env." + env,
		"backend/.env." + env,
		".env",
		"backend/.env",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok || key == "" || strings.HasPrefix(key, "#") {
				continue
			}
			if os.Getenv(key) == "" {
				os.Setenv(key, strings.Trim(value, `"'`))
			}
		}
		return
	}
}
