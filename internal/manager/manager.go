package manager

import (
	"bufio"
	"context"
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
	managerHost string
	leaseID     clientv3.LeaseID

	mu        sync.Mutex
	processes map[string]*exec.Cmd

	logMu sync.RWMutex
}

func New(cfg config.Config, etcd *clientv3.Client, signalClient signal.Client) *Manager {
	return &Manager{
		cfg:         cfg,
		etcd:        etcd,
		signal:      signalClient,
		managerHost: detectManagerHost(),
		processes:   map[string]*exec.Cmd{},
	}
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

func detectManagerHost() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		h, _ := os.Hostname()
		return h
	}

	for _, iface := range ifaces {
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
			if strings.HasPrefix(ip.String(), "172.20.") {
				return ip.String()
			}
		}
	}

	h, _ := os.Hostname()
	return h
}

func (m *Manager) registerManager(ctx context.Context) error {
	leaseResp, err := m.etcd.Grant(ctx, 10)
	if err != nil {
		return err
	}
	m.leaseID = leaseResp.ID

	key := "/managers/" + m.managerHost
	val := fmt.Sprintf("http://%s:%d", m.managerHost, m.cfg.ManagerPort)

	if _, err := m.etcd.Put(ctx, key, val, clientv3.WithLease(m.leaseID)); err != nil {
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
