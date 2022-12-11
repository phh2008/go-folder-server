package main

import (
	"embed"
	"io/fs"
)

//go:embed index.html
var Index embed.FS

//go:embed 404.html
var NotFound embed.FS

// fsFunc is short-hand for constructing a http.FileSystem
// implementation
type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}
