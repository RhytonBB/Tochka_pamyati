package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalFS struct {
	baseDir string
}

func NewLocalFS(baseDir string) *LocalFS {
	return &LocalFS{baseDir: baseDir}
}

func (s *LocalFS) Save(ctx context.Context, path string, r io.Reader) error {
	full := filepath.Join(s.baseDir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(full, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (s *LocalFS) Delete(ctx context.Context, path string) error {
	full := filepath.Join(s.baseDir, filepath.FromSlash(path))
	err := os.Remove(full)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
