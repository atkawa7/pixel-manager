package httpserver

import "net/http"

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
