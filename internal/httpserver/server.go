package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path"
	"pixel-manager/internal/config"
	"pixel-manager/internal/manager"
	numconv "strconv"
	"strings"
	"time"
)

const maxBuildZipSize int64 = 4 * 1024 * 1024 * 1024

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

func (s *Server) handleBuilds(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if isJSONRequest(r) {
			s.handleBuildURLImport(w, r)
			return
		}

		s.handleBuildMultipartUpload(w, r)

	case http.MethodGet:
		builds := s.mgr.ListBuilds()
		page := 1
		pageSize := 10
		if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
			if parsed, err := numconv.Atoi(raw); err == nil && parsed > 0 {
				page = parsed
			}
		}
		if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
			if parsed, err := numconv.Atoi(raw); err == nil && parsed > 0 {
				if parsed > 100 {
					parsed = 100
				}
				pageSize = parsed
			}
		}

		total := len(builds)
		totalPages := 1
		if total == 0 {
			totalPages = 0
		} else {
			totalPages = (total + pageSize - 1) / pageSize
		}
		if totalPages > 0 && page > totalPages {
			page = totalPages
		}

		start := 0
		end := 0
		if total > 0 && totalPages > 0 {
			start = (page - 1) * pageSize
			if start < 0 {
				start = 0
			}
			if start > total {
				start = total
			}
			end = start + pageSize
			if end > total {
				end = total
			}
		}

		paged := builds
		if total == 0 {
			paged = []manager.Build{}
		} else {
			paged = builds[start:end]
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"count":      len(paged),
			"builds":     paged,
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
		})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type buildURLImportRequest struct {
	URL string `json:"url"`
}

func (s *Server) handleBuildMultipartUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBuildZipSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid multipart payload or file exceeds 4GB limit",
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "file field is required"})
		return
	}
	defer file.Close()

	if !strings.EqualFold(path.Ext(header.Filename), ".zip") {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "only Windows build packages are supported in .ZIP format",
		})
		return
	}

	build, err := s.mgr.RegisterBuild(header.Filename, header.Size)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if err := s.mgr.SaveBuildZip(build.ID, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	if err := s.mgr.EnqueueBuild(build.ID); err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": err.Error()})
		return
	}

	out, ok := s.mgr.GetBuild(build.ID)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load build status"})
		return
	}
	writeJSON(w, http.StatusAccepted, out)
}

func (s *Server) handleBuildURLImport(w http.ResponseWriter, r *http.Request) {
	var req buildURLImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	downloadURL, err := normalizeBuildImportURL(req.URL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, downloadURL, nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid source url"})
		return
	}
	httpReq.Header.Set("User-Agent", "pixel-manager/1.0")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"error": "failed to download build from url",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": fmt.Sprintf("source url returned status %d", resp.StatusCode),
		})
		return
	}

	if resp.ContentLength > maxBuildZipSize {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "maximum upload size is 4GB",
		})
		return
	}

	fileName := resolveImportedFileName(resp, downloadURL)
	if !strings.EqualFold(path.Ext(fileName), ".zip") {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "imported file must be a .zip package",
		})
		return
	}

	fileSize := resp.ContentLength
	if fileSize < 0 {
		fileSize = 0
	}

	build, err := s.mgr.RegisterBuild(fileName, fileSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	limited := &io.LimitedReader{R: resp.Body, N: maxBuildZipSize + 1}
	if err := s.mgr.SaveBuildZip(build.ID, limited); err != nil {
		s.mgr.RemoveBuild(build.ID)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if limited.N == 0 {
		s.mgr.RemoveBuild(build.ID)
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "maximum upload size is 4GB",
		})
		return
	}

	if err := s.mgr.EnqueueBuild(build.ID); err != nil {
		s.mgr.RemoveBuild(build.ID)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": err.Error()})
		return
	}

	out, ok := s.mgr.GetBuild(build.ID)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load build status"})
		return
	}
	writeJSON(w, http.StatusAccepted, out)
}

func (s *Server) handleBuildByID(w http.ResponseWriter, r *http.Request) {
	id, isExecutablesPath, ok := parseBuildPath(r.URL.Path)
	if !ok || id == "" {
		http.NotFound(w, r)
		return
	}

	build, exists := s.mgr.GetBuild(id)
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "build not found"})
		return
	}

	if isExecutablesPath {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"buildId":     build.ID,
			"status":      build.Status,
			"executables": build.Executables,
		})
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, build)
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	s.serveOpenAPIFile(w, "openapi.json")
}

func (s *Server) handleOpenAPIAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	relPath := strings.TrimPrefix(r.URL.Path, "/openapi/")
	if strings.HasPrefix(r.URL.Path, "/api/openapi/") {
		relPath = strings.TrimPrefix(r.URL.Path, "/api/openapi/")
	}
	relPath = strings.TrimSpace(relPath)
	relPath = path.Clean(relPath)
	if relPath == "." || relPath == "" || strings.HasPrefix(relPath, "..") {
		http.NotFound(w, r)
		return
	}

	s.serveOpenAPIFile(w, relPath)
}

func (s *Server) serveOpenAPIFile(w http.ResponseWriter, relPath string) {
	data, err := fs.ReadFile(embeddedOpenAPIFiles, path.Join("openapi", relPath))
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if ct := mime.TypeByExtension(path.Ext(relPath)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
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

func parseBuildPath(path string) (id string, isExecutablesPath bool, ok bool) {
	clean := strings.Trim(path, "/")
	parts := strings.Split(clean, "/")

	// /builds/{id}
	// /builds/{id}/executables
	if len(parts) >= 2 && parts[0] == "builds" {
		if len(parts) == 2 {
			return parts[1], false, true
		}
		if len(parts) == 3 && parts[2] == "executables" {
			return parts[1], true, true
		}
		return "", false, false
	}

	// /api/builds/{id}
	// /api/builds/{id}/executables
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "builds" {
		if len(parts) == 3 {
			return parts[2], false, true
		}
		if len(parts) == 4 && parts[3] == "executables" {
			return parts[2], true, true
		}
	}

	return "", false, false
}

func isJSONRequest(r *http.Request) bool {
	contentType := strings.TrimSpace(strings.ToLower(r.Header.Get("Content-Type")))
	baseType := strings.TrimSpace(strings.Split(contentType, ";")[0])
	return baseType == "application/json" || strings.HasSuffix(baseType, "+json")
}

func normalizeBuildImportURL(raw string) (string, error) {
	source := strings.TrimSpace(raw)
	if source == "" {
		return "", fmt.Errorf("url is required")
	}

	u, err := url.Parse(source)
	if err != nil || u == nil || u.Host == "" {
		return "", fmt.Errorf("invalid source url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("source url must start with http:// or https://")
	}

	host := strings.ToLower(u.Hostname())
	switch {
	case strings.Contains(host, "dropbox.com"):
		q := u.Query()
		q.Set("dl", "1")
		q.Del("raw")
		u.RawQuery = q.Encode()
	case strings.Contains(host, "drive.google.com"):
		id := googleDriveFileID(u)
		if id != "" {
			u = &url.URL{
				Scheme: "https",
				Host:   "drive.google.com",
				Path:   "/uc",
				RawQuery: url.Values{
					"export": []string{"download"},
					"id":     []string{id},
				}.Encode(),
			}
		}
	case strings.Contains(host, "onedrive.live.com"), strings.Contains(host, "1drv.ms"), strings.Contains(host, "weshare"):
		q := u.Query()
		q.Set("download", "1")
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

func googleDriveFileID(u *url.URL) string {
	trimmedPath := strings.Trim(u.Path, "/")
	if trimmedPath != "" {
		parts := strings.Split(trimmedPath, "/")
		for idx := 0; idx < len(parts)-1; idx++ {
			if parts[idx] == "d" && parts[idx+1] != "" {
				return parts[idx+1]
			}
		}
	}

	query := u.Query()
	if id := strings.TrimSpace(query.Get("id")); id != "" {
		return id
	}
	return strings.TrimSpace(query.Get("file_id"))
}

func resolveImportedFileName(resp *http.Response, sourceURL string) string {
	disposition := strings.TrimSpace(resp.Header.Get("Content-Disposition"))
	if disposition != "" {
		if _, params, err := mime.ParseMediaType(disposition); err == nil {
			if name := strings.TrimSpace(params["filename"]); name != "" {
				return path.Base(name)
			}
		}
	}

	if parsed, err := url.Parse(sourceURL); err == nil {
		if base := strings.TrimSpace(path.Base(parsed.Path)); base != "" && base != "." && base != "/" && base != "uc" {
			return base
		}
	}

	return "build.zip"
}

func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			serveFileFromFS(staticFS, w, "/index.html")
			return
		}

		if cleanPath == "index.html" {
			serveFileFromFS(staticFS, w, "/index.html")
			return
		}

		if _, err := fs.Stat(staticFS, cleanPath); err == nil {
			if strings.HasPrefix(cleanPath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		if path.Ext(cleanPath) == "" {
			serveFileFromFS(staticFS, w, "/index.html")
			return
		}

		http.NotFound(w, r)
	})
}

func serveFileFromFS(staticFS fs.FS, w http.ResponseWriter, assetPath string) {
	normalized := strings.TrimPrefix(assetPath, "/")
	data, err := fs.ReadFile(staticFS, normalized)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if normalized == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	}

	if ct := mime.TypeByExtension(path.Ext(normalized)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	_, _ = w.Write(data)
}
