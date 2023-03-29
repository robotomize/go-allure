package golist

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func DirPackages(ctx context.Context, sfs fs.FS, args ...string) ([]Package, error) {
	packages := make([]Package, 0)

	if err := fs.WalkDir(
		sfs, ".", func(pth string, entry fs.DirEntry, err error) error {
			skip := strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "..") || entry.IsDir()
			if skip {
				return nil
			}

			if entry.Name() == "go.mod" {
				d, _ := filepath.Split(pth)
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

func listPackages(ctx context.Context, dir string, args ...string) ([]Package, error) {
	names, err := packageNames(dir)
	if err != nil {
		return nil, fmt.Errorf("packageNames: %w", err)
	}

	goListArgs := append([]string{"list", "-json"}, args...)
	goPackages := make([]Package, 0, len(names))

	ch := make(chan Package)

	closeCh := make(chan struct{})

	wg, grpCtx := errgroup.WithContext(ctx)
	wg.SetLimit(runtime.NumCPU())

	go func() {
		defer close(closeCh)

		for goPackage := range ch {
			goPackages = append(goPackages, goPackage)
		}
	}()

OuterLoop:
	for _, packageName := range names {
		packageName := packageName

		select {
		case <-grpCtx.Done():
			break OuterLoop
		default:
		}

		wg.Go(
			func() error {
				select {
				case <-grpCtx.Done():
					return nil
				default:
				}

				const sampleBufferSize = 4096
				buf := bytes.NewBuffer(make([]byte, 0, sampleBufferSize))
				goListCmd := exec.Command("go", append(goListArgs, packageName)...)
				goListCmd.Stdout = buf
				goListCmd.Dir = dir
				goListCmd.Stdin = strings.NewReader("")
				if err = goListCmd.Run(); err != nil {
					return fmt.Errorf("command Run go list -json %s.: %w", packageName, err)
				}

				var goPackage Package
				if err = json.Unmarshal(buf.Bytes(), &goPackage); err != nil {
					return fmt.Errorf("json.Unmarshal: %w", err)
				}

				ch <- goPackage

				return nil
			},
		)
	}

	if err = wg.Wait(); err != nil {
		return nil, err
	}

	close(ch)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-closeCh:
	}

	return goPackages, nil
}

func packageNames(dir string) ([]string, error) {
	var names []string

	buf := bytes.NewBuffer(make([]byte, 0, 4096))
	goListCmd := exec.Command("go", "list", "./...")
	goListCmd.Stdin = strings.NewReader("")
	goListCmd.Stdout = buf
	goListCmd.Dir = dir
	if err := goListCmd.Run(); err != nil {
		return nil, fmt.Errorf("command Run go list %s/./...: %w", dir, err)
	}

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		names = append(names, scanner.Text())
	}

	return names, nil
}
