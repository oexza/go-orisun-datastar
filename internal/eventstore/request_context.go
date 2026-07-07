package eventstore

import "context"

type requestContextKey struct{}

func WithRequestLogMetadata(ctx context.Context, metadata map[string]any) context.Context {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return context.WithValue(ctx, requestContextKey{}, metadata)
}

func RequestLogMetadata(ctx context.Context) map[string]any {
	metadata, _ := ctx.Value(requestContextKey{}).(map[string]any)
	if metadata == nil {
		return map[string]any{}
	}
	copied := map[string]any{}
	for key, value := range metadata {
		copied[key] = value
	}
	return copied
}

func SetRequestAction(ctx context.Context, action string, fields map[string]any) {
	metadata, _ := ctx.Value(requestContextKey{}).(map[string]any)
	if metadata == nil {
		return
	}
	metadata["action"] = action
	if len(fields) > 0 {
		metadata["actionFields"] = fields
	}
}
