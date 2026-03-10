package manager

import (
	"context"
	"encoding/json"
	"fmt"
)

func (m *Manager) EnsureDefaultModel(ctx context.Context) (map[string]string, error) {
	models, err := m.GetModels(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := models["default"]; !ok {
		models["default"] = m.cfg.DefaultExe
		if err := m.saveModels(ctx, models); err != nil {
			return nil, err
		}
	}
	return models, nil
}

func (m *Manager) GetModels(ctx context.Context) (map[string]string, error) {
	resp, err := m.etcd.Get(ctx, ModelsKey)
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return map[string]string{"default": m.cfg.DefaultExe}, nil
	}

	var models map[string]string
	if err := json.Unmarshal(resp.Kvs[0].Value, &models); err != nil {
		return nil, err
	}

	if _, ok := models["default"]; !ok {
		models["default"] = m.cfg.DefaultExe
	}

	return models, nil
}

func (m *Manager) saveModels(ctx context.Context, models map[string]string) error {
	b, err := json.Marshal(models)
	if err != nil {
		return err
	}
	_, err = m.etcd.Put(ctx, ModelsKey, string(b))
	return err
}

func (m *Manager) SetModel(ctx context.Context, name, exePath string) (map[string]string, error) {
	models, err := m.GetModels(ctx)
	if err != nil {
		return nil, err
	}
	models[name] = exePath
	if err := m.saveModels(ctx, models); err != nil {
		return nil, err
	}
	return models, nil
}

func (m *Manager) DeleteModel(ctx context.Context, name string) (map[string]string, error) {
	if name == "default" {
		return nil, fmt.Errorf(`cannot delete the "default" model`)
	}
	models, err := m.GetModels(ctx)
	if err != nil {
		return nil, err
	}
	delete(models, name)
	if err := m.saveModels(ctx, models); err != nil {
		return nil, err
	}
	return models, nil
}
