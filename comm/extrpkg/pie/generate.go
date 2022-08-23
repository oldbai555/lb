//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/elliotchance/pie/functions"
)

var packageTemplate = template.Must(template.New("").
	Parse("// Code generated by go generate; DO NOT EDIT.\n" +
		"package main\n" +
		"\n" +
		"var pieTemplates = map[string]string{\n" +
		"{{ range $fn, $file := . }}" +
		"\t\"{{ $fn }}\": `{{ $file }}`,\n" +
		"{{ end }}" +
		"}\n"))

type Function struct {
	Name            string
	DescriptiveName string
	For             int
	Doc             string
	BigO            string
}

func (f Function) BriefDoc() (brief string) {
	for _, line := range strings.Split(f.Doc, "\n") {
		if line == "" {
			break
		}

		brief += line + " "
	}

	return
}

func updateREADME() {
	var funcs []Function

	for _, function := range functions.Functions {
		file, err := ioutil.ReadFile("functions/" + function.File)
		if err != nil {
			panic(err)
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", file, parser.ParseComments)
		if err != nil {
			panic(err)
		}

		var qualifiers []string

		if strings.Contains(string(file), ".Equals(") {
			qualifiers = append(qualifiers, "E")
		}

		if strings.Contains(string(file), ".String(") {
			qualifiers = append(qualifiers, "S")
		}

		descriptiveName := fmt.Sprintf("[`%s`](#%s)", function.Name, strings.ToLower(function.Name))

		if len(qualifiers) > 0 {
			descriptiveName += fmt.Sprintf(" (%s)", strings.Join(qualifiers, ""))
		}

		funcs = append(funcs, Function{
			Name:            function.Name,
			DescriptiveName: descriptiveName,
			For:             function.For,
			Doc:             getDoc(f.Decls),
			BigO:            function.BigO,
		})
	}

	longestFunctionName := 0
	for _, function := range funcs {
		if newLen := len(function.DescriptiveName); newLen > longestFunctionName {
			longestFunctionName = newLen
		}
	}

	newDocs := fmt.Sprintf("| Function%s | String | Number | Struct | Maps | Big-O    | Description |\n", strings.Repeat(" ", longestFunctionName-8))
	newDocs += fmt.Sprintf("| %s | :----: | :----: | :----: | :--: | :------: | ----------- |\n", strings.Repeat("-", longestFunctionName))

	for _, function := range funcs {
		newDocs += fmt.Sprintf("| %s%s | %s      | %s      | %s      | %s    | %s%s | %s |\n",
			function.DescriptiveName,
			strings.Repeat(" ", longestFunctionName-len(function.DescriptiveName)),
			tick(function.For&functions.ForStrings),
			tick(function.For&functions.ForNumbers),
			tick(function.For&functions.ForStructs),
			tick(function.For&functions.ForMaps),
			function.BigO,
			strings.Repeat(" ", 8-utf8.RuneCountInString(function.BigO)),
			function.BriefDoc())
	}

	newDocs += "\n"

	for _, function := range funcs {
		newDocs += fmt.Sprintf("## %s\n\n%s\n",
			function.Name, function.Doc)
	}

	readme, err := ioutil.ReadFile("README.md")
	if err != nil {
		panic(err)
	}

	newReadme := regexp.MustCompile(`(?s)\| Function.*# FAQ`).
		ReplaceAllString(string(readme), newDocs+"# FAQ")

	err = ioutil.WriteFile("README.md", []byte(newReadme), 0644)
	if err != nil {
		panic(err)
	}
}

func tick(x int) string {
	if x != 0 {
		return "✓"
	}

	return " "
}

func getDoc(decls []ast.Decl) (doc string) {
	for _, decl := range decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Doc != nil {
			for _, comment := range f.Doc.List {
				if len(comment.Text) < 3 {
					doc += "\n"
				} else {
					doc += comment.Text[3:] + "\n"
				}
			}
		}
	}

	// Fix code snippets
	var newLines []string
	inCodeBlock := false

	for _, line := range strings.Split(doc, "\n") {
		if line == "" {
			newLines = append(newLines, "")
		}

		if strings.HasPrefix(line, "  ") {
			if inCodeBlock {
				newLines = append(newLines, line[2:])
			} else {
				inCodeBlock = true
				newLines = append(newLines, "```go", line[2:])
			}
		} else {
			if inCodeBlock {
				inCodeBlock = false
				newLines = append(newLines, "```", line)
			} else {
				newLines = append(newLines, line)
			}
		}
	}

	return strings.Join(newLines, "\n")
}

func main() {
	data := map[string]string{}

	for _, function := range functions.Functions {
		tmpl, err := ioutil.ReadFile("functions/" + function.File)
		if err != nil {
			panic(err)
		}

		data[function.Name] = string(tmpl)
	}

	f, err := os.Create("template.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	packageTemplate.Execute(f, data)

	updateREADME()
}
