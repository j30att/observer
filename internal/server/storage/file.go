package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"j30att/observer/internal/server/model"
)

type FileStorage struct {
	path   string
	logger zerolog.Logger
}

func NewFileStorage(path string, logger zerolog.Logger) *FileStorage {
	return &FileStorage{
		path:   path,
		logger: logger,
	}
}

func NewRestoredFileStorage(path string, logger zerolog.Logger) (*FileStorage, []model.Metrics, error) {
	storage := NewFileStorage(path, logger)
	metrics, err := storage.load()
	if err != nil {
		return nil, nil, err
	}

	return storage, metrics, nil
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
			if err := os.Remove(tmpPath); err != nil {
				s.logger.Error().Err(err).Str("path", tmpPath).Msg("failed to remove temporary metrics file")
			}
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

func (s *FileStorage) load() ([]model.Metrics, error) {
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
