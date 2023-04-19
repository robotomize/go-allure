package fs

import (
	"io/fs"
	"os"
)

type FS interface {
	fs.FS
	RootDir() string
}

var _ FS = (*rootDirFS)(nil)

func New(entry string) FS {
	return &rootDirFS{entry: entry, FS: os.DirFS(entry)}
}

type rootDirFS struct {
	fs.FS
	entry string
}

func (r rootDirFS) RootDir() string {
	return r.entry
}
