package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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

	srv := &http.Server{
		Addr:         addr,
		Handler:      gzipMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("Listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}

// gzipResponseWriter wraps http.ResponseWriter to compress the response body.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	headerWritten bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.headerWritten = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.Header().Del("Content-Length")
		w.headerWritten = true
	}
	return w.Writer.Write(b)
}

// gzipMiddleware transparently compresses responses for clients that support it.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		defer func() {
			if err := gz.Close(); err != nil {
				log.Printf("gzip close: %v", err)
			}
		}()
		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(&gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	})
}
