package httpserver

import (
	"net/http"

	"pixel-manager/internal/config"
)

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
