package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	encodeResponse(
		generateResponse(
			parseRequest(
				decodeRequest(os.Stdin),
			),
		),
		os.Stdout,
	)
}

// decodeRequest unmarshals the protobuf request.
func decodeRequest(r io.Reader) *plugin.CodeGeneratorRequest {
	var req plugin.CodeGeneratorRequest
	input, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal("unable to read stdin: " + err.Error())
	}
	if err := proto.Unmarshal(input, &req); err != nil {
		log.Fatal("unable to marshal stdin as protobuf: " + err.Error())
	}
	return &req
}

func goPackageName(d *descriptor.FileDescriptorProto) string {
	// Does the file have a "go_package" option?
	if _, pkg, ok := goPackageOption(d); ok {
		return pkg
	}

	// Does the file have a package clause?
	if pkg := d.GetPackage(); pkg != "" {
		return pkg
	}

	// Use the file base name.
	return baseName(d.GetName())
}

// goFileName returns the output name for the generated Go file.
func goFileName(d *descriptor.FileDescriptorProto) string {
	name := *d.Name
	if ext := path.Ext(name); ext == ".proto" || ext == ".protodevel" {
		name = name[:len(name)-len(ext)]
	}
	name += ".assembly.go"

	// Does the file have a "go_package" option?
	// If it does, it may override the filename.
	if impPath, _, ok := goPackageOption(d); ok && impPath != "" {
		// Replace the existing dirname with the declared import path.
		_, name = path.Split(name)
		name = path.Join(impPath, name)
		return name
	}

	return name
}

// baseName returns the last path element of the name, with the last dotted suffix removed.
func baseName(name string) string {
	// First, find the last element
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	// Now drop the suffix
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[0:i]
	}
	return name
}

// getGoPackage returns the file's go_package option.
// If it containts a semicolon, only the part before it is returned.
func getGoPackage(fd *descriptor.FileDescriptorProto) string {
	pkg := fd.GetOptions().GetGoPackage()
	if strings.Contains(pkg, ";") {
		parts := strings.Split(pkg, ";")
		if len(parts) > 2 {
			log.Fatalf(
				"protoc-gen-nrpc: go_package '%s' contains more than 1 ';'",
				pkg)
		}
		pkg = parts[0]
	}

	return pkg
}

// goPackageOption interprets the file's go_package option.
// If there is no go_package, it returns ("", "", false).
// If there's a simple name, it returns ("", pkg, true).
// If the option implies an import path, it returns (impPath, pkg, true).
func goPackageOption(d *descriptor.FileDescriptorProto) (impPath, pkg string, ok bool) {
	pkg = getGoPackage(d)
	if pkg == "" {
		return
	}
	ok = true
	// The presence of a slash implies there's an import path.
	slash := strings.LastIndex(pkg, "/")
	if slash < 0 {
		return
	}
	impPath, pkg = pkg, pkg[slash+1:]
	// A semicolon-delimited suffix overrides the package name.
	sc := strings.IndexByte(impPath, ';')
	if sc < 0 {
		return
	}
	impPath, pkg = impPath[:sc], impPath[sc+1:]
	return
}

// parseRequest wrangles the request to fit needs of the template.
func parseRequest(req *plugin.CodeGeneratorRequest) []params {
	var ps []params
	for _, pf := range req.GetProtoFile() {
		for _, svc := range pf.GetService() {

			p := params{
				ServiceDescriptorProto: *svc,
				PackageName:            pf.GetPackage(),
				ProtoName:              pf.GetName(),
				GoPackageName:          goPackageName(pf),
				GoFileName:             goFileName(pf),
			}

			for _, mtd := range p.ServiceDescriptorProto.GetMethod() {
				m := method{
					MethodDescriptorProto: *mtd,
					Name:        mtd.GetName(),
					serviceName: p.ServiceDescriptorProto.GetName(),
					packageName: p.PackageName,
				}
				p.Methods = append(p.Methods, m)
			}

			ps = append(ps, p)
		}

	}
	return ps
}

// generateResponse executes the template.
func generateResponse(ps []params) *plugin.CodeGeneratorResponse {
	var resp plugin.CodeGeneratorResponse

	for _, p := range ps {
		w := &bytes.Buffer{}
		if err := tmpl.Execute(w, p); err != nil {
			log.Fatal("unable to execute template: " + err.Error())
		}

		source := []byte(w.String())

		fmted, err := format.Source(source)
		if err != nil {
			log.Fatal("unable to go-fmt output: ,"+err.Error(), " :", string(source))
		}

		fileName := p.GoFileName
		fileContent := string(fmted)
		resp.File = append(resp.File, &plugin.CodeGeneratorResponse_File{
			Name:    &fileName,
			Content: &fileContent,
		})
	}

	return &resp
}

// encodeResponse marshals the protobuf response.
func encodeResponse(resp *plugin.CodeGeneratorResponse, w io.Writer) {
	outBytes, err := proto.Marshal(resp)
	if err != nil {
		log.Fatal("unable to marshal response to protobuf: " + err.Error())
	}

	if _, err := w.Write(outBytes); err != nil {
		log.Fatal("unable to write protobuf to stdout: " + err.Error())
	}
}

// params is the data provided to the template.
type params struct {
	descriptor.ServiceDescriptorProto
	ProtoName     string
	PackageName   string
	GoPackageName string
	Methods       []method
	fileName      string
	GoFileName    string
}

type method struct {
	Name string
	descriptor.MethodDescriptorProto
	serviceName string
	packageName string
}

// The following methods are used by the template.
func (m method) TrimmedInput() string {
	return strings.TrimPrefix(m.GetInputType(), fmt.Sprintf(".%s.", m.packageName))
}
func (m method) TrimmedOutput() string {
	return strings.TrimPrefix(m.GetOutputType(), fmt.Sprintf(".%s.", m.packageName))
}
func (m method) StreamName() string {
	return fmt.Sprintf("%s_%sServer", m.serviceName, m.GetName())
}

var tmpl = template.Must(template.New("server").Parse(`
// Code initially generated by protoc-gen-grpc-impl
package {{.GoPackageName}}

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"context"
)

{{$Type := .Name}}
{{$TypeSuffix := "Assembly"}}
{{$MethodSuffix := "Method"}}

// {{$Type}}{{$TypeSuffix}} is an implementation of the grpc-defined type, {{$Type}}.
// Its members are functions which implement the defined rpc endpoints.
type {{$Type}}{{$TypeSuffix}} struct {
	{{range .Methods}}
		{{ if .GetClientStreaming }}
			{{ if .GetServerStreaming }}
				{{.Name}}{{$MethodSuffix}} func(stream {{.StreamName}}) error 
			{{ else }}	
				 {{.Name}}{{$MethodSuffix}} func(stream {{.StreamName}}) error 
			{{ end }}
		{{ else }}
			{{ if .GetServerStreaming }}
				{{.Name}}{{$MethodSuffix}} func(input *{{.TrimmedInput}}, stream {{.StreamName}}) error

			{{ else }}
				{{.Name}}{{$MethodSuffix}} func(ctx context.Context, input *{{.TrimmedInput}}) (*{{.TrimmedOutput}}, error)
			{{end}}
		{{end}}
	{{end}}
}

{{range .Methods}}
	{{ if .GetClientStreaming }}
		{{ if .GetServerStreaming }}
			// {{.Name}} calls the provided implementation, {{.Name}}{{$MethodSuffix}}.
			func (t *{{$Type}}{{$TypeSuffix}}) {{.Name}}(stream {{.StreamName}}) error {
				return t.{{.Name}}{{$MethodSuffix}}(stream)
			}
		{{ else }}	
			// {{.Name}} calls the provided implementation, {{.Name}}{{$MethodSuffix}}.
			func (t *{{$Type}}{{$TypeSuffix}}) {{.Name}}(stream {{.StreamName}}) error {
				return t.{{.Name}}{{$MethodSuffix}}(stream)
			}
		{{ end }}
	{{ else }}
		{{ if .GetServerStreaming }}
			// {{.Name}} calls the provided implementation, {{.Name}}{{$MethodSuffix}}.
			func (t *{{$Type}}{{$TypeSuffix}}) {{.Name}}(input *{{.TrimmedInput}}, stream {{.StreamName}}) error {
				return t.{{.Name}}{{$MethodSuffix}}(input, stream)
			}
		{{ else }}
			// {{.Name}} calls the provided implementation, {{.Name}}{{$MethodSuffix}}.
			func (t *{{$Type}}{{$TypeSuffix}}) {{.Name}}(ctx context.Context, input *{{.TrimmedInput}}) (*{{.TrimmedOutput}}, error) {
				return t.{{.Name}}{{$MethodSuffix}}(ctx, input)
			}
		{{end}}
	{{end}}
{{end}}

// Register associates the implementation with a grpc server.
func (t *{{$Type}}{{$TypeSuffix}}) Register(srv *grpc.Server) {
	Register{{$Type}}Server(srv, t)
}

// New{{$Type}}{{$TypeSuffix}} creates an instance of {{$Type}} with unimplemented method stubs.
// NOTE: you should provide your own functions which implement the underlying methods.
func New{{$Type}}{{$TypeSuffix}}() *{{$Type}}{{$TypeSuffix}} {
	var t = new({{$Type}}{{$TypeSuffix}})
	{{range .Methods}}
		{{ if .GetClientStreaming }}
			{{ if .GetServerStreaming }}
				t.{{.Name}}{{$MethodSuffix}} = func(stream {{.StreamName}}) error {
					return status.Errorf(codes.Unimplemented, "{{.Name}} has not been implemented")
				}
			{{ else }}	
				 t.{{.Name}}{{$MethodSuffix}} = func(stream {{.StreamName}}) error  {
					return status.Errorf(codes.Unimplemented, "{{.Name}} has not been implemented")
				}
			{{ end }}
		{{ else }}
			{{ if .GetServerStreaming }}
				t.{{.Name}}{{$MethodSuffix}} = func(input *{{.TrimmedInput}}, stream {{.StreamName}}) error {
					return status.Errorf(codes.Unimplemented, "{{.Name}} has not been implemented")
				}

			{{ else }}
				t.{{.Name}}{{$MethodSuffix}} = func(ctx context.Context, input *{{.TrimmedInput}}) (*{{.TrimmedOutput}}, error) {
					return nil, status.Errorf(codes.Unimplemented, "{{.Name}} has not been implemented")
				}
			{{end}}
		{{end}}
	{{end}}
	return t
}
`))
