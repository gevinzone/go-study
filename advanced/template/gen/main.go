package main

import (
	"bytes"
	"errors"
	"fmt"
	"gitee.com/geektime-geekbang/geektime-go/advanced/template/gen/annotation"
	"gitee.com/geektime-geekbang/geektime-go/advanced/template/gen/http"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// 实际上 main 函数这里要考虑接收参数
// src 源目标
// dst 目标目录
// type src 里面可能有很多类型，那么用户可能需要指定具体的类型
// 这里我们简化操作，只读取当前目录下的数据，并且扫描下面的所有源文件，然后生成代码
// 在当前目录下运行 go install 就将 main 安装成功了，
// 可以在命令行中运行 gen
// 在 testdata 里面运行 gen，则会生成能够通过所有测试的代码
func main() {
	err := gen(".")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("success")
}

func gen(src string) error {
	// 第一步找出符合条件的文件
	srcFiles, err := scanFiles(src)
	if err != nil {
		return err
	}
	// 第二步，AST 解析源代码文件，拿到 service definition 定义
	defs, err := parseFiles(srcFiles)
	if err != nil {
		return err
	}
	// 生成代码
	return genFiles(src, defs)
}

// 根据 defs 来生成代码
// src 是源代码所在目录，在测试里面它是 ./testdata
func genFiles(src string, defs []http.ServiceDefinition) error {
	for _, def := range defs {
		bs := &bytes.Buffer{}
		err := http.Gen(bs, def)
		if err != nil {
			return err
		}
		fileName := underscoreName(def.Name) + "_gen.go"
		filePath := src + "/" + fileName
		err = os.WriteFile(filePath, bs.Bytes(), os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseFiles(srcFiles []string) ([]http.ServiceDefinition, error) {
	defs := make([]http.ServiceDefinition, 0, 20)
	for _, src := range srcFiles {
		fmt.Println(src)
		// 你需要利用 annotation 里面的东西来扫描 src，然后生成 file
		//panic("implement me")
		fSet := token.NewFileSet()
		f, err := parser.ParseFile(fSet, src, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		rootVisitor := &annotation.SingleFileEntryVisitor{}
		ast.Walk(rootVisitor, f)
		file := rootVisitor.Get()

		//var file annotation.File

		for _, typ := range file.Types {
			_, ok := typ.Annotations.Get("HttpClient")
			if !ok {
				continue
			}
			def, err := parseServiceDefinition(file.Node.Name.Name, typ)
			if err != nil {
				return nil, err
			}
			defs = append(defs, def)
		}
	}
	return defs, nil
}

// 你需要利用 typ 来构造一个 http.ServiceDefinition
// 注意你可能需要检测用户的定义是否符合你的预期
func parseServiceDefinition(pkg string, typ annotation.Type) (http.ServiceDefinition, error) {
	serviceName, err := getServiceName(typ)
	if err != nil {
		return http.ServiceDefinition{}, err
	}
	methods, err := getMethods(typ)
	if err != nil {
		return http.ServiceDefinition{}, err
	}

	def := http.ServiceDefinition{
		Package: pkg,
		Name:    serviceName,
		Methods: methods,
	}
	return def, nil
}

func getServiceName(typ annotation.Type) (string, error) {
	serviceName := typ.Annotations.Node.Name.Name
	for _, ann := range typ.Ans {
		if ann.Key == "ServiceName" {
			serviceName = ann.Value
			break
		}
	}
	return serviceName, nil
}

func getMethods(typ annotation.Type) ([]http.ServiceMethod, error) {
	methods := make([]http.ServiceMethod, 0, len(typ.Fields))
	for i := 0; i < len(typ.Fields); i++ {
		fd := typ.Fields[i]
		reqTypeName, err := getReqTypeName(fd)
		if err != nil {
			//continue
			return nil, err
		}
		respTypeName, err := getRespTypeName(fd)
		if err != nil {
			//continue
			return nil, err
		}
		path, err := getMethodPath(fd)
		if err != nil {
			return nil, err
		}
		method := http.ServiceMethod{
			Name:         fd.Node.Names[0].Name,
			Path:         path,
			ReqTypeName:  reqTypeName,
			RespTypeName: respTypeName,
		}
		methods = append(methods, method)
	}
	return methods, nil
}

func getMethodPath(field annotation.Field) (string, error) {
	for _, ann := range field.Annotations.Ans {
		if ann.Key == "Path" {
			return ann.Value, nil
		}
	}
	path := "/" + field.Node.Names[0].Name
	return path, nil
}

func getReqTypeName(field annotation.Field) (string, error) {
	err := errors.New("gen: 方法必须接收两个参数，其中第一个参数是 context.Context，第二个参数请求")
	list := field.Node.Type.(*ast.FuncType).Params.List
	if len(list) != 2 || list[1].Type == nil {
		return "", err
	}
	reqTypeName := list[1].Type.(*ast.StarExpr).X.(*ast.Ident).Name
	return reqTypeName, nil
}

func getRespTypeName(field annotation.Field) (string, error) {
	err := errors.New("gen: 方法必须返回两个参数，其中第一个返回值是响应，第二个返回值是error")
	list := field.Node.Type.(*ast.FuncType).Results.List
	if len(list) != 2 || list[0].Type == nil {
		return "", err
	}
	reqTypeName := list[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
	return reqTypeName, nil
}

// 返回符合条件的 Go 源代码文件，也就是你要用 AST 来分析这些文件的代码
func scanFiles(src string) ([]string, error) {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(file.Name(), "_test.go") || strings.HasSuffix(file.Name(), "gen.go") {
			continue
		}
		fileFullPath, _ := filepath.Abs("./testdata/" + file.Name())
		res = append(res, fileFullPath)
	}
	return res, nil
}

// underscoreName 驼峰转字符串命名，在决定生成的文件名的时候需要这个方法
// 可以用正则表达式，然而我写不出来，我是正则渣
func underscoreName(name string) string {
	var buf []byte
	for i, v := range name {
		if unicode.IsUpper(v) {
			if i != 0 {
				buf = append(buf, '_')
			}
			buf = append(buf, byte(unicode.ToLower(v)))
		} else {
			buf = append(buf, byte(v))
		}

	}
	return string(buf)
}
