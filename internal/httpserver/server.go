package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"pixel-manager/internal/config"
	"pixel-manager/internal/manager"
	numconv "strconv"
	"strings"
	"time"
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
	mux.HandleFunc(prefix+"/openapi.json", s.handleOpenAPI)
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

func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req manager.StartInstanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}

		resp, status, err := s.mgr.CreateInstance(r.Context(), req, false)
		if err != nil {
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)

	case http.MethodGet:
		instances, err := s.mgr.GetAllInstances(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		active := make([]map[string]any, 0, len(instances))
		for _, inst := range instances {
			active = append(active, map[string]any{
				"pixelStreamingId": inst.PixelStreamingID,
				"host":             inst.Host,
				"port":             inst.Port,
				"pid":              inst.PID,
				"model":            inst.Model,
				"userId":           inst.UserID,
				"subscribed":       inst.Subscribed,
				"lastSubscribed":   inst.LastSubscribed,
				"startTime":        time.UnixMilli(inst.StartTime).Format(time.RFC3339),
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"count":        len(active),
			"maxInstances": s.cfg.MaxInstances,
			"active":       active,
		})

	case http.MethodDelete:
		result, err := s.mgr.StopAllInstances(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleInstanceByID(w http.ResponseWriter, r *http.Request) {
	id, isLogsPath, ok := parseInstancePath(r.URL.Path)
	if !ok || id == "" {
		http.NotFound(w, r)
		return
	}

	if isLogsPath {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tail := 200
		if tailQuery := strings.TrimSpace(r.URL.Query().Get("tail")); tailQuery != "" {
			if parsed, err := numconv.Atoi(tailQuery); err == nil && parsed > 0 {
				if parsed > 1000 {
					parsed = 1000
				}
				tail = parsed
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"instanceId": id,
			"tail":       tail,
			"lines":      s.mgr.InstanceLogs(id, tail),
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		inst, err := s.mgr.GetAllInstances(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		for _, item := range inst {
			if item.PixelStreamingID == id {
				writeJSON(w, http.StatusOK, map[string]any{
					"exists":           true,
					"pixelStreamingId": item.PixelStreamingID,
					"host":             item.Host,
					"port":             item.Port,
					"pid":              item.PID,
					"model":            item.Model,
					"userId":           item.UserID,
					"subscribed":       item.Subscribed,
					"lastSubscribed":   item.LastSubscribed,
					"startTime":        time.UnixMilli(item.StartTime).Format(time.RFC3339),
				})
				return
			}
		}

		writeJSON(w, http.StatusNotFound, map[string]any{
			"exists":  false,
			"message": "instance not found",
		})

	case http.MethodDelete:
		if err := s.mgr.StopInstance(r.Context(), id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": "stopped instance " + id})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		models, err := s.mgr.GetModels(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"models": models})

	case http.MethodPost:
		var req manager.ModelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if req.Name == "" || req.ExePath == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name and exePath are required"})
			return
		}

		models, err := s.mgr.SetModel(r.Context(), req.Name, req.ExePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "model saved",
			"models":  models,
		})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleModelByName(w http.ResponseWriter, r *http.Request) {
	name := suffixAfter(r.URL.Path, "/models/")
	if name == "" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := s.mgr.DeleteModel(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "model deleted",
		"models":  models,
	})
}

func (s *Server) handleManagers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	managers, err := s.mgr.ClusterManagers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	formatted := make([]map[string]string, 0, len(managers))
	for _, item := range managers {
		formatted = append(formatted, map[string]string{
			"host": item.Host,
			"name": item.Name,
			"url":  item.URL,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":    len(formatted),
		"managers": formatted,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"configPath": config.ConfigPath(),
		"config":     s.cfg.SafeView(),
	})
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Pixel Manager API",
			"version": "1.0.0",
		},
	})
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

func suffixAfter(path, token string) string {
	idx := strings.Index(path, token)
	if idx < 0 {
		return ""
	}
	value := strings.TrimPrefix(path[idx:], token)
	if strings.Contains(value, "/") {
		return ""
	}
	return value
}

func parseInstancePath(path string) (id string, isLogsPath bool, ok bool) {
	clean := strings.Trim(path, "/")
	parts := strings.Split(clean, "/")

	// /instances/{id}
	// /instances/{id}/logs
	if len(parts) >= 2 && parts[0] == "instances" {
		if len(parts) == 2 {
			return parts[1], false, true
		}
		if len(parts) == 3 && parts[2] == "logs" {
			return parts[1], true, true
		}
		return "", false, false
	}

	// /api/instances/{id}
	// /api/instances/{id}/logs
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "instances" {
		if len(parts) == 3 {
			return parts[2], false, true
		}
		if len(parts) == 4 && parts[3] == "logs" {
			return parts[2], true, true
		}
	}

	return "", false, false
}

func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			serveFileFromFS(fileServer, r, w, "/index.html")
			return
		}

		if _, err := fs.Stat(staticFS, cleanPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		if filepath.Ext(cleanPath) == "" {
			serveFileFromFS(fileServer, r, w, "/index.html")
			return
		}

		http.NotFound(w, r)
	})
}

func serveFileFromFS(fileServer http.Handler, r *http.Request, w http.ResponseWriter, path string) {
	cloned := r.Clone(r.Context())
	cloned.URL = new(url.URL)
	*cloned.URL = *r.URL
	cloned.URL.Path = path
	fileServer.ServeHTTP(w, cloned)
}
