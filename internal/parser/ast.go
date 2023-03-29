package parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/robotomize/go-allure/internal/golist"
)

type GoTestFile struct {
	TestName     string
	PackageName  string
	FileName     string
	TestFileLine int
	TestFileCol  int
	GoVersion    string
}

func ParseTestFiles(ctx context.Context, packages []golist.Package) ([]GoTestFile, error) {
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
	for _, pkg := range packages {
		pkg := pkg

		files := append(pkg.TestGoFiles, pkg.XTestGoFiles...)
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

					pth := fmt.Sprintf("%s/%s", pkg.Dir, file)
					files, err := parse(pth, pkg)
					if err != nil {
						return fmt.Errorf("parse: %w", err)
					}

					for _, f := range files {
						ch <- f
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

func parse(pth string, pkg golist.Package) ([]GoTestFile, error) {
	fileSet := token.NewFileSet()

	f, err := parser.ParseFile(fileSet, pth, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("Parser.ParseFile: %w", err)
	}

	var files []GoTestFile
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

				files = append(
					files, GoTestFile{
						TestName:     x.Name.Name,
						PackageName:  pkg.ImportPath,
						FileName:     fileDetails[0],
						TestFileLine: lineNum,
						TestFileCol:  colNum,
						GoVersion:    pkg.Module.GoVersion,
					},
				)
			}

			return true
		},
	)

	return files, nil
}
