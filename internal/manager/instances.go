package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func (m *Manager) ClusterManagers(ctx context.Context) (map[string]string, error) {
	return m.getClusterManagers(ctx)
}
func (m *Manager) getClusterManagers(ctx context.Context) (map[string]string, error) {
	resp, err := m.etcd.Get(ctx, "/managers/", clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, kv := range resp.Kvs {
		out[string(kv.Key)] = string(kv.Value)
	}
	return out, nil
}

func (m *Manager) putInstance(ctx context.Context, inst Instance) error {
	b, err := json.Marshal(inst)
	if err != nil {
		return err
	}
	_, err = m.etcd.Put(ctx, "/instances/"+inst.PixelStreamingID, string(b))
	return err
}

func (m *Manager) getInstance(ctx context.Context, id string) (*Instance, error) {
	resp, err := m.etcd.Get(ctx, "/instances/"+id)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, nil
	}
	var inst Instance
	if err := json.Unmarshal(resp.Kvs[0].Value, &inst); err != nil {
		return nil, err
	}
	return &inst, nil
}

func (m *Manager) GetAllInstances(ctx context.Context) ([]Instance, error) {
	resp, err := m.etcd.Get(ctx, "/instances/", clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var out []Instance
	for _, kv := range resp.Kvs {
		var inst Instance
		if err := json.Unmarshal(kv.Value, &inst); err == nil {
			out = append(out, inst)
		}
	}
	return out, nil
}

func (m *Manager) listLocalInstances(ctx context.Context) ([]Instance, error) {
	all, err := m.GetAllInstances(ctx)
	if err != nil {
		return nil, err
	}
	var out []Instance
	for _, inst := range all {
		if inst.Host == m.managerHost {
			out = append(out, inst)
		}
	}
	return out, nil
}

func (m *Manager) CreateInstance(ctx context.Context, req StartInstanceRequest, initialize bool) (StartInstanceResponse, int, error) {
	if req.PixelStreamingServerPort == 0 {
		req.PixelStreamingServerPort = 8888
	}
	if req.Model == "" {
		req.Model = "default"
	}
	if req.ResX == 0 {
		req.ResX = m.cfg.DefaultResX
	}
	if req.ResY == 0 {
		req.ResY = m.cfg.DefaultResY
	}

	managers, err := m.getClusterManagers(ctx)
	if err != nil {
		return StartInstanceResponse{}, http.StatusInternalServerError, err
	}

	var availableHosts []string
	for k := range managers {
		availableHosts = append(availableHosts, strings.TrimPrefix(k, "/managers/"))
	}

	if !initialize {
		streamers, err := m.signal.FetchAllStreamers(ctx)
		if err == nil {
			inst, err := m.allocateIdleInstance(ctx, streamers, req.UserID, availableHosts)
			if err == nil && inst != nil {
				return StartInstanceResponse{
					Message:                  "Existing instance found and idle (no subscribers)",
					PixelStreamingID:         inst.PixelStreamingID,
					PixelStreamingIP:         m.cfg.PixelStreamingIP,
					PixelStreamingServerPort: inst.Port,
					PID:                      inst.PID,
					Model:                    inst.Model,
					Reused:                   true,
				}, http.StatusOK, nil
			}
		}
	}

	localInstances, err := m.listLocalInstances(ctx)
	if err != nil {
		return StartInstanceResponse{}, http.StatusInternalServerError, err
	}

	if len(localInstances) >= m.cfg.MaxInstances {
		if !req.NoCheckOther {
			for key, url := range managers {
				host := strings.TrimPrefix(key, "/managers/")
				if host == m.managerHost {
					continue
				}

				payload, _ := json.Marshal(StartInstanceRequest{
					PixelStreamingServerPort: req.PixelStreamingServerPort,
					Model:                    req.Model,
					NoCheckOther:             true,
					ResX:                     req.ResX,
					ResY:                     req.ResY,
					PixelStreamingID:         req.PixelStreamingID,
					UserID:                   req.UserID,
				})

				httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url+"/instances", bytes.NewReader(payload))
				httpReq.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(httpReq)
				if err != nil {
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode < 300 {
					var out StartInstanceResponse
					if err := json.NewDecoder(resp.Body).Decode(&out); err == nil {
						out.Message = fmt.Sprintf("Instance started on remote host %s", host)
						return out, http.StatusOK, nil
					}
				}
			}
		}

		return StartInstanceResponse{}, http.StatusTooManyRequests, fmt.Errorf("maximum instances reached on this node and no remote slots available")
	}

	models, err := m.EnsureDefaultModel(ctx)
	if err != nil {
		return StartInstanceResponse{}, http.StatusInternalServerError, err
	}

	exePath := models[req.Model]
	if exePath == "" {
		exePath = models["default"]
	}
	if exePath == "" {
		return StartInstanceResponse{}, http.StatusBadRequest, fmt.Errorf(`model "%s" not found and no default configured`, req.Model)
	}

	id := req.PixelStreamingID
	if _, err := uuid.Parse(id); err != nil {
		id = uuid.NewString()
	}

	existing, err := m.getInstance(ctx, id)
	if err == nil && existing != nil {
		id = uuid.NewString()
	}

	args := buildPixelArgs(req.PixelStreamingServerPort, id, req.ResX, req.ResY)

	cmd := exec.Command(exePath, args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return StartInstanceResponse{}, http.StatusInternalServerError, err
	}

	m.mu.Lock()
	m.processes[id] = cmd
	m.mu.Unlock()

	go m.logPipe(id, stdout)
	go m.logPipe(id, stderr)
	go m.waitForExit(id, cmd)

	inst := Instance{
		PixelStreamingID: id,
		Host:             m.managerHost,
		Port:             req.PixelStreamingServerPort,
		PID:              cmd.Process.Pid,
		StartTime:        time.Now().UnixMilli(),
		Model:            req.Model,
		UserID:           req.UserID,
	}

	if err := m.putInstance(ctx, inst); err != nil {
		return StartInstanceResponse{}, http.StatusInternalServerError, err
	}

	return StartInstanceResponse{
		Message:                  "Instance started successfully",
		PixelStreamingID:         id,
		PixelStreamingIP:         m.cfg.PixelStreamingIP,
		PixelStreamingServerPort: req.PixelStreamingServerPort,
		PID:                      cmd.Process.Pid,
		Model:                    req.Model,
		Reused:                   false,
	}, http.StatusOK, nil
}

func buildPixelArgs(port int, pixelStreamingID string, resX, resY int) []string {
	return []string{
		fmt.Sprintf("-PixelStreamingPort=%d", port),
		fmt.Sprintf("-ResX=%d", resX),
		fmt.Sprintf("-ResY=%d", resY),
		"-Windowed",
		"-RenderOffScreen",
		"-ForceRes",
		fmt.Sprintf("-PixelStreamingID=%s", pixelStreamingID),
	}
}

func (m *Manager) logPipe(id string, r io.ReadCloser) {
	defer r.Close()
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			fmt.Printf("[%s] %s\n", id, strings.TrimSpace(string(buf[:n])))
		}
		if err != nil {
			return
		}
	}
}

func (m *Manager) waitForExit(id string, cmd *exec.Cmd) {
	_ = cmd.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = m.etcd.Delete(ctx, "/instances/"+id)

	m.mu.Lock()
	delete(m.processes, id)
	m.mu.Unlock()
}

func (m *Manager) allocateIdleInstance(ctx context.Context, currentStreamers []Streamer, userID string, availableManagerHosts []string) (*Instance, error) {
	streamerMap := map[string]Streamer{}
	for _, s := range currentStreamers {
		streamerMap[s.StreamerID] = s
	}

	instances, err := m.GetAllInstances(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	for _, instance := range instances {
		if !contains(availableManagerHosts, instance.Host) {
			continue
		}

		streamer := streamerMap[instance.PixelStreamingID]
		hasSubscribers := len(streamer.Subscribers) > 0

		if hasSubscribers {
			if streamer.Subscribers[0].PlayerID == userID || instance.UserID == userID {
				return &instance, nil
			}
			if !instance.Subscribed {
				instance.Subscribed = true
				instance.LastSubscribed = time.Now().Format(time.RFC3339)
				instance.UserID = userID
				_ = m.putInstance(ctx, instance)
			}
			continue
		}

		var last time.Time
		if instance.LastSubscribed != "" {
			last, _ = time.Parse(time.RFC3339, instance.LastSubscribed)
		}

		if last.IsZero() || now.Sub(last) > FiveMinutes {
			instance.Subscribed = true
			instance.LastSubscribed = time.Now().Format(time.RFC3339)
			instance.UserID = userID
			if err := m.putInstance(ctx, instance); err == nil {
				return &instance, nil
			}
		}
	}

	return nil, nil
}

func (m *Manager) StopInstance(ctx context.Context, id string) error {
	inst, err := m.getInstance(ctx, id)
	if err != nil {
		return err
	}
	if inst == nil {
		return fmt.Errorf("instance %s does not exist", id)
	}

	if err := m.killInstance(ctx, *inst); err != nil {
		return err
	}

	_, _ = m.etcd.Delete(ctx, "/instances/"+id)
	return nil
}

func (m *Manager) StopAllInstances(ctx context.Context) (map[string]any, error) {
	instances, err := m.GetAllInstances(ctx)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"message": "Stop all instances completed",
		"total":   len(instances),
		"stopped": 0,
		"failed":  0,
		"errors":  []string{},
	}

	var errs []string

	for _, inst := range instances {
		if err := m.killInstance(ctx, inst); err != nil {
			result["failed"] = result["failed"].(int) + 1
			errs = append(errs, fmt.Sprintf("%s: %v", inst.PixelStreamingID, err))
			continue
		}
		_, _ = m.etcd.Delete(ctx, "/instances/"+inst.PixelStreamingID)
		result["stopped"] = result["stopped"].(int) + 1
	}

	result["errors"] = errs
	return result, nil
}

func (m *Manager) killInstance(ctx context.Context, inst Instance) error {
	if inst.Host == m.managerHost {
		m.mu.Lock()
		cmd := m.processes[inst.PixelStreamingID]
		m.mu.Unlock()

		if cmd != nil && cmd.Process != nil {
			if runtime.GOOS == "windows" {
				return exec.Command("taskkill", "/PID", strconv.Itoa(inst.PID), "/T", "/F").Run()
			}
			return cmd.Process.Kill()
		}

		if runtime.GOOS == "windows" {
			return exec.Command("taskkill", "/PID", strconv.Itoa(inst.PID), "/T", "/F").Run()
		}
		return exec.Command("kill", "-9", strconv.Itoa(inst.PID)).Run()
	}

	managers, err := m.getClusterManagers(ctx)
	if err != nil {
		return err
	}

	var remoteURL string
	for k, v := range managers {
		if strings.TrimPrefix(k, "/managers/") == inst.Host {
			remoteURL = v
			break
		}
	}
	if remoteURL == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, remoteURL+"/instances/"+inst.PixelStreamingID, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (m *Manager) ClearInstances(ctx context.Context) error {
	all, err := m.GetAllInstances(ctx)
	if err != nil {
		return err
	}

	for _, inst := range all {
		if inst.Host != m.managerHost {
			continue
		}
		_ = m.killInstance(ctx, inst)
		_, _ = m.etcd.Delete(ctx, "/instances/"+inst.PixelStreamingID)
	}

	return nil
}

func (m *Manager) startupInstances(ctx context.Context) {
	for i := 0; i < m.cfg.StartupInstances; i++ {
		_, _, _ = m.CreateInstance(ctx, StartInstanceRequest{
			PixelStreamingServerPort: 8888,
			Model:                    "default",
			ResX:                     m.cfg.DefaultResX,
			ResY:                     m.cfg.DefaultResY,
		}, true)

		time.Sleep(2 * time.Second)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
