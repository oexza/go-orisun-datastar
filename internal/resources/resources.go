package resources

import (
	"net/http"
	"path"
	"strings"
)

const StaticDirectoryPath = "static"

func Handler() http.Handler {
	files := http.StripPrefix("/static/", http.FileServer(http.Dir(StaticDirectoryPath)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
}

func StaticPath(assetPath string) string {
	return "/static/" + strings.TrimPrefix(path.Clean(assetPath), "/")
}
