package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"j30att/observer/internal/server/model"
)

type FileStorage struct {
	path string
}

func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: path}
}

func (s *FileStorage) Save(metrics []model.Metrics) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(s.path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	encoder := json.NewEncoder(tmpFile)
	if err := encoder.Encode(metrics); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}

	removeTmp = false
	return nil
}

func (s *FileStorage) Load() ([]model.Metrics, error) {
	file, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	var metrics []model.Metrics
	if err := json.NewDecoder(file).Decode(&metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}
