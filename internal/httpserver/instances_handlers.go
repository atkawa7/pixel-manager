package httpserver

import (
	"encoding/json"
	"net/http"
	numconv "strconv"
	"strings"
	"time"

	"pixel-manager/internal/manager"
)

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
				"pixelStreamingId":         inst.PixelStreamingID,
				"pixelStreamingIp":         inst.PixelStreamingIP,
				"host":                     inst.Host,
				"port":                     inst.Port,
				"pixelStreamingServerPort": inst.PixelStreamingServerPort,
				"pid":                      inst.PID,
				"model":                    inst.Model,
				"executablePath":           inst.ExecutablePath,
				"args":                     inst.Args,
				"userId":                   inst.UserID,
				"subscribed":               inst.Subscribed,
				"lastSubscribed":           inst.LastSubscribed,
				"startTime":                time.UnixMilli(inst.StartTime).Format(time.RFC3339),
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
					"exists":                   true,
					"pixelStreamingId":         item.PixelStreamingID,
					"pixelStreamingIp":         item.PixelStreamingIP,
					"host":                     item.Host,
					"port":                     item.Port,
					"pixelStreamingServerPort": item.PixelStreamingServerPort,
					"pid":                      item.PID,
					"model":                    item.Model,
					"executablePath":           item.ExecutablePath,
					"args":                     item.Args,
					"userId":                   item.UserID,
					"subscribed":               item.Subscribed,
					"lastSubscribed":           item.LastSubscribed,
					"startTime":                time.UnixMilli(item.StartTime).Format(time.RFC3339),
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
