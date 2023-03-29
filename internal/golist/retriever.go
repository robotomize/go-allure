package golist

import (
	"context"
	"fmt"
	"io/fs"
)

type PackageRetriever interface {
	Retrieve(ctx context.Context) ([]Package, error)
}

func NewRetriever(fs fs.FS, goBuildTags ...string) PackageRetriever {
	return &retriever{fs: fs, args: goBuildTags}
}

type retriever struct {
	fs   fs.FS
	args []string
}

func (r *retriever) Retrieve(ctx context.Context) ([]Package, error) {
	packages, err := DirPackages(ctx, r.fs, r.args...)
	if err != nil {
		return nil, fmt.Errorf("DirPackages: %w", err)
	}

	return packages, nil
}
