// LaLune Panel backend entrypoint.
//
// Serves the REST API under /api/ and the static frontend under /.
// All protocol binaries and client configs live under LALUNE_DATA_DIR.
package main

import (
	"log"
	"net/http"
	"os"

	"lalune-panel/internal/api"
	"lalune-panel/internal/store"
)

func main() {
	dataDir := envOr("LALUNE_DATA_DIR", "/data")
	frontendDir := envOr("LALUNE_FRONTEND_DIR", "./frontend")
	protocolsDir := envOr("LALUNE_PROTOCOLS_DIR", "./protocols")
	listen := envOr("LALUNE_LISTEN", ":6333")

	if err := os.MkdirAll(dataDir+"/bin", 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(dataDir+"/clients", 0o755); err != nil {
		log.Fatalf("create clients dir: %v", err)
	}

	st, err := store.New(dataDir + "/clients")
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	mux := http.NewServeMux()

	// API
	apiSrv := api.New(st, dataDir, protocolsDir)
	apiSrv.Register(mux)

	// Frontend (SPA fallback to index.html)
	fs := http.FileServer(http.Dir(frontendDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for non-file paths (SPA routing)
		path := frontendDir + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, frontendDir+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	log.Printf("LaLune Panel listening on %s", listen)
	log.Printf("data=%s frontend=%s protocols=%s", dataDir, frontendDir, protocolsDir)

	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
