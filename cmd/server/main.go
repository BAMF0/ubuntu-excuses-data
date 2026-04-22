package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/BAMF0/ubuntu-excuses-data/internal/api"
	"github.com/BAMF0/ubuntu-excuses-data/internal/ingest"
	yaml "github.com/BAMF0/ubuntu-excuses-data/internal/ingest/yaml"
)

func main() {
	path := "update_excuses.yaml"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	addr := ":8080"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("close %s: %v", path, err)
		}
	}()

	raw, err := yaml.ReadExcusesYAML(f)
	if err != nil {
		log.Fatalf("decode YAML: %v", err)
	}

	excuses := ingest.ToExcuses(raw)
	fmt.Printf("Loaded %d sources (%d candidates) generated %s\n",
		len(excuses.ByName), len(excuses.Candidates),
		excuses.GeneratedDate.Format("2006-01-02 15:04:05 UTC"))

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, excuses)

	fmt.Printf("Listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
