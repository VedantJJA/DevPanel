// Package ui provides access to the embedded Svelte SPA build artifacts.
//
// The ui/build directory must exist before running `go build` — run
// `npm run build` inside the ui/ directory first.
package ui

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed all:build
var buildFS embed.FS

// FS returns an fs.FS rooted at the build output directory.
// All files from ui/build are available at the top level.
func FS() fs.FS {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		log.Fatalf("ui.FS: %v", err)
	}
	return sub
}
