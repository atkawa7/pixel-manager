package manager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"pixel-manager/internal/config"
	"pixel-manager/internal/signal"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ModelsKey      = "/config/models"
	FiveMinutes    = 5 * time.Minute
	MaxLogLinesPer = 1000
	LogsRootDir    = "logs"
)

type Manager struct {
	cfg         config.Config
	etcd        *clientv3.Client
	signal      signal.Client
	managerName string
	managerHost string
	leaseID     clientv3.LeaseID

	mu        sync.Mutex
	processes map[string]*exec.Cmd

	logMu sync.RWMutex
}

func New(cfg config.Config, etcd *clientv3.Client, signalClient signal.Client) *Manager {
	managerHost := detectManagerHost(cfg.ManagerSubnetPrefixes)
	managerName := detectManagerName()
	if managerName == "" {
		managerName = managerHost
	}

	return &Manager{
		cfg:         cfg,
		etcd:        etcd,
		signal:      signalClient,
		managerName: managerName,
		managerHost: managerHost,
		processes:   map[string]*exec.Cmd{},
	}
}

type ManagerRegistration struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ClusterManagerInfo struct {
	Key  string `json:"key"`
	Host string `json:"host"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (m *Manager) appendLogLine(id, stream, line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	formatted := fmt.Sprintf("[%s] %s", stream, line)

	m.logMu.Lock()
	defer m.logMu.Unlock()

	logPath := m.instanceLogPath(id)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		log.Printf("failed creating log directory for %s: %v", id, err)
		return
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("failed opening log file for %s: %v", id, err)
		return
	}
	defer f.Close()

	_, _ = f.WriteString(formatted + "\n")
}

func (m *Manager) InstanceLogs(id string, tail int) []string {
	m.logMu.RLock()
	defer m.logMu.RUnlock()

	if tail <= 0 {
		tail = 200
	}
	if tail > MaxLogLinesPer {
		tail = MaxLogLinesPer
	}

	logPath := m.instanceLogPath(id)
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}
		}
		log.Printf("failed reading log file for %s: %v", id, err)
		return []string{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := make([]string, 0, tail)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > tail {
			lines = lines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("failed scanning log file for %s: %v", id, err)
	}

	out := make([]string, len(lines))
	copy(out, lines)
	return out
}

func (m *Manager) logPipe(id, stream string, r io.ReadCloser) {
	defer r.Close()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m.appendLogLine(id, stream, line)
		fmt.Printf("[%s][%s] %s\n", id, stream, line)
	}
}

func (m *Manager) instanceLogPath(id string) string {
	return filepath.Join(LogsRootDir, id, "instance.log")
}

func (m *Manager) ManagerHost() string {
	return m.managerHost
}

func (m *Manager) ManagerName() string {
	return m.managerName
}

func (m *Manager) Init(ctx context.Context) error {
	if err := m.registerManager(ctx); err != nil {
		return err
	}

	go m.watchManagers(context.Background())

	if err := m.ClearInstances(context.Background()); err != nil {
		return err
	}

	go m.startupInstances(context.Background())

	return nil
}

func detectManagerHost(subnetPrefixesCSV string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		h, _ := os.Hostname()
		return h
	}

	preferredPrefixes := parseSubnetPrefixes(subnetPrefixesCSV)
	var firstNonLoopbackIPv4 string

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch a := addr.(type) {
			case *net.IPNet:
				ip = a.IP
			case *net.IPAddr:
				ip = a.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}

			ipStr := ip.String()
			if firstNonLoopbackIPv4 == "" {
				firstNonLoopbackIPv4 = ipStr
			}
			if hasAnyPrefix(ipStr, preferredPrefixes) {
				return ipStr
			}
			if isPrivateIPv4(ip) {
				return ipStr
			}
		}
	}

	if firstNonLoopbackIPv4 != "" {
		return firstNonLoopbackIPv4
	}

	h, _ := os.Hostname()
	return h
}

func parseSubnetPrefixes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func detectManagerName() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

func parseManagerRegistration(value []byte, host string) ManagerRegistration {
	raw := strings.TrimSpace(string(value))
	if raw == "" {
		return ManagerRegistration{Name: host, URL: ""}
	}

	if strings.HasPrefix(raw, "{") {
		var parsed ManagerRegistration
		if err := json.Unmarshal(value, &parsed); err == nil {
			if strings.TrimSpace(parsed.Name) == "" {
				parsed.Name = host
			}
			return parsed
		}
	}

	return ManagerRegistration{
		Name: host,
		URL:  raw,
	}
}

func hasAnyPrefix(v string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(v, prefix) {
			return true
		}
	}
	return false
}

func isPrivateIPv4(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}

	if v4[0] == 10 {
		return true
	}
	if v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31 {
		return true
	}
	if v4[0] == 192 && v4[1] == 168 {
		return true
	}

	return false
}

func (m *Manager) registerManager(ctx context.Context) error {
	leaseResp, err := m.etcd.Grant(ctx, 10)
	if err != nil {
		return err
	}
	m.leaseID = leaseResp.ID

	key := "/managers/" + m.managerHost
	registration := ManagerRegistration{
		Name: m.managerName,
		URL:  fmt.Sprintf("http://%s:%d", m.managerHost, m.cfg.ManagerPort),
	}
	payload, err := json.Marshal(registration)
	if err != nil {
		return err
	}

	if _, err := m.etcd.Put(ctx, key, string(payload), clientv3.WithLease(m.leaseID)); err != nil {
		return err
	}

	ch, err := m.etcd.KeepAlive(context.Background(), m.leaseID)
	if err != nil {
		return err
	}

	go func() {
		for ka := range ch {
			if ka == nil {
				log.Println("manager lease keepalive lost")
				return
			}
		}
	}()

	return nil
}

func (m *Manager) watchManagers(ctx context.Context) {
	wch := m.etcd.Watch(ctx, "/managers/", clientv3.WithPrefix())
	for wr := range wch {
		for _, ev := range wr.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				log.Printf("manager added key=%s value=%s", string(ev.Kv.Key), string(ev.Kv.Value))
			case clientv3.EventTypeDelete:
				log.Printf("manager removed key=%s", string(ev.Kv.Key))
			}
		}
	}
}
