package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ManagerPort        int
	PixelStreamingIP   string
	DefaultExe         string
	MaxInstances       int
	EtcdHost           string
	EtcdUser           string
	EtcdPassword       string
	EtcdDialTimeoutMS  int
	EtcdRequestTimeout int
	DefaultResX        int
	DefaultResY        int
	SignalServerURL    string
	StartupInstances   int
}

func env(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func envInt(key string, def int) int {
	v := env(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func Load() Config {
	return Config{
		ManagerPort:        envInt("MANAGER_PORT", 4000),
		PixelStreamingIP:   env("PIXEL_STREAMING_IP", "172.20.0.4"),
		DefaultExe:         env("PIXEL_EXE_PATH", `C:\pixel-manager\Windows\ToyotaHiluxConvers\Binaries\Win64\ToyotaHiluxConvers.exe`),
		MaxInstances:       envInt("MAX_INSTANCES", 3),
		EtcdHost:           env("ETCD_HOST", "http://172.20.0.4:2379"),
		EtcdUser:           env("ETCD_USER", "root"),
		EtcdPassword:       env("ETCD_PASSWORD", "yourpassword"),
		EtcdDialTimeoutMS:  envInt("ETCD_DIAL_TIMEOUT", 5000),
		EtcdRequestTimeout: envInt("ETCD_REQUEST_TIMEOUT", 10000),
		DefaultResX:        envInt("DEFAULT_RES_X", 1280),
		DefaultResY:        envInt("DEFAULT_RES_Y", 720),
		SignalServerURL:    env("SIGNAL_SERVER_URL", "http://172.20.0.4"),
		StartupInstances:   envInt("STARTUP_INSTANCES", 3),
	}
}
