package migrations

import (
	"embed"

	toolbeltdb "github.com/delaneyj/toolbelt/db"
)

//go:embed *.sql
var files embed.FS

func SQL() ([]string, error) {
	return toolbeltdb.MigrationsFromFS(files, ".")
}
