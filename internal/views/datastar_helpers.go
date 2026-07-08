package views

import (
	"fmt"
	"os"
)

func postFormSSE(path string) string {
	return fmt.Sprintf("@post('%s', {contentType: 'form'})", path)
}

func longRunningGetSSE(path string) string {
	return fmt.Sprintf("@get('%s', {requestCancellation: 'disabled', retryMaxCount: 1000, retryInterval: 1000, retryMaxWaitMs: 5000})", path)
}

func hotReloadSSE() string {
	return "@get('/reload', {retryMaxCount: 1000, retryInterval: 20, retryMaxWaitMs: 200})"
}

func isDevelopment() bool {
	return os.Getenv("NODE_ENV") != "production"
}
