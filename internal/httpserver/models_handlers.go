package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"pixel-manager/internal/manager"
)

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
