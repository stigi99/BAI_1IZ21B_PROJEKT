//go:build ignore

// generate_godocs builds a Markdown API reference from the Go source tree.
//
// The standard go doc tooling focuses primarily on exported identifiers. This
// project also needs laboratory documentation for private helpers and tests, so
// this generator walks the Go AST directly and records every function, method,
// type, constant and variable from the active source files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// parameter describes one named input or output field in a function signature.
type parameter struct {
	Name string
	Type string
}

// functionDoc captures the metadata needed to render one function entry.
type functionDoc struct {
	Name       string
	Receiver   string
	Signature  string
	Comment    string
	File       string
	Line       int
	Exported   bool
	Parameters []parameter
	Returns    []parameter
}

// valueDoc captures a documented constant or variable declaration.
type valueDoc struct {
	Name     string
	Kind     string
	Type     string
	Value    string
	Comment  string
	File     string
	Line     int
	Exported bool
}

// typeDoc captures a documented type declaration and its fields.
type typeDoc struct {
	Name     string
	Kind     string
	Comment  string
	File     string
	Line     int
	Exported bool
	Fields   []parameter
}

// packageDoc groups all documented declarations that belong to one package.
type packageDoc struct {
	Name      string
	Files     []string
	Types     []typeDoc
	Values    []valueDoc
	Functions []functionDoc
}

// main parses the source tree and writes the generated GoDocs Markdown file.
func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "docs/generated/godocs/BAI_GoDocs.md", "output Markdown file")
	generatedAt := flag.String("date", time.Now().Format("2006-01-02 15:04:05 MST"), "generation timestamp written to the document")
	flag.Parse()

	packages, skipped, err := collectDocs(*root)
	if err != nil {
		fatal(err)
	}
	content := renderMarkdown(*root, packages, skipped, *generatedAt)

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*out, []byte(content), 0o644); err != nil {
		fatal(err)
	}
}

// collectDocs walks the repository and builds documentation records for all Go
// source files that are meant to be documented.
func collectDocs(root string) ([]packageDoc, []string, error) {
	var files []string
	var skipped []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == ".claude" || name == "node_modules" || name == "docs" || strings.HasPrefix(path, filepath.Join(root, "Bai orginal")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "pages_templ.go") {
			skipped = append(skipped, cleanPath(root, path)+" - kod generowany przez Templ")
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(files)

	fset := token.NewFileSet()
	byPackage := map[string]*packageDoc{}
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, err
		}
		pkgPath := cleanPath(root, filepath.Dir(path))
		if pkgPath == "." {
			pkgPath = "root"
		}
		key := pkgPath + "|" + file.Name.Name
		doc := byPackage[key]
		if doc == nil {
			doc = &packageDoc{Name: pkgPath}
			byPackage[key] = doc
		}
		doc.Files = append(doc.Files, cleanPath(root, path))
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				collectGenDecl(root, fset, path, d, doc)
			case *ast.FuncDecl:
				doc.Functions = append(doc.Functions, collectFuncDecl(root, fset, path, d))
			}
		}
	}

	var packages []packageDoc
	for _, doc := range byPackage {
		sort.Strings(doc.Files)
		sort.Slice(doc.Types, func(i, j int) bool { return doc.Types[i].Name < doc.Types[j].Name })
		sort.Slice(doc.Values, func(i, j int) bool { return doc.Values[i].Name < doc.Values[j].Name })
		sort.Slice(doc.Functions, func(i, j int) bool {
			if doc.Functions[i].File == doc.Functions[j].File {
				return doc.Functions[i].Line < doc.Functions[j].Line
			}
			return doc.Functions[i].File < doc.Functions[j].File
		})
		packages = append(packages, *doc)
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Name < packages[j].Name })
	return packages, skipped, nil
}

// collectGenDecl extracts documented types, constants, and variables from a
// general declaration.
func collectGenDecl(root string, fset *token.FileSet, path string, decl *ast.GenDecl, pkg *packageDoc) {
	declComment := commentText(decl.Doc)
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			pos := fset.Position(s.Pos())
			comment := firstNonEmpty(commentText(s.Doc), declComment)
			pkg.Types = append(pkg.Types, typeDoc{
				Name:     s.Name.Name,
				Kind:     typeKind(s.Type),
				Comment:  comment,
				File:     cleanPath(root, path),
				Line:     pos.Line,
				Exported: ast.IsExported(s.Name.Name),
				Fields:   fieldsFromType(s.Type),
			})
		case *ast.ValueSpec:
			comment := firstNonEmpty(commentText(s.Doc), declComment)
			for index, name := range s.Names {
				pos := fset.Position(name.Pos())
				val := ""
				if index < len(s.Values) {
					val = exprString(fset, s.Values[index])
				}
				pkg.Values = append(pkg.Values, valueDoc{
					Name:     name.Name,
					Kind:     strings.ToLower(decl.Tok.String()),
					Type:     exprString(fset, s.Type),
					Value:    val,
					Comment:  comment,
					File:     cleanPath(root, path),
					Line:     pos.Line,
					Exported: ast.IsExported(name.Name),
				})
			}
		}
	}
}

// collectFuncDecl extracts a single function or method declaration.
func collectFuncDecl(root string, fset *token.FileSet, path string, decl *ast.FuncDecl) functionDoc {
	pos := fset.Position(decl.Pos())
	name := decl.Name.Name
	receiver := ""
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		receiver = exprString(fset, decl.Recv.List[0].Type)
		name = receiver + "." + name
	}
	return functionDoc{
		Name:       name,
		Receiver:   receiver,
		Signature:  signatureString(fset, decl),
		Comment:    commentText(decl.Doc),
		File:       cleanPath(root, path),
		Line:       pos.Line,
		Exported:   ast.IsExported(decl.Name.Name),
		Parameters: fieldList(fset, decl.Type.Params),
		Returns:    fieldList(fset, decl.Type.Results),
	}
}

// renderMarkdown converts the collected documentation data into Markdown.
func renderMarkdown(root string, packages []packageDoc, skipped []string, generatedAt string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# GoDocs - dokumentacja kodu projektu BAI\n\n")
	fmt.Fprintf(&b, "**Projekt:** MikuMiku Fan Hub / BAI_1IZ21B_PROJEKT  \n")
	fmt.Fprintf(&b, "**Zakres:** aktywny kod Go z katalogu głównego oraz `internal/`  \n")
	fmt.Fprintf(&b, "**Generator:** `docs/tools/generate_godocs.go` oparty o `go/parser` i AST  \n")
	fmt.Fprintf(&b, "**Data generowania:** %s  \n\n", generatedAt)
	fmt.Fprintf(&b, "Dokumentacja obejmuje również funkcje nieeksportowane, ponieważ projekt laboratoryjny wymaga opisu pełnego przepływu: router, handlery, serwisy, dostęp do bazy danych, tryb `vulnerable/secure`, pomocnicze walidatory oraz testy integracyjne.\n\n")
	if len(skipped) > 0 {
		fmt.Fprintf(&b, "## Pliki celowo pominięte\n\n")
		for _, item := range skipped {
			fmt.Fprintf(&b, "- `%s`\n", item)
		}
		fmt.Fprintf(&b, "\n")
	}
	fmt.Fprintf(&b, "## Spis treści\n\n")
	for _, pkg := range packages {
		fmt.Fprintf(&b, "- %s\n", pkg.Name)
	}
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "## Spis pakietów\n\n")
	for _, pkg := range packages {
		fmt.Fprintf(&b, "- [%s](#%s)\n", pkg.Name, anchor(pkg.Name))
	}
	fmt.Fprintf(&b, "\n")

	for _, pkg := range packages {
		renderPackage(&b, pkg)
	}
	_ = root
	return b.String()
}

// renderPackage writes one package section into the generated Markdown buffer.
func renderPackage(b *strings.Builder, pkg packageDoc) {
	fmt.Fprintf(b, "## %s\n\n", pkg.Name)
	fmt.Fprintf(b, "**Pliki:** ")
	for i, file := range pkg.Files {
		if i > 0 {
			fmt.Fprint(b, ", ")
		}
		fmt.Fprintf(b, "`%s`", file)
	}
	fmt.Fprintf(b, "\n\n")

	if len(pkg.Types) > 0 {
		fmt.Fprintf(b, "### Typy\n\n")
		for _, item := range pkg.Types {
			fmt.Fprintf(b, "#### `%s`\n\n", item.Name)
			fmt.Fprintf(b, "- **Rodzaj:** %s\n", item.Kind)
			fmt.Fprintf(b, "- **Widoczność:** %s\n", visibility(item.Exported))
			fmt.Fprintf(b, "- **Lokalizacja:** `%s:%d`\n", item.File, item.Line)
			fmt.Fprintf(b, "- **Opis:** %s\n", docOrFallback(item.Comment, "typ "+item.Name+" przechowuje dane używane przez pakiet "+pkg.Name))
			if len(item.Fields) > 0 {
				fmt.Fprintf(b, "- **Pola/wejścia struktury:**\n")
				for _, field := range item.Fields {
					fmt.Fprintf(b, "  - `%s` `%s` - pole danych struktury.\n", field.Name, field.Type)
				}
			}
			fmt.Fprintf(b, "\n")
		}
	}

	if len(pkg.Values) > 0 {
		fmt.Fprintf(b, "### Stałe i zmienne\n\n")
		for _, item := range pkg.Values {
			fmt.Fprintf(b, "#### `%s`\n\n", item.Name)
			fmt.Fprintf(b, "- **Rodzaj:** %s\n", item.Kind)
			fmt.Fprintf(b, "- **Widoczność:** %s\n", visibility(item.Exported))
			fmt.Fprintf(b, "- **Lokalizacja:** `%s:%d`\n", item.File, item.Line)
			if item.Type != "" {
				fmt.Fprintf(b, "- **Typ:** `%s`\n", item.Type)
			}
			if item.Value != "" {
				fmt.Fprintf(b, "- **Wartość/początek:** `%s`\n", truncate(item.Value, 180))
			}
			fmt.Fprintf(b, "- **Opis:** %s\n\n", docOrFallback(item.Comment, "wartość pomocnicza używana przez pakiet "+pkg.Name))
		}
	}

	if len(pkg.Functions) > 0 {
		fmt.Fprintf(b, "### Funkcje i metody\n\n")
		for _, fn := range pkg.Functions {
			fmt.Fprintf(b, "#### `%s`\n\n", fn.Name)
			fmt.Fprintf(b, "- **Widoczność:** %s\n", visibility(fn.Exported))
			fmt.Fprintf(b, "- **Lokalizacja:** `%s:%d`\n", fn.File, fn.Line)
			fmt.Fprintf(b, "- **Opis:** %s\n", docOrFallback(fn.Comment, fallbackFunctionDescription(fn)))
			fmt.Fprintf(b, "- **Sygnatura:**\n\n```go\n%s\n```\n\n", fn.Signature)
			fmt.Fprintf(b, "- **Wejścia:**\n")
			if len(fn.Parameters) == 0 {
				fmt.Fprintf(b, "  - brak parametrów wejściowych.\n")
			} else {
				for _, param := range fn.Parameters {
					fmt.Fprintf(b, "  - `%s` `%s` - parametr przekazywany do funkcji/metody.\n", param.Name, param.Type)
				}
			}
			fmt.Fprintf(b, "- **Wyjścia:**\n")
			if len(fn.Returns) == 0 {
				fmt.Fprintf(b, "  - brak wartości zwracanych; funkcja działa przez efekt uboczny albo kończy przepływ sterowania.\n")
			} else {
				for _, ret := range fn.Returns {
					fmt.Fprintf(b, "  - `%s` `%s` - wartość zwracana przez funkcję/metodę.\n", ret.Name, ret.Type)
				}
			}
			fmt.Fprintf(b, "- **Uwagi wykonawcze:** %s\n\n", executionNote(fn))
		}
	}
}

// fieldList converts an AST field list into a parameter slice for display.
func fieldList(fset *token.FileSet, fields *ast.FieldList) []parameter {
	if fields == nil {
		return nil
	}
	var out []parameter
	for _, field := range fields.List {
		typ := exprString(fset, field.Type)
		if len(field.Names) == 0 {
			out = append(out, parameter{Name: "(anonimowe)", Type: typ})
			continue
		}
		for _, name := range field.Names {
			out = append(out, parameter{Name: name.Name, Type: typ})
		}
	}
	return out
}

// fieldsFromType extracts struct fields from the supplied type expression.
func fieldsFromType(expr ast.Expr) []parameter {
	structType, ok := expr.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return nil
	}
	fset := token.NewFileSet()
	var out []parameter
	for _, field := range structType.Fields.List {
		typ := exprString(fset, field.Type)
		if len(field.Names) == 0 {
			out = append(out, parameter{Name: "(embedded)", Type: typ})
			continue
		}
		for _, name := range field.Names {
			out = append(out, parameter{Name: name.Name, Type: typ})
		}
	}
	return out
}

// signatureString renders a function signature in source form.
func signatureString(fset *token.FileSet, decl *ast.FuncDecl) string {
	var b bytes.Buffer
	copyDecl := *decl
	copyDecl.Body = nil
	if err := format.Node(&b, fset, &copyDecl); err != nil {
		return decl.Name.Name
	}
	return strings.TrimSpace(b.String())
}

// exprString formats an AST expression as source code.
func exprString(fset *token.FileSet, expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var b bytes.Buffer
	if err := format.Node(&b, fset, expr); err != nil {
		return ""
	}
	return strings.TrimSpace(b.String())
}

// commentText returns a trimmed comment block or an empty string.
func commentText(group *ast.CommentGroup) string {
	if group == nil {
		return ""
	}
	return strings.TrimSpace(group.Text())
}

// typeKind returns a short label for the underlying AST type expression.
func typeKind(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface"
	case *ast.FuncType:
		return "func type"
	default:
		return "alias/defined type"
	}
}

// fallbackFunctionDescription synthesizes a one-line description from a
// function signature when no doc comment is present.
func fallbackFunctionDescription(fn functionDoc) string {
	name := strings.TrimPrefix(fn.Name, fn.Receiver+".")
	switch {
	case strings.HasPrefix(name, "Test"):
		return "test integracyjny weryfikujący zachowanie aplikacji w scenariuszu opisanym nazwą funkcji"
	case strings.HasPrefix(name, "Page"):
		return "handler HTTP renderujący stronę UI albo fragment widoku"
	case strings.HasPrefix(name, "Post"):
		return "handler HTTP obsługujący żądanie POST lub operację modyfikującą stan"
	case strings.HasPrefix(name, "Get"):
		return "funkcja/metoda odczytująca dane z bazy albo warstwy serwisowej"
	case strings.HasPrefix(name, "Create"):
		return "funkcja/metoda tworząca rekord lub zasób aplikacji"
	case strings.HasPrefix(name, "Update"):
		return "funkcja/metoda aktualizująca istniejący rekord lub ustawienie"
	case strings.HasPrefix(name, "Delete"):
		return "funkcja/metoda usuwająca rekord lub zasób aplikacji"
	case strings.Contains(name, "Secure"):
		return "funkcja/metoda używana w zabezpieczonej ścieżce trybu secure"
	case strings.Contains(name, "Vulnerable"):
		return "funkcja/metoda używana w celowo podatnej ścieżce trybu vulnerable"
	default:
		return "funkcja pomocnicza używana przez pakiet; szczegóły wejść i wyjść wynikają z sygnatury"
	}
}

// executionNote adds a short note for functions that are meant to be run or
// are otherwise important to call order.
func executionNote(fn functionDoc) string {
	if len(fn.Returns) == 1 && fn.Returns[0].Type == "gin.HandlerFunc" {
		return "zwraca handler Gin; właściwe wejście runtime pochodzi z `*gin.Context`, parametrów URL, formularza, JSON body lub cookies."
	}
	if strings.HasPrefix(fn.Name, "Test") {
		return "uruchamiana przez `go test`; nie stanowi API produkcyjnego, ale dokumentuje oczekiwane zachowanie systemu."
	}
	if strings.Contains(strings.ToLower(fn.Name), "password") || strings.Contains(strings.ToLower(fn.Name), "csrf") || strings.Contains(strings.ToLower(fn.Name), "cookie") {
		return "funkcja jest częścią mechanizmu bezpieczeństwa albo scenariusza porównania vulnerable/secure."
	}
	return "funkcja jest wywoływana bezpośrednio z kodu projektu; walidacja błędów jest sygnalizowana przez zwracane `error`, status HTTP albo efekt w `gin.Context`."
}

// visibility converts an exported flag into a display label.
func visibility(exported bool) string {
	if exported {
		return "eksportowana"
	}
	return "wewnętrzna/nieeksportowana"
}

// docOrFallback returns the doc comment when present or the supplied fallback.
func docOrFallback(doc, fallback string) string {
	if strings.TrimSpace(doc) != "" {
		return strings.ReplaceAll(strings.TrimSpace(doc), "\n", " ")
	}
	return fallback + " _(opis wygenerowany, brak komentarza GoDoc w kodzie)._"
}

// anchor converts a heading into a Markdown anchor slug.
func anchor(text string) string {
	text = strings.ToLower(text)
	var b strings.Builder
	lastDash := false
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// firstNonEmpty returns the first non-empty string in the provided values.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// truncate shortens a string to the requested maximum length.
func truncate(value string, max int) string {
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

// cleanPath returns a repository-relative path for display in the output.
func cleanPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// fatal terminates the generator after printing the supplied error message.
func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
