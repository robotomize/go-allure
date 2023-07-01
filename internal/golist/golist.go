package golist

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"
)

type Package struct {
	Dir            string   `json:"Dir"`
	ImportPath     string   `json:"ImportPath"`
	Name           string   `json:"Name"`
	Root           string   `json:"Root"`
	Module         Module   `json:"Module"`
	Match          []string `json:"Match"`
	Stale          bool     `json:"Stale"`
	StaleReason    string   `json:"StaleReason"`
	GoFiles        []string `json:"GoFiles"`
	TestGoFiles    []string `json:"TestGoFiles"`
	XTestGoFiles   []string `json:"XTestGoFiles"`
	IgnoredGoFiles []string `json:"IgnoredGoFiles"`
	Imports        []string `json:"Imports"`
	Deps           []string `json:"Deps"`
}

type Module struct {
	Path      string `json:"Path"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
}

// DirPackages - walk fs and collects all go packages found on directory.
func DirPackages(ctx context.Context, dfs FS, args ...string) ([]Package, error) {
	packages := make([]Package, 0)

	// Use fs.WalkDir to recursively walk through the file system and detect Go modules.
	if err := fs.WalkDir(
		dfs, ".", func(pth string, entry fs.DirEntry, err error) error {
			// Skip hidden directories and files, and non-directories.
			skip := strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "..") || entry.IsDir()
			if skip {
				return nil
			}

			if entry.Name() == "go.mod" {
				d, _ := filepath.Split(pth)
				if d == "" {
					d = dfs.RootDir()
				}

				list, err := listPackages(ctx, d, args...)
				if err != nil {
					return fmt.Errorf("listPackages: %w", err)
				}

				packages = append(packages, list...)
			}

			return nil
		},
	); err != nil {
		return nil, fmt.Errorf("fs.WalkDir: %w", err)
	}

	return packages, nil
}

// list go packages in given directory.
func listPackages(ctx context.Context, dir string, args ...string) ([]Package, error) {
	var pkgNames []string

	// Use the goList function to list all packages in the directory, append them to the pkgNames slice.
	packagesBuf, err := goList(ctx, dir, append(args, "./..."))
	if err != nil {
		return nil, err
	}

	// Scan the buffer and collect package names
	scanner := bufio.NewScanner(packagesBuf)
	for scanner.Scan() {
		if err = scanner.Err(); err != nil {
			return nil, fmt.Errorf("bufio.NewScaner.Err: %w", err)
		}

		pkgNames = append(pkgNames, scanner.Text())
	}

	goPkgs := make([]Package, 0, len(pkgNames))

	// Use errgroup to limit the number of concurrent goroutines.
	// Declare a channel to hold the parsed Package objects, and a wait group.
	ch := make(chan Package)
	closeCh := make(chan struct{})
	wg, grpCtx := errgroup.WithContext(ctx)
	wg.SetLimit(runtime.NumCPU())

	// Start a goroutine to append the parsed Package objects to the goPkgs slice.
	go func() {
		defer close(closeCh)

		for pkg := range ch {
			goPkgs = append(goPkgs, pkg)
		}
	}()

	// Loop through the package names and use errgroup to parse the package data concurrently.
OuterLoop:
	for _, pkgName := range pkgNames {
		pkg := pkgName

		// Check if the group context is done to break out of the loop.
		select {
		case <-grpCtx.Done():
			break OuterLoop
		default:
		}

		// Use errgroup to run the goList command on the package name and parse the returned JSON data.
		wg.Go(
			func() error {
				pkgArgs := append([]string{"-json"}, args...)
				pkgArgs = append(pkgArgs, pkg)

				packageBuf, pkgErr := goList(grpCtx, dir, pkgArgs)
				if pkgErr != nil {
					return fmt.Errorf("goList: %w", pkgErr)
				}

				var goPkg Package
				if dErr := json.NewDecoder(packageBuf).Decode(&goPkg); dErr != nil {
					return fmt.Errorf("json.NewDecoder.Decode: %w", dErr)
				}

				// Send the parsed Package object to the channel.
				ch <- goPkg

				return nil
			},
		)
	}

	if err = wg.Wait(); err != nil {
		return nil, err
	}

	close(ch)

	// Check if the context is done to return early.
	// Otherwise, wait for the channel to close and return the slice of Package objects.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-closeCh:
	}

	return goPkgs, nil
}

func goList(ctx context.Context, dir string, args []string) (io.Reader, error) {
	const bufSize = 4096

	b := bytes.NewBuffer(make([]byte, 0, bufSize))
	cmd := exec.CommandContext(ctx, "go", append([]string{"list"}, args...)...)
	cmd.Stdout = b
	cmd.Dir = dir

	cmd.Stdin = strings.NewReader("")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command Run go list %s: %w", strings.Join(args, " "), err)
	}

	b1 := bytes.NewBuffer(b.Bytes())

	return b1, nil
}
