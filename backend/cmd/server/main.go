package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/CVWO/sample-go-app/internal/router"
)

func main() {
	loadEnv()
	r := router.Setup()
	fmt.Print("Listening on port 8000 at http://localhost:8000!")

	log.Fatalln(http.ListenAndServe(":8000", r))
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
