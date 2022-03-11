package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var (
	outputFile = flag.String("outputFile", "route__.go", "")
	inputDir   = flag.String("inputDir", "", "")
)

func init() {
	flag.Parse()
}

func main() {
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, *inputDir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	for pkgName, astPkg := range packages {
		g := &GoFile{
			Package:      pkgName,
			PkgMapImport: map[string]*Import{},
		}
		for name, astFile := range astPkg.Files {
			dir, filename := filepath.Split(name)
			if filename == *outputFile || strings.HasSuffix(filename, "_test.go") {
				continue
			}
			g.dir = dir
			g.scan(astFile)
		}
		g.output()
	}
}

type (
	GoFile struct {
		Package      string
		PkgMapImport map[string]*Import
		Receivers    []*Receiver

		dir string
	}
	Import struct {
		Alias string
		Path  string
	}
	Receiver struct {
		Var        string
		Type       string
		HandleFunc []*HandleFunc
	}
	HandleFunc struct {
		RequestMapping *RequestMapping
		Name           string
		ReqType        string
		CtxType        string
	}
	RequestMapping struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
)

var (
	versionSuffixRegex = regexp.MustCompile(`^v\d+$`)
)

const (
	annotation = "@RequestMapping"
)

func (g *GoFile) scan(astFile *ast.File) {
	pkgMapImport := make(map[string]*Import, len(astFile.Imports))
	for _, i := range astFile.Imports {
		split := strings.Split(strings.Trim(i.Path.Value, `"`), "/")
		name := split[len(split)-1]
		if versionSuffixRegex.MatchString(name) {
			name = split[len(split)-2]
		}
		alias := ""
		if i.Name != nil {
			name = i.Name.Name
			alias = i.Name.Name
		}
		pkgMapImport[name] = &Import{
			Alias: alias,
			Path:  i.Path.Value,
		}
	}
	for _, decl := range astFile.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		l := len(funcDecl.Doc.List)
		if l <= 0 {
			continue
		}
		index := strings.Index(funcDecl.Doc.List[l-1].Text, annotation)
		if index < 0 {
			continue
		}
		requestMapping := new(RequestMapping)
		if err := json.Unmarshal([]byte(funcDecl.Doc.List[l-1].Text[index+len(annotation):]), requestMapping); err != nil {
			panic(err)
		}
		if funcDecl.Recv == nil {
			continue
		}
		receiverType := ""
		astReceiver := funcDecl.Recv.List[0]
		switch t := astReceiver.Type.(type) {
		case *ast.StarExpr:
			receiverType = "*" + t.X.(*ast.Ident).Name
		case *ast.Ident:
			receiverType = "*" + t.Name
		default:
			continue
		}
		receiverVar := strings.ToLower(receiverType[1:2])
		if len(astReceiver.Names) > 0 {
			receiverVar = astReceiver.Names[0].Name
		}
		reqType, ok := g.getParamType(pkgMapImport, funcDecl.Type.Params.List[0].Type)
		if !ok {
			continue
		}
		ctxType, ok := g.getParamType(pkgMapImport, funcDecl.Type.Params.List[1].Type)
		if !ok {
			continue
		}
		g.appendHandleFunc(receiverVar, receiverType, &HandleFunc{
			RequestMapping: requestMapping,
			Name:           funcDecl.Name.Name,
			ReqType:        reqType,
			CtxType:        ctxType,
		})
	}
}

func (g *GoFile) getParamType(pkgMapImport map[string]*Import, expr ast.Expr) (t string, ok bool) {
	starExpr, ok := expr.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	switch x := starExpr.X.(type) {
	case *ast.SelectorExpr:
		pkg := x.X.(*ast.Ident).Name
		if _, ok := g.PkgMapImport[pkg]; !ok {
			g.PkgMapImport[pkg] = pkgMapImport[pkg]
		}
		t = pkg + "." + x.Sel.Name
	case *ast.Ident:
		t = x.Name
	default:
		return "", false
	}
	return t, true
}

func (g *GoFile) appendHandleFunc(receiverVar, receiverType string, handleFunc *HandleFunc) {
	var receiver *Receiver
	for _, r := range g.Receivers {
		if r.Type == receiverType {
			receiver = r
			break
		}
	}
	if receiver == nil {
		receiver = &Receiver{
			Var:  receiverVar,
			Type: receiverType,
		}
		g.Receivers = append(g.Receivers, receiver)
	}
	receiver.HandleFunc = append(receiver.HandleFunc, handleFunc)
}

func (g *GoFile) output() {
	sort.Slice(g.Receivers, func(i, j int) bool {
		return g.Receivers[i].Type < g.Receivers[j].Type
	})
	for _, receiver := range g.Receivers {
		sort.Slice(receiver.HandleFunc, func(i, j int) bool {
			ri := receiver.HandleFunc[i].RequestMapping
			rj := receiver.HandleFunc[j].RequestMapping
			if ri.Path != rj.Path {
				return ri.Path < rj.Path
			}
			return ri.Method < rj.Method
		})
	}
	buf := new(bytes.Buffer)
	if err := tpl.Execute(buf, g); err != nil {
		panic(err)
	}
	bs, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	filename := filepath.Join(g.dir, *outputFile)
	if err = os.WriteFile(filename, bs, os.ModePerm); err != nil {
		panic(err)
	}
	log.Printf("output:%v", filename)
}

var (
	tpl = template.Must(template.New("").Parse(templateContent))
)

const (
	templateContent = `// Code generated by _http. DO NOT EDIT.

package {{.Package}}

import (
	"github.com/go-productive/_http"
    {{range $pkg, $import := .PkgMapImport}}{{$import.Alias}} {{$import.Path}}
    {{end}}
)

{{range $i, $receiver := .Receivers}}
func ({{$receiver.Var}} {{$receiver.Type}}) RegisterRoute(server *_http.Server) {
    {{range $i, $handleFunc := $receiver.HandleFunc}}
    server.Engine.{{$handleFunc.RequestMapping.Method}}("{{$handleFunc.RequestMapping.Path}}", server.GinHandlerFunc(func() interface{} { return new({{$handleFunc.ReqType}}) },
		func(req interface{}, ctx interface{}) (interface{}, error) {
			return e.{{$handleFunc.Name}}(req.(*{{$handleFunc.ReqType}}), ctx.(*{{$handleFunc.CtxType}}))
		}))
    {{end}}
}
{{end}}
`
)