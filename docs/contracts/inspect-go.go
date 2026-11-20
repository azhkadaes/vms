// Source inventory only. Uses the Go parser; never loads configuration or a database.
package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type route struct{ Method, Path, Handler, File string }
type field struct {
	Name, Type string
	Optional   bool
}
type inventory struct {
	Routes           []route                      `json:"routes"`
	UnresolvedRoutes []string                     `json:"unresolvedRoutes"`
	Interfaces       map[string]map[string]string `json:"interfaces"`
	DTOs             map[string][]field           `json:"dtos"`
	Functions        map[string]string            `json:"functions"`
	Calls            map[string][]string          `json:"calls"`
	ErrorCatalogue   map[string]string            `json:"errorCatalogue"`
	StatusCodes      map[string]string            `json:"statusCodes"`
}

// Literal maps read for the error contract: the failure package's code -> message
// catalogue and the error middleware's status -> default code table.
var literalMaps = map[string]string{
	"pkg/helper/failure/catalogue.go:messages":             "errorCatalogue",
	"internal/server/rest/middleware/error.go:statusCodes": "statusCodes",
}

var fs = token.NewFileSet()

func source(n ast.Node) string { var b bytes.Buffer; _ = printer.Fprint(&b, fs, n); return b.String() }
func literal(e ast.Expr) (string, bool) {
	v, ok := e.(*ast.BasicLit)
	if !ok || v.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(v.Value)
	return s, err == nil
}
func receiver(f *ast.FuncDecl) string {
	if f.Recv == nil {
		return ""
	}
	return strings.TrimPrefix(source(f.Recv.List[0].Type), "*") + "."
}
func main() {
	if len(os.Args) != 2 {
		panic("usage: go run inspect-go.go BACKEND_ROOT")
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		panic(err)
	}
	out := inventory{Interfaces: map[string]map[string]string{}, DTOs: map[string][]field{}, Functions: map[string]string{}, Calls: map[string][]string{}, ErrorCatalogue: map[string]string{}, StatusCodes: map[string]string{}}
	for _, dir := range []string{"internal", filepath.Join("pkg", "dto"), filepath.Join("pkg", "helper", "failure")} {
		err = filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			return inspect(root, path, &out)
		})
		if err != nil {
			panic(err)
		}
	}
	sort.Slice(out.Routes, func(i, j int) bool {
		return out.Routes[i].Path+out.Routes[i].Method < out.Routes[j].Path+out.Routes[j].Method
	})
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
func inspect(root, path string, out *inventory) error {
	f, err := parser.ParseFile(fs, path, nil, 0)
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, path)
	rel = filepath.ToSlash(rel)
	for _, decl := range f.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gen.Specs {
				if v, ok := spec.(*ast.ValueSpec); ok {
					for i, name := range v.Names {
						if target, ok := literalMaps[rel+":"+name.Name]; ok && i < len(v.Values) {
							literalMap(v.Values[i], out.mapNamed(target))
						}
					}
				}
				t, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				key := rel + ":" + t.Name.Name
				if in, ok := t.Type.(*ast.InterfaceType); ok {
					methods := map[string]string{}
					for _, m := range in.Methods.List {
						if len(m.Names) == 0 {
							methods["@"+source(m.Type)] = source(m.Type)
							continue
						}
						methods[m.Names[0].Name] = source(m.Type)
					}
					out.Interfaces[key] = methods
				}
				if st, ok := t.Type.(*ast.StructType); ok && strings.HasPrefix(rel, "pkg/dto/") {
					fields := []field{}
					for _, m := range st.Fields.List {
						if m.Tag == nil {
							continue
						}
						tag, _ := strconv.Unquote(m.Tag.Value)
						js := reflect.StructTag(tag).Get("json")
						if js == "" || js == "-" {
							continue
						}
						parts := strings.Split(js, ",")
						fields = append(fields, field{Name: parts[0], Type: source(m.Type), Optional: strings.Contains(js, ",omitempty")})
					}
					out.DTOs[key] = fields
				}
			}
		}
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		key := rel + ":" + receiver(fn) + fn.Name.Name
		out.Functions[key] = source(fn.Type)
		calls := []string{}
		groups := map[string]string{"deps.App": "", "app": ""}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if a, ok := n.(*ast.AssignStmt); ok && len(a.Lhs) == 1 && len(a.Rhs) == 1 {
				if c, ok := a.Rhs[0].(*ast.CallExpr); ok {
					if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Group" && len(c.Args) > 0 {
						if prefix, ok := groups[source(sel.X)]; ok {
							if p, ok := literal(c.Args[0]); ok {
								groups[source(a.Lhs[0])] = prefix + p
							}
						}
					}
				}
			}
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			calls = append(calls, source(c.Fun))
			sel, ok := c.Fun.(*ast.SelectorExpr)
			if !ok || len(c.Args) < 2 {
				return true
			}
			method := strings.ToUpper(sel.Sel.Name)
			if method != "GET" && method != "POST" && method != "DELETE" && method != "PATCH" && method != "PUT" {
				return true
			}
			prefix, ok := groups[source(sel.X)]
			if !ok {
				if strings.HasPrefix(rel, "internal/server/rest/router/") {
					out.UnresolvedRoutes = append(out.UnresolvedRoutes, key+":"+source(c))
				}
				return true
			}
			p, ok := literal(c.Args[0])
			if !ok {
				out.UnresolvedRoutes = append(out.UnresolvedRoutes, key+":"+source(c))
				return true
			}
			if strings.HasPrefix(prefix+p, "/v1/") {
				out.Routes = append(out.Routes, route{method, prefix + p, source(c.Args[len(c.Args)-1]), rel})
			}
			return true
		})
		out.Calls[key] = calls
	}
	return nil
}

func (out *inventory) mapNamed(name string) map[string]string {
	if name == "statusCodes" {
		return out.StatusCodes
	}
	return out.ErrorCatalogue
}

// literalMap copies a map composite literal whose keys are string or integer
// literals and whose values are string literals. Any other entry is reported as
// "?" so a computed value cannot silently disappear from the contract check.
func literalMap(e ast.Expr, into map[string]string) {
	lit, ok := e.(*ast.CompositeLit)
	if !ok {
		into["?"] = source(e)
		return
	}
	for _, element := range lit.Elts {
		kv, ok := element.(*ast.KeyValueExpr)
		if !ok {
			into["?"] = source(element)
			continue
		}
		key, ok := kv.Key.(*ast.BasicLit)
		value, valueOK := literal(kv.Value)
		if !ok || !valueOK {
			into["?"] = source(element)
			continue
		}
		name := key.Value
		if key.Kind == token.STRING {
			name, _ = strconv.Unquote(key.Value)
		}
		into[name] = value
	}
}
