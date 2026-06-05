package profile

import "context"

type ObjectStore interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) error
	PublicURL(key string) string
}
