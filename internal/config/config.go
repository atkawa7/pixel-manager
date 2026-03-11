package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ManagerPort           int    `yaml:"manager_port"`
	ManagerSubnetPrefixes string `yaml:"manager_subnet_prefixes"`
	PixelStreamingIP      string `yaml:"pixel_streaming_ip"`
	DefaultExe            string `yaml:"pixel_exe_path"`
	DefaultD3DRenderer    string `yaml:"default_d3d_renderer"`
	MaxInstances          int    `yaml:"max_instances"`
	EtcdHost              string `yaml:"etcd_host"`
	EtcdEnableAuth        bool   `yaml:"etcd_enable_auth"`
	EtcdUser              string `yaml:"etcd_user"`
	EtcdPassword          string `yaml:"etcd_password"`
	EtcdDialTimeoutMS     int    `yaml:"etcd_dial_timeout"`
	EtcdRequestTimeout    int    `yaml:"etcd_request_timeout"`
	DefaultResX           int    `yaml:"default_res_x"`
	DefaultResY           int    `yaml:"default_res_y"`
	SignalServerURL       string `yaml:"signal_server_url"`
	StartupInstances      int    `yaml:"startup_instances"`
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

func parseBool(v string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func envBool(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	if b, ok := parseBool(raw); ok {
		return b
	}
	return def
}

func defaultConfig() Config {
	return Config{
		ManagerPort:           4000,
		ManagerSubnetPrefixes: "",
		PixelStreamingIP:      "172.20.0.4",
		DefaultExe:            `C:\pixel-manager\Windows\ToyotaHiluxConvers\Binaries\Win64\ToyotaHiluxConvers.exe`,
		DefaultD3DRenderer:    "d3d11",
		MaxInstances:          3,
		EtcdHost:              "http://172.20.0.4:2379",
		EtcdEnableAuth:        false,
		EtcdUser:              "root",
		EtcdPassword:          "yourpassword",
		EtcdDialTimeoutMS:     5000,
		EtcdRequestTimeout:    10000,
		DefaultResX:           1280,
		DefaultResY:           720,
		SignalServerURL:       "http://172.20.0.4",
		StartupInstances:      3,
	}
}

func normalizeKey(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	k = strings.ReplaceAll(k, "_", "")
	k = strings.ReplaceAll(k, "-", "")
	k = strings.ReplaceAll(k, " ", "")
	k = strings.ReplaceAll(k, ".", "")
	return k
}

func flattenYAML(prefix string, value any, out map[string]any) {
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			next := k
			if prefix != "" {
				next = prefix + "." + k
			}
			flattenYAML(next, v, out)
		}
	case map[any]any:
		for k, v := range typed {
			key := fmt.Sprint(k)
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flattenYAML(next, v, out)
		}
	default:
		out[normalizeKey(prefix)] = typed
	}
}

func yamlValue(values map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if val, ok := values[normalizeKey(key)]; ok {
			return val, true
		}
	}
	return nil, false
}

func yamlString(values map[string]any, def string, keys ...string) string {
	raw, ok := yamlValue(values, keys...)
	if !ok {
		return def
	}
	v := strings.TrimSpace(fmt.Sprint(raw))
	if v == "" {
		return def
	}
	return v
}

func yamlInt(values map[string]any, def int, keys ...string) int {
	raw, ok := yamlValue(values, keys...)
	if !ok {
		return def
	}

	v := strings.TrimSpace(fmt.Sprint(raw))
	if v == "" {
		return def
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func yamlBool(values map[string]any, def bool, keys ...string) bool {
	raw, ok := yamlValue(values, keys...)
	if !ok {
		return def
	}
	if b, ok := parseBool(fmt.Sprint(raw)); ok {
		return b
	}
	return def
}

func configPath() string {
	path := strings.TrimSpace(os.Getenv("CONFIG_FILE"))
	if path != "" {
		return path
	}
	path = strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if path != "" {
		return path
	}
	return "config.yaml"
}

func ConfigPath() string {
	return configPath()
}

func maskSecret(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return "******"
}

func normalizeD3DRendererDefault(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "auto":
		return ""
	case "d3d11", "d3d12":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "d3d11"
	}
}

func (c Config) SafeView() map[string]any {
	return map[string]any{
		"managerPort":           c.ManagerPort,
		"managerSubnetPrefixes": c.ManagerSubnetPrefixes,
		"pixelStreamingIP":      c.PixelStreamingIP,
		"defaultExe":            c.DefaultExe,
		"defaultD3DRenderer":    c.DefaultD3DRenderer,
		"maxInstances":          c.MaxInstances,
		"etcdHost":              c.EtcdHost,
		"etcdEnableAuth":        c.EtcdEnableAuth,
		"etcdUser":              c.EtcdUser,
		"etcdPassword":          maskSecret(c.EtcdPassword),
		"etcdDialTimeoutMS":     c.EtcdDialTimeoutMS,
		"etcdRequestTimeout":    c.EtcdRequestTimeout,
		"defaultResX":           c.DefaultResX,
		"defaultResY":           c.DefaultResY,
		"signalServerURL":       c.SignalServerURL,
		"startupInstances":      c.StartupInstances,
	}
}

func Load() Config {
	cfg := defaultConfig()

	path := configPath()
	if raw, err := os.ReadFile(path); err == nil {
		var parsed map[string]any
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			log.Printf("failed parsing yaml config %q: %v", path, err)
		} else {
			yamlValues := map[string]any{}
			flattenYAML("", parsed, yamlValues)

			cfg.ManagerPort = yamlInt(yamlValues, cfg.ManagerPort, "MANAGER_PORT", "manager.port", "managerPort")
			cfg.ManagerSubnetPrefixes = yamlString(yamlValues, cfg.ManagerSubnetPrefixes, "MANAGER_SUBNET_PREFIX", "MANAGER_SUBNET_PREFIXES", "manager.subnet_prefix", "manager.subnet_prefixes", "managerSubnetPrefix", "managerSubnetPrefixes")
			cfg.PixelStreamingIP = yamlString(yamlValues, cfg.PixelStreamingIP, "PIXEL_STREAMING_IP", "pixel.streaming.ip", "pixelStreaming.ip")
			cfg.DefaultExe = yamlString(yamlValues, cfg.DefaultExe, "PIXEL_EXE_PATH", "pixel.exe.path", "pixelExePath")
			cfg.MaxInstances = yamlInt(yamlValues, cfg.MaxInstances, "MAX_INSTANCES", "max.instances", "maxInstances")
			cfg.EtcdEnableAuth = yamlBool(yamlValues, cfg.EtcdEnableAuth, "ETCD_ENABLE_AUTH", "etcd.enable_auth", "etcd.enableAuth", "etcdEnableAuth")
			cfg.EtcdUser = yamlString(yamlValues, cfg.EtcdUser, "ETCD_USER", "etcd.user")
			cfg.EtcdPassword = yamlString(yamlValues, cfg.EtcdPassword, "ETCD_PASSWORD", "etcd.password")
			cfg.EtcdDialTimeoutMS = yamlInt(yamlValues, cfg.EtcdDialTimeoutMS, "ETCD_DIAL_TIMEOUT", "etcd.dial.timeout")
			cfg.EtcdRequestTimeout = yamlInt(yamlValues, cfg.EtcdRequestTimeout, "ETCD_REQUEST_TIMEOUT", "etcd.request.timeout")
			cfg.DefaultResX = yamlInt(yamlValues, cfg.DefaultResX, "DEFAULT_RES_X", "default.res.x", "defaultResX")
			cfg.DefaultResY = yamlInt(yamlValues, cfg.DefaultResY, "DEFAULT_RES_Y", "default.res.y", "defaultResY")
			cfg.DefaultD3DRenderer = yamlString(yamlValues, cfg.DefaultD3DRenderer, "DEFAULT_D3D_RENDERER", "default.d3d.renderer", "defaultD3DRenderer")
			cfg.SignalServerURL = yamlString(yamlValues, cfg.SignalServerURL, "SIGNAL_SERVER_URL", "signal.server.url", "signalServerUrl")
			cfg.StartupInstances = yamlInt(yamlValues, cfg.StartupInstances, "STARTUP_INSTANCES", "startup.instances", "startupInstances")

			etcdHost := yamlString(yamlValues, "", "etcd.host")
			etcdPort := yamlInt(yamlValues, 0, "etcd.port")
			if etcdHost != "" && etcdPort > 0 {
				cfg.EtcdHost = fmt.Sprintf("http://%s:%d", etcdHost, etcdPort)
			} else {
				explicitEtcdHost := yamlString(yamlValues, "", "ETCD_HOST", "etcd.hostUrl", "etcd.url")
				cfg.EtcdHost = explicitEtcdHost
			}
		}
	} else if !os.IsNotExist(err) {
		log.Printf("failed reading config file %q: %v", path, err)
	}

	cfg.ManagerPort = envInt("MANAGER_PORT", cfg.ManagerPort)
	cfg.ManagerSubnetPrefixes = env("MANAGER_SUBNET_PREFIXES", env("MANAGER_SUBNET_PREFIX", cfg.ManagerSubnetPrefixes))
	cfg.PixelStreamingIP = env("PIXEL_STREAMING_IP", cfg.PixelStreamingIP)
	cfg.DefaultExe = env("PIXEL_EXE_PATH", cfg.DefaultExe)
	cfg.MaxInstances = envInt("MAX_INSTANCES", cfg.MaxInstances)
	cfg.EtcdHost = env("ETCD_HOST", cfg.EtcdHost)
	cfg.EtcdEnableAuth = envBool("ETCD_ENABLE_AUTH", cfg.EtcdEnableAuth)
	cfg.EtcdUser = env("ETCD_USER", cfg.EtcdUser)
	cfg.EtcdPassword = env("ETCD_PASSWORD", cfg.EtcdPassword)
	cfg.EtcdDialTimeoutMS = envInt("ETCD_DIAL_TIMEOUT", cfg.EtcdDialTimeoutMS)
	cfg.EtcdRequestTimeout = envInt("ETCD_REQUEST_TIMEOUT", cfg.EtcdRequestTimeout)
	cfg.DefaultResX = envInt("DEFAULT_RES_X", cfg.DefaultResX)
	cfg.DefaultResY = envInt("DEFAULT_RES_Y", cfg.DefaultResY)
	cfg.DefaultD3DRenderer = env("DEFAULT_D3D_RENDERER", env("D3D_RENDERER", cfg.DefaultD3DRenderer))
	cfg.SignalServerURL = env("SIGNAL_SERVER_URL", cfg.SignalServerURL)
	cfg.StartupInstances = envInt("STARTUP_INSTANCES", cfg.StartupInstances)
	cfg.DefaultD3DRenderer = normalizeD3DRendererDefault(cfg.DefaultD3DRenderer)

	if b, err := json.Marshal(cfg.SafeView()); err == nil {
		log.Printf("effective config path=%q values=%s", path, string(b))
	}

	return cfg
}
