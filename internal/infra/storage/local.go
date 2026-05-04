package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	dir     string
	baseURL string
}

func NewLocalStorage(dir, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{dir: dir, baseURL: baseURL}, nil
}

func (s *LocalStorage) Upload(_ context.Context, filename string, src io.Reader) (string, error) {
	dst, err := os.Create(filepath.Join(s.dir, filename))
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return s.baseURL + "/uploads/" + filename, nil
}

func (s *LocalStorage) Delete(_ context.Context, filename string) error {
	err := os.Remove(filepath.Join(s.dir, filename))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
