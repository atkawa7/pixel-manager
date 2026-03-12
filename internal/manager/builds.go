package manager

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (m *Manager) RegisterBuild(fileName string, fileSize int64) (Build, error) {
	id := uuid.NewString()
	baseDir := filepath.Join(BuildsRootDir, id)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return Build{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	b := &Build{
		ID:           id,
		FileName:     fileName,
		FileSize:     fileSize,
		Status:       BuildStatusQueued,
		Message:      "Queued: build is waiting for processing.",
		CreatedAt:    now,
		UpdatedAt:    now,
		ZipPath:      filepath.Join(baseDir, "package.zip"),
		ExtractedDir: filepath.Join(baseDir, "unzipped_processes"),
		Executables:  []string{},
	}

	m.buildMu.Lock()
	m.builds[id] = b
	m.buildMu.Unlock()

	return m.copyBuild(*b), nil
}

func (m *Manager) SaveBuildZip(id string, src io.Reader) error {
	m.buildMu.RLock()
	build, ok := m.builds[id]
	m.buildMu.RUnlock()
	if !ok {
		return fmt.Errorf("build %s not found", id)
	}

	dst, err := os.Create(build.ZipPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

func (m *Manager) EnqueueBuild(id string) error {
	m.buildMu.RLock()
	_, ok := m.builds[id]
	m.buildMu.RUnlock()
	if !ok {
		return fmt.Errorf("build %s not found", id)
	}

	select {
	case m.buildQueue <- id:
		return nil
	default:
		return fmt.Errorf("build queue is full")
	}
}

func (m *Manager) GetBuild(id string) (Build, bool) {
	m.buildMu.RLock()
	defer m.buildMu.RUnlock()
	build, ok := m.builds[id]
	if !ok {
		return Build{}, false
	}
	return m.copyBuild(*build), true
}

func (m *Manager) ListBuilds() []Build {
	m.buildMu.RLock()
	defer m.buildMu.RUnlock()
	out := make([]Build, 0, len(m.builds))
	for _, build := range m.builds {
		out = append(out, m.copyBuild(*build))
	}
	return out
}

func (m *Manager) processBuildQueue() {
	for id := range m.buildQueue {
		m.updateBuildStatus(id, BuildStatusExtractingAndScanning, "Extracting and Scanning: checking package integrity and safety.")
		if err := m.extractAndScanBuild(id); err != nil {
			m.updateBuildFailure(id, err.Error())
			continue
		}
		m.updateBuildReady(id)
	}
}

func (m *Manager) extractAndScanBuild(id string) error {
	m.buildMu.RLock()
	build, ok := m.builds[id]
	m.buildMu.RUnlock()
	if !ok {
		return fmt.Errorf("build %s not found", id)
	}

	if err := os.RemoveAll(build.ExtractedDir); err != nil {
		return err
	}
	if err := os.MkdirAll(build.ExtractedDir, 0o755); err != nil {
		return err
	}

	reader, err := zip.OpenReader(build.ZipPath)
	if err != nil {
		return fmt.Errorf("invalid zip: %w", err)
	}
	defer reader.Close()

	var executables []string
	for _, file := range reader.File {
		if file.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("zip contains unsupported symlink entry: %s", file.Name)
		}

		targetPath, err := safeExtractTarget(build.ExtractedDir, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			src.Close()
			return err
		}

		_, copyErr := io.Copy(dst, src)
		_ = src.Close()
		_ = dst.Close()
		if copyErr != nil {
			return copyErr
		}

		if strings.EqualFold(filepath.Ext(targetPath), ".exe") {
			executables = append(executables, filepath.ToSlash(targetPath))
		}
	}

	if len(executables) == 0 {
		return fmt.Errorf("no .exe found in package; upload packaged Windows build output")
	}

	m.buildMu.Lock()
	if b, ok := m.builds[id]; ok {
		b.Executables = executables
		b.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	m.buildMu.Unlock()

	return nil
}

func (m *Manager) updateBuildStatus(id, status, message string) {
	m.buildMu.Lock()
	defer m.buildMu.Unlock()
	if b, ok := m.builds[id]; ok {
		b.Status = status
		b.Message = message
		b.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
}

func (m *Manager) updateBuildReady(id string) {
	m.buildMu.Lock()
	defer m.buildMu.Unlock()
	if b, ok := m.builds[id]; ok {
		b.Status = BuildStatusReady
		b.Message = "Ready: build extracted and scanned successfully."
		b.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
}

func (m *Manager) updateBuildFailure(id, reason string) {
	m.buildMu.Lock()
	defer m.buildMu.Unlock()
	if b, ok := m.builds[id]; ok {
		b.Status = BuildStatusFailed
		b.Message = "Failed: " + reason
		b.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
}

func safeExtractTarget(rootDir, entry string) (string, error) {
	cleanName := filepath.Clean(entry)
	if cleanName == "." || cleanName == "" {
		return "", fmt.Errorf("invalid zip entry")
	}

	target := filepath.Join(rootDir, cleanName)
	cleanRoot := filepath.Clean(rootDir)
	cleanTarget := filepath.Clean(target)
	prefix := cleanRoot + string(os.PathSeparator)
	if cleanTarget != cleanRoot && !strings.HasPrefix(cleanTarget, prefix) {
		return "", fmt.Errorf("zip entry escapes extraction root: %s", entry)
	}
	return cleanTarget, nil
}

func (m *Manager) copyBuild(b Build) Build {
	out := b
	out.Executables = append([]string(nil), b.Executables...)
	return out
}
