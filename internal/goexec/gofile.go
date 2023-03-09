package goexec

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
)

type GoTestFile struct {
	TestName     string
	PackageName  string
	FileName     string
	TestFileLine int
	TestFileCol  int
	GoVersion    string
}

func ParseTestFiles(ctx context.Context, goListPackages []Package) ([]GoTestFile, error) {
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
					goTestFiles, err := parseFile(sourceFilePath, goListPackage)
					if err != nil {
						return fmt.Errorf("parseFile: %w", err)
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

func parseFile(sourceFilePath string, goListPackage Package) ([]GoTestFile, error) {
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
