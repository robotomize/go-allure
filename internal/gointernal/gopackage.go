package gointernal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

type GoListPackage struct {
	Dir            string       `json:"Dir"`
	ImportPath     string       `json:"ImportPath"`
	Name           string       `json:"Name"`
	Root           string       `json:"Root"`
	Module         GoListModule `json:"Module"`
	Match          []string     `json:"Match"`
	Stale          bool         `json:"Stale"`
	StaleReason    string       `json:"StaleReason"`
	GoFiles        []string     `json:"GoFiles"`
	TestGoFiles    []string     `json:"TestGoFiles"`
	XTestGoFiles   []string     `json:"XTestGoFiles"`
	IgnoredGoFiles []string     `json:"IgnoredGoFiles"`
	Imports        []string     `json:"Imports"`
	Deps           []string     `json:"Deps"`
}

type GoListModule struct {
	Path      string `json:"Path"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
}

type GoTestFile struct {
	TestName     string
	PackageName  string
	FileName     string
	TestFileLine int
	TestFileCol  int
	GoVersion    string
}

func ReadGoModules(ctx context.Context, dir string, args ...string) ([]GoListPackage, error) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("os.ReadDir: %w", err)
	}

	modulePackages := make([]GoListPackage, 0)
	for _, entry := range dirs {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if entry.IsDir() {
			packages, err := ReadGoModules(ctx, filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("ReadGoModules: %w", err)
			}

			modulePackages = append(modulePackages, packages...)
		}

		if entry.Name() == "go.mod" {
			packages, err := ReadGoPackages(ctx, dir, args...)
			if err != nil {
				return nil, fmt.Errorf("ReadGoPackages: %w", err)
			}

			modulePackages = append(modulePackages, packages...)
		}
	}

	return modulePackages, nil
}

func ReadGoPackages(ctx context.Context, dir string, args ...string) ([]GoListPackage, error) {
	names, err := readGoPackageNames(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("readGoPackageNames: %w", err)
	}

	goListArgs := append([]string{"list", "-json"}, args...)
	goPackages := make([]GoListPackage, 0, len(names))

	ch := make(chan GoListPackage)

	closeCh := make(chan struct{})

	wg, childCtx := errgroup.WithContext(ctx)
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
		case <-childCtx.Done():
			break OuterLoop
		default:
		}

		wg.Go(
			func() error {
				select {
				case <-childCtx.Done():
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

				var goPackage GoListPackage
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

func ReadGoTestFiles(ctx context.Context, goListPackages []GoListPackage) ([]GoTestFile, error) {
	var goTestFiles []GoTestFile

	wg, childCtx := errgroup.WithContext(ctx)
	wg.SetLimit(runtime.NumCPU())

	ch := make(chan GoTestFile)
	closeCh := make(chan struct{})

	go func() {
		defer close(closeCh)

		for goTestFile := range ch {
			goTestFiles = append(goTestFiles, goTestFile)
		}
	}()

OuterLoop:
	for _, goListPackage := range goListPackages {
		goListPackage := goListPackage

		files := append(goListPackage.TestGoFiles, goListPackage.XTestGoFiles...)
		for _, file := range files {
			file := file

			select {
			case <-childCtx.Done():
				break OuterLoop
			default:
			}

			wg.Go(
				func() error {
					select {
					case <-childCtx.Done():
						return nil
					default:
					}

					sourceFilePath := fmt.Sprintf("%s/%s", goListPackage.Dir, file)
					goTestFiles, err := parseAstFile(sourceFilePath, goListPackage)
					if err != nil {
						return fmt.Errorf("parseAstFile: %w", err)
					}

					for _, goTestFile := range goTestFiles {
						ch <- goTestFile
					}

					return nil
				},
			)
		}
	}

	if err := wg.Wait(); err != nil {
		return nil, err
	}

	close(ch)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-closeCh:
	}

	return goTestFiles, nil
}

func parseAstFile(sourceFilePath string, goListPackage GoListPackage) ([]GoTestFile, error) {
	var goTestFiles []GoTestFile
	fileSet := token.NewFileSet()

	f, err := parser.ParseFile(fileSet, sourceFilePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parser.ParseFile: %w", err)
	}

	ast.Inspect(
		f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				fileSetPos := fileSet.Position(n.Pos())
				folders := strings.Split(fileSetPos.String(), "/")
				fileNameWithPos := folders[len(folders)-1]
				fileDetails := strings.Split(fileNameWithPos, ":")
				lineNum, _ := strconv.Atoi(fileDetails[1])
				colNum, _ := strconv.Atoi(fileDetails[2])

				goTestFiles = append(
					goTestFiles, GoTestFile{
						TestName:     x.Name.Name,
						PackageName:  goListPackage.ImportPath,
						FileName:     fileDetails[0],
						TestFileLine: lineNum,
						TestFileCol:  colNum,
						GoVersion:    goListPackage.Module.GoVersion,
					},
				)
			}

			return true
		},
	)

	return goTestFiles, nil
}

func readGoPackageNames(ctx context.Context, dir string) ([]string, error) {
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
		select {
		case <-ctx.Done():
		default:
		}

		packageNames = append(packageNames, scanner.Text())
	}

	return packageNames, nil
}
