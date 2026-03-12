package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"

	"pixel-manager/internal/config"
	"pixel-manager/internal/manager"
)

type Server struct {
	cfg    config.Config
	mgr    *manager.Manager
	server *http.Server
}

func New(cfg config.Config, mgr *manager.Manager) *Server {
	s := &Server{
		cfg: cfg,
		mgr: mgr,
	}

	mux := http.NewServeMux()

	s.registerAPIRoutes(mux, "")
	s.registerAPIRoutes(mux, "/api")

	mux.HandleFunc("/portal.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/portal", http.StatusFound)
	})
	mux.HandleFunc("/managers.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/managers", http.StatusFound)
	})
	mux.HandleFunc("/models.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/models", http.StatusFound)
	})

	staticFS, err := fs.Sub(embeddedPublicFiles, "public")
	if err == nil {
		mux.Handle("/", spaHandler(staticFS))
	}

	s.server = &http.Server{
		Addr:    ":" + strconv(cfg.ManagerPort),
		Handler: s.withMiddleware(mux),
	}

	return s
}

func (s *Server) registerAPIRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/instances", s.handleInstances)
	mux.HandleFunc(prefix+"/instances/", s.handleInstanceByID)
	mux.HandleFunc(prefix+"/models", s.handleModels)
	mux.HandleFunc(prefix+"/models/", s.handleModelByName)
	mux.HandleFunc(prefix+"/managers", s.handleManagers)
	mux.HandleFunc(prefix+"/config", s.handleConfig)
	mux.HandleFunc(prefix+"/builds", s.handleBuilds)
	mux.HandleFunc(prefix+"/builds/", s.handleBuildByID)
	mux.HandleFunc(prefix+"/openapi.json", s.handleOpenAPI)
	mux.HandleFunc(prefix+"/openapi/", s.handleOpenAPIAsset)
}

func (s *Server) Start() error {
	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func strconv(v int) string {
	return fmt.Sprintf("%d", v)
}
