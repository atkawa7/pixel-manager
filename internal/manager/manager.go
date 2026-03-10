package manager

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"pixel-manager/internal/config"
	"pixel-manager/internal/signal"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ModelsKey   = "/config/models"
	FiveMinutes = 5 * time.Minute
)

type Manager struct {
	cfg         config.Config
	etcd        *clientv3.Client
	signal      signal.Client
	managerHost string
	leaseID     clientv3.LeaseID

	mu        sync.Mutex
	processes map[string]*exec.Cmd
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
