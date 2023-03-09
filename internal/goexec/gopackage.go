package goexec

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
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

func WalkModules(ctx context.Context, dir string, args ...string) ([]Package, error) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	modulePackages := make([]Package, 0)
	for _, entry := range dirs {
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "..") {
			continue
		}

		if entry.IsDir() {
			packages, err := WalkModules(ctx, filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("WalkModules: %w", err)
			}

			modulePackages = append(modulePackages, packages...)
		}

		if entry.Name() == "go.mod" {
			packages, err := ListPackages(ctx, dir, args...)
			if err != nil {
				return nil, fmt.Errorf("ListPackages: %w", err)
			}

			modulePackages = append(modulePackages, packages...)
		}
	}

	return modulePackages, nil
}

func ListPackages(ctx context.Context, dir string, args ...string) ([]Package, error) {
	names, err := readPackageNames(dir)
	if err != nil {
		return nil, fmt.Errorf("readPackageNames: %w", err)
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

				buf := bytes.NewBuffer(make([]byte, 0, 4096))
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

func readPackageNames(dir string) ([]string, error) {
	var packageNames []string

	buf := bytes.NewBuffer(make([]byte, 0, 4096))
	goListCmd := exec.Command("go", "list", "./...")
	goListCmd.Stdin = strings.NewReader("")
	goListCmd.Stdout = buf
	goListCmd.Dir = dir
	if err := goListCmd.Run(); err != nil {
		return nil, fmt.Errorf("command Run go list ./...: %w", err)
	}

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		packageNames = append(packageNames, scanner.Text())
	}

	return packageNames, nil
}
