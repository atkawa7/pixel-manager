package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"pixel-manager/internal/config"
	"pixel-manager/internal/manager"
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

	mux.HandleFunc("/instances", s.handleInstances)
	mux.HandleFunc("/instances/", s.handleInstanceByID)
	mux.HandleFunc("/models", s.handleModels)
	mux.HandleFunc("/models/", s.handleModelByName)
	mux.HandleFunc("/managers", s.handleManagers)
	mux.HandleFunc("/openapi.json", s.handleOpenAPI)

	publicDir := filepath.Join(".", "public")

	fileServer := http.FileServer(http.Dir(publicDir))

	if _, err := os.Stat(publicDir); err == nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Redirect(w, r, "/portal.html", http.StatusFound)
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	s.server = &http.Server{
		Addr:    ":" + strconv(cfg.ManagerPort),
		Handler: s.withMiddleware(mux),
	}

	return s
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
	id := strings.TrimPrefix(r.URL.Path, "/instances/")
	if id == "" {
		http.NotFound(w, r)
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
	name := strings.TrimPrefix(r.URL.Path, "/models/")
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
	for key, url := range managers {
		formatted = append(formatted, map[string]string{
			"host": strings.TrimPrefix(key, "/managers/"),
			"url":  url,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":    len(formatted),
		"managers": formatted,
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
