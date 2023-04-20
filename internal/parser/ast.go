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

	"github.com/robotomize/go-allure/internal/golist"
	"golang.org/x/sync/errgroup"
)

type GoTestMethod struct {
	TestName     string
	PackageName  string
	FileName     string
	TestFileLine int
	TestFileCol  int
	GoVersion    string
}

// ParseTestFiles - parse go test files into slice of GoTestMethod.
func ParseTestFiles(ctx context.Context, packages []golist.Package) ([]GoTestMethod, error) {
	var goTestFiles []GoTestMethod

	// Use errgroup to limit the number of goroutines.
	// Declare a child context and a wait group.
	wg, childCtx := errgroup.WithContext(ctx)
	wg.SetLimit(runtime.NumCPU())

	ch := make(chan GoTestMethod)
	closeCh := make(chan struct{})

	// Start a goroutine to append the GoTestMethod objects to the goTestFiles slice.
	go func() {
		defer close(closeCh)

		for goTestFile := range ch {
			goTestFiles = append(goTestFiles, goTestFile)
		}
	}()

	// Loop through the packages and test files to parse them.
	// Use errgroup to run the parsing concurrently.
OuterLoop:
	for _, pkg := range packages {
		pkg := pkg

		files := append(pkg.TestGoFiles, pkg.XTestGoFiles...)
		for _, file := range files {
			file := file

			// Check if the child context is done to break out of the loop.
			select {
			case <-childCtx.Done():
				break OuterLoop
			default:
			}

			// Use errgroup to parse the test files concurrently.
			wg.Go(
				func() error {
					// Check if the child context is done to return early.
					select {
					case <-childCtx.Done():
						return nil
					default:
					}

					// Build the path to the test file and parse it.
					pth := fmt.Sprintf("%s/%s", pkg.Dir, file)
					files, err := parse(pth, pkg)
					if err != nil {
						return fmt.Errorf("parse: %w", err)
					}

					// Send the parsed GoTestMethod objects to the channel.
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

	// Check if the context is done to return early.
	// Otherwise, wait for the channel to close and return the slice of GoTestMethod objects.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-closeCh:
	}

	return goTestFiles, nil
}

// parse - parse go files into slice of func declarations.
func parse(pth string, pkg golist.Package) ([]GoTestMethod, error) {
	fileSet := token.NewFileSet()

	// Use the parser.ParseFile method to parse the test file.
	f, err := parser.ParseFile(fileSet, pth, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("Parser.ParseFile: %w", err)
	}

	var files []GoTestMethod

	// Traverse the AST of the parsed file and process the test function declarations.
	ast.Inspect(
		f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				// Get the position of the test function declaration in the file.
				fileSetPos := fileSet.Position(n.Pos())

				// Extract the file name, line number, and column number from the position object.
				folders := strings.Split(fileSetPos.String(), "/")
				fileNameWithPos := folders[len(folders)-1]
				fileDetails := strings.Split(fileNameWithPos, ":")
				lineNum, _ := strconv.Atoi(fileDetails[1])
				colNum, _ := strconv.Atoi(fileDetails[2])

				files = append(
					files, GoTestMethod{
						TestName:     x.Name.Name,
						PackageName:  pkg.ImportPath,
						FileName:     fileDetails[0],
						TestFileLine: lineNum,
						TestFileCol:  colNum,
						GoVersion:    pkg.Module.GoVersion,
					},
				)
			}

			// Always return true to continue the traversal of the AST.
			return true
		},
	)

	return files, nil
}
