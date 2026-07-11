package golang

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/mparvin/repo-miner/internal/analyzer"
	"github.com/mparvin/repo-miner/internal/core/domain"
)

func init() {
	analyzer.Register("golang", New)
}

// Analyzer implements Go source code analysis via AST parsing.
type Analyzer struct{}

// New creates a new Go analyzer from config.
func New(_ map[string]string) (analyzer.LanguageAnalyzer, error) {
	return &Analyzer{}, nil
}

func (a *Analyzer) Name() string { return "golang" }

func (a *Analyzer) Analyze(_ context.Context, repoPath string) (*domain.AnalysisResult, error) {
	result := &domain.AnalysisResult{Language: "go"}
	depSet := make(map[string]struct{})
	packagesWithTests := make(map[string]bool)
	allPackages := make(map[string]bool)

	var totalComplexity int
	var funcCount int

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		isTest := strings.HasSuffix(path, "_test.go")
		if isTest {
			result.TestFiles++
			pkg := filepath.Dir(path)
			packagesWithTests[pkg] = true
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		pkgName := node.Name.Name
		if pkgName != "" {
			allPackages[pkgName] = true
		}

		for _, imp := range node.Imports {
			dep := strings.Trim(imp.Path.Value, `"`)
			depSet[dep] = struct{}{}
		}

		ast.Inspect(node, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.FuncDecl:
				funcCount++
				if decl.Doc != nil && len(decl.Doc.List) > 0 {
					result.DocumentedFunctions++
				}
				totalComplexity += cyclomaticComplexity(decl.Body)
			case *ast.GenDecl:
				if decl.Tok == token.TYPE {
					for _, spec := range decl.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						switch ts.Type.(type) {
						case *ast.StructType:
							result.Structs++
						case *ast.InterfaceType:
							result.Interfaces++
						}
					}
				}
			}
			return true
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	result.Packages = len(allPackages)
	result.Functions = funcCount
	result.Dependencies = sortedKeys(depSet)
	result.HasTests = result.TestFiles > 0

	if funcCount > 0 {
		result.ComplexityScore = clamp(100 - float64(totalComplexity)/float64(funcCount)*2, 0, 100)
	} else {
		result.ComplexityScore = 100
	}

	pkgCount := 0
	for pkg := range allPackages {
		_ = pkg
		pkgCount++
	}
	testedPkgs := len(packagesWithTests)
	if pkgCount > 0 {
		result.TestCoverageSignal = float64(testedPkgs) / float64(pkgCount) * 100
	}

	result.StructureQualityScore = structureScore(repoPath, result.HasTests)

	return result, nil
}

func cyclomaticComplexity(body *ast.BlockStmt) int {
	if body == nil {
		return 1
	}
	complexity := 1
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.TypeSwitchStmt, *ast.SwitchStmt, *ast.SelectStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		case *ast.CommClause:
			complexity++
		case *ast.BinaryExpr:
			if node.Op.String() == "&&" || node.Op.String() == "||" {
				complexity++
			}
		}
		return true
	})
	return complexity
}

func structureScore(repoPath string, hasTests bool) float64 {
	score := 0.0
	checks := []struct {
		path   string
		points float64
	}{
		{"go.mod", 20},
		{"README.md", 15},
		{"cmd", 15},
		{"internal", 10},
		{"docs", 10},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(repoPath, c.path)); err == nil {
			score += c.points
		}
	}
	if hasTests {
		score += 20
	}
	return clamp(score, 0, 100)
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
