package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"pixel-manager/internal/config"
	"pixel-manager/internal/httpserver"
	"pixel-manager/internal/manager"
	"pixel-manager/internal/signal"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cfg := config.Load()

	etcd, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.EtcdHost},
		DialTimeout: time.Duration(cfg.EtcdDialTimeoutMS) * time.Millisecond,
		Username:    cfg.EtcdUser,
		Password:    cfg.EtcdPassword,
	})
	if err != nil {
		log.Fatalf("failed to connect to etcd: %v", err)
	}
	defer etcd.Close()

	signalClient := &signal.HTTPClient{
		BaseURL: cfg.SignalServerURL,
	}

	mgr := manager.New(cfg, etcd, signalClient)

	ctx := context.Background()

	if err := mgr.Init(ctx); err != nil {
		log.Fatalf("failed to initialize manager: %v", err)
	}

	srv := httpserver.New(cfg, mgr)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	}()

	log.Printf("Distributed Pixel Manager API running on http://%s:%d", mgr.ManagerHost(), cfg.ManagerPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	if err := mgr.ClearInstances(context.Background()); err != nil {
		log.Printf("clear instances error: %v", err)
	}
}
