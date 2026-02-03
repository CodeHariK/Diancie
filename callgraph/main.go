package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type InfoType string

const (
	FUNCTION InfoType = "FUNCTION"
	COMMENT           = "COMMENT"
	GLOBAL            = "GLOBAL"
	STRUCT            = "STRUCT"
)

// Info holds information about a function and its calls.
type Info struct {
	Name      string
	Type      InfoType
	Calls     []string
	Comments  []string
	Content   string
	Fields    map[string]string
	Range     [2]int
	Processed bool
}

// ParseGoFile parses a Go file and extracts functions, global variables, and structs.
func ParseGoFile(filename string, withContent bool) ([]Info, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	commentsMap := make(map[int]Info)
	infoMap := make(map[int]Info)

	for _, commentGroup := range node.Comments {
		for _, comment := range commentGroup.List {
			commentsMap[fset.Position(comment.Pos()).Line] = Info{Type: COMMENT, Comments: []string{NormalizeComment(comment.Text)}}
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {

		case *ast.FuncDecl: // Function declaration
			funcName := n.Name.Name

			start, end := n.Pos(), n.End()
			pos := fset.Position(n.Pos())
			endPos := fset.Position(n.Body.Rbrace)

			funcInfo := Info{
				Type:    FUNCTION,
				Name:    fmt.Sprintf("%s %s:%d", funcName, filename, pos.Line),
				Range:   [2]int{pos.Line, endPos.Line},
				Content: Ternary(withContent, string(src[start-1:end]), ""),
			}

			// Find function calls inside body
			ast.Inspect(n.Body, func(bodyNode ast.Node) bool {
				switch bodyNode := bodyNode.(type) {
				case *ast.CallExpr: // Function call
					if ident, ok := bodyNode.Fun.(*ast.Ident); ok {
						callPos := fset.Position(bodyNode.Pos())
						funcInfo.Calls = append(funcInfo.Calls,
							fmt.Sprintf("%s %s:%d", ident.Name, filename, callPos.Line))
					}
				}
				return true
			})

			infoMap[fset.Position(n.Pos()).Line] = funcInfo

		case *ast.GenDecl: // General declarations (variables, structs, etc.)
			if n.Tok == token.IMPORT {
				return true // Skip imports
			}

			for _, spec := range n.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec: // Global variable
					for _, ident := range spec.Names {

						start, end := n.Pos(), n.End()
						pos := fset.Position(ident.Pos())
						endPos := fset.Position(spec.End())

						typeName := "unknown"
						if spec.Type != nil {
							typeName = fmt.Sprintf("%s", spec.Type)
						}

						infoMap[fset.Position(n.Pos()).Line] = Info{
							Name:    fmt.Sprintf("%s %s:%d %s", ident.Name, filename, pos.Line, typeName),
							Type:    GLOBAL,
							Range:   [2]int{pos.Line, endPos.Line},
							Content: Ternary(withContent, string(src[start-1:end]), ""),
						}

					}

				case *ast.TypeSpec: // Structs
					if structType, ok := spec.Type.(*ast.StructType); ok {

						start, end := n.Pos(), n.End()
						pos := fset.Position(spec.Pos())
						endPos := fset.Position(structType.Fields.Closing)

						fields := make(map[string]string)

						for _, field := range structType.Fields.List {
							fieldType := "unknown"
							if field.Type != nil {
								fieldType = fmt.Sprintf("%s", field.Type)
							}
							for _, fieldName := range field.Names {
								fields[fieldName.Name] = fieldType
							}
						}

						infoMap[pos.Line] = Info{
							Name:    fmt.Sprintf("%s %s:%d", spec.Name.Name, filename, pos.Line),
							Type:    STRUCT,
							Fields:  fields,
							Range:   [2]int{pos.Line, endPos.Line},
							Content: Ternary(withContent, string(src[start-1:end]), ""),
						}
					}
				}
			}
		}
		return true
	})

	sortedInfos := SortMapByKeyDesc(infoMap)
	sortedComments := SortMapByKeyDesc(commentsMap)

	infos := []Info{}

	for _, info := range sortedInfos {
		for _, comment := range sortedComments {
			if comment.Key < info.Value.Range[1] && !comment.Value.Processed {
				info.Value.Comments = append(info.Value.Comments, comment.Value.Comments...)
				comment.Value.Processed = true
			}
		}
		infos = append(infos, *info.Value)
	}

	return infos, nil
}

func NormalizeComment(comment string) string {
	// Remove line comment markers (//)
	comment = strings.TrimPrefix(comment, "//")

	// Remove block comment markers (/* */)
	comment = strings.TrimPrefix(comment, "/*")
	comment = strings.TrimSuffix(comment, "*/")

	// Remove leading '*' in block comments (common in formatted comments)
	comment = regexp.MustCompile(`(?m)^\s*\*\s?`).ReplaceAllString(comment, "")

	// Trim leading and trailing spaces
	return strings.TrimSpace(comment)
}

// Main function to run the parser on a file
func main() {
	withContent := false

	var infos []Info
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return nil
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Process only Go files
		if strings.HasSuffix(path, ".go") {

			// Call your function
			fileInfo, err := ParseGoFile(path, withContent)
			if err != nil {
				fmt.Println("Error parsing file:", err)
				return nil
			}

			infos = append(infos, fileInfo...)
		}

		return nil
	})
	if err != nil {
		fmt.Println("Error walking directory:", err)
	}

	file, err := os.OpenFile("callgraph.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Println("Error opening coverage file:", err)
		return
	}
	defer file.Close()

	if err = json.NewEncoder(file).Encode(infos); err != nil {
		fmt.Println(err)
	}
}

func cover() {
	// Run tests and generate cover.out
	cmdTest := exec.Command("go", "test", "-coverprofile=cover.txt", "./...")
	cmdTest.Stdout = os.Stdout
	cmdTest.Stderr = os.Stderr

	if err := cmdTest.Run(); err != nil {
		fmt.Println("Error running tests:", err)
		return
	}

	// Append coverage results to cover.txt
	file, err := os.OpenFile("cover.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Println("Error opening coverage file:", err)
		return
	}
	defer file.Close()

	cmdCover := exec.Command("go", "tool", "cover", "-html=./cover.txt", "-o", "./cover.html")
	cmdCover.Stdout = file
	cmdCover.Stderr = os.Stderr

	if err := cmdCover.Run(); err != nil {
		fmt.Println("Error generating coverage report:", err)
		return
	}

	fmt.Println("Coverage report appended to cover.txt")
}
