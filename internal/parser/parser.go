package parser

import (
	"context"
	"fmt"

	"github.com/robotomize/go-allure/internal/golist"
)

type PackageRetriever interface {
	Retrieve(ctx context.Context) ([]golist.Package, error)
}

func New(packageRetriever PackageRetriever) *Parser {
	return &Parser{PackageRetriever: packageRetriever}
}

type Parser struct {
	PackageRetriever
}

// ParseFiles retrieves Go packages using the PackageRetriever and parses their test files.
func (p *Parser) ParseFiles(ctx context.Context) ([]GoTestMethod, error) {
	packages, err := p.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("PackageRetriever Retrieve: %w", err)
	}

	files, err := ParseTestFiles(ctx, packages)
	if err != nil {
		return nil, fmt.Errorf("ParseTestFiles: %w", err)
	}

	return files, nil
}
