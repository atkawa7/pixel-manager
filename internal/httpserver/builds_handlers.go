package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"pixel-manager/internal/manager"
	numconv "strconv"
	"strings"
)

const maxBuildZipSize int64 = 4 * 1024 * 1024 * 1024

type buildURLImportRequest struct {
	URL string `json:"url"`
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
