package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFootpathArtifactOnlyFallsBackWhenDataIsMissing(t *testing.T) {
	temporaryDirectory := t.TempDir()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temporaryDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })
	t.Setenv("FOOTPATH_GRAPH_PATH", "")
	t.Setenv("FOOTPATH_FEED_PATH", "")

	if _, _, err := loadFootpathArtifact(context.Background()); !errors.Is(err, errFootpathUnavailable) {
		t.Fatalf("missing data error = %v, want errFootpathUnavailable", err)
	}

	brokenPath := filepath.Join(temporaryDirectory, "broken.walk")
	if err := os.WriteFile(brokenPath, []byte("not an artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FOOTPATH_GRAPH_PATH", brokenPath)
	if _, _, err := loadFootpathArtifact(context.Background()); err == nil || errors.Is(err, errFootpathUnavailable) {
		t.Fatalf("configured corrupt artifact error = %v; must fail startup", err)
	}
}

func TestLoadEnvSeparatesApplicationAndDatabaseEnvironments(t *testing.T) {
	temporaryDirectory := t.TempDir()
	writeTestEnvironment(t, temporaryDirectory, ".env.development", `APP_ENV=development
DATABASE_ENV=production
DATABASE_URL=postgresql://local.example/local
FRONTEND_ORIGINS=http://localhost:5173
`)
	writeTestEnvironment(t, temporaryDirectory, ".env.production", `APP_ENV=production
DATABASE_URL=postgresql://neon.example/production
FRONTEND_ORIGINS=https://iso-map.vercel.app
`)

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(temporaryDirectory); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	for _, key := range []string{"APP_ENV", "GO_ENV", "DATABASE_ENV", "DATABASE_URL", "FRONTEND_ORIGINS"} {
		t.Setenv(key, "")
	}
	loadEnv()

	if got := os.Getenv("APP_ENV"); got != "development" {
		t.Fatalf("APP_ENV = %q, want development", got)
	}
	if got := os.Getenv("DATABASE_URL"); got != "postgresql://neon.example/production" {
		t.Fatalf("DATABASE_URL = %q, want the configured production database", got)
	}
	if got := os.Getenv("FRONTEND_ORIGINS"); got != "http://localhost:5173" {
		t.Fatalf("FRONTEND_ORIGINS = %q, want development origin", got)
	}
}

func TestLoadEnvPreservesExplicitDatabaseURL(t *testing.T) {
	temporaryDirectory := t.TempDir()
	writeTestEnvironment(t, temporaryDirectory, ".env.development", `APP_ENV=development
DATABASE_ENV=production
DATABASE_URL=postgresql://local.example/local
`)
	writeTestEnvironment(t, temporaryDirectory, ".env.production", "DATABASE_URL=postgresql://neon.example/production\n")

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(temporaryDirectory); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	t.Setenv("APP_ENV", "")
	t.Setenv("GO_ENV", "")
	t.Setenv("DATABASE_ENV", "")
	t.Setenv("DATABASE_URL", "postgresql://explicit.example/database")
	loadEnv()

	if got := os.Getenv("DATABASE_URL"); got != "postgresql://explicit.example/database" {
		t.Fatalf("DATABASE_URL = %q, want explicit value", got)
	}
}

func writeTestEnvironment(t *testing.T, directory, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}
