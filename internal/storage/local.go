package storage

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Provider interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) error
	PublicURL(key string) string
	DeleteObject(ctx context.Context, key string) error
	ObjectKeyFromPublicURL(url string) (string, bool)
}

type LocalProvider struct {
	rootDir string
	baseURL string
}

func NewLocalProvider(rootDir, baseURL string) LocalProvider {
	return LocalProvider{
		rootDir: strings.TrimRight(rootDir, "/"),
		baseURL: "/" + strings.Trim(strings.TrimSpace(baseURL), "/"),
	}
}

func (p LocalProvider) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	_ = contentType
	if err := ctx.Err(); err != nil {
		return err
	}

	cleanKey, err := cleanObjectKey(key)
	if err != nil {
		return err
	}
	filePath := filepath.Join(p.rootDir, filepath.FromSlash(cleanKey))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0o644)
}

func (p LocalProvider) PublicURL(key string) string {
	cleanKey, err := cleanObjectKey(key)
	if err != nil {
		return p.baseURL
	}
	return p.baseURL + "/" + cleanKey
}

func (p LocalProvider) DeleteObject(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cleanKey, err := cleanObjectKey(key)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(p.rootDir, filepath.FromSlash(cleanKey)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (p LocalProvider) ObjectKeyFromPublicURL(url string) (string, bool) {
	normalized := strings.SplitN(url, "?", 2)[0]
	prefix := strings.TrimRight(p.baseURL, "/") + "/"
	if !strings.HasPrefix(normalized, prefix) {
		return "", false
	}
	key := strings.TrimPrefix(normalized, prefix)
	cleanKey, err := cleanObjectKey(key)
	if err != nil {
		return "", false
	}
	return cleanKey, true
}

func cleanObjectKey(key string) (string, error) {
	cleanKey := strings.TrimPrefix(path.Clean("/"+key), "/")
	if cleanKey == "." || cleanKey == "" || strings.HasPrefix(cleanKey, "../") {
		return "", errors.New("invalid storage key")
	}
	return cleanKey, nil
}
