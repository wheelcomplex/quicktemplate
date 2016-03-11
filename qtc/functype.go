package main

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"strings"
)

type funcType struct {
	name       string
	defPrefix  string
	callPrefix string
	argNames   string
	args       string
}

func parseFuncDef(b []byte) (*funcType, error) {
	defStr := string(b)

	// extract func name
	n := strings.Index(defStr, "(")
	if n < 0 {
		return nil, fmt.Errorf("cannot find '(' in function definition")
	}
	name := defStr[:n]
	defStr = defStr[n+1:]
	defPrefix := ""
	callPrefix := ""
	if len(name) == 0 {
		// Either empty func name or valid method definition. Let's check.

		// parse method receiver
		n = strings.Index(defStr, ")")
		if n < 0 {
			return nil, fmt.Errorf("cannot find ')' in func")
		}
		recvStr := defStr[:n]
		defStr = defStr[n+1:]
		exprStr := fmt.Sprintf("func (%s)", recvStr)
		expr, err := goparser.ParseExpr(exprStr)
		if err != nil {
			return nil, fmt.Errorf("invalid method definition: %s", err)
		}
		ft := expr.(*ast.FuncType)
		if len(ft.Params.List) != 1 || len(ft.Params.List[0].Names) != 1 {
			// method reciever must contain only one param
			return nil, fmt.Errorf("missing func or method name")
		}
		recvName := ft.Params.List[0].Names[0].Name
		if isReservedArgName(recvName) {
			return nil, fmt.Errorf("arg name %q is reserved. Use another name", recvName)
		}
		defPrefix = fmt.Sprintf("(%s) ", recvStr)
		callPrefix = recvName + "."

		// extract method name
		n = strings.Index(defStr, "(")
		if n < 0 {
			return nil, fmt.Errorf("missing func name")
		}
		name = string(stripLeadingSpace([]byte(defStr[:n])))
		if len(name) == 0 {
			return nil, fmt.Errorf("missing method name")
		}
		defStr = defStr[n+1:]
	}

	// validate and collect func args
	if len(defStr) == 0 || defStr[len(defStr)-1] != ')' {
		return nil, fmt.Errorf("missing ')' at the end of func")
	}
	args := defStr[:len(defStr)-1]
	exprStr := fmt.Sprintf("func (%s)", args)
	expr, err := goparser.ParseExpr(exprStr)
	if err != nil {
		return nil, fmt.Errorf("invalid func args: %s", err)
	}
	ft := expr.(*ast.FuncType)
	if ft.Results != nil {
		return nil, fmt.Errorf("func mustn't return any results")
	}

	// extract arg names
	var tmp []string
	for _, f := range ft.Params.List {
		if len(f.Names) == 0 {
			return nil, fmt.Errorf("func cannot contain untyped arguments")
		}
		for _, n := range f.Names {
			if n == nil {
				return nil, fmt.Errorf("func cannot contain untyped arguments")
			}
			if isReservedArgName(n.Name) {
				return nil, fmt.Errorf("arg name %q is reserved. Use other name", n.Name)
			}
			tmp = append(tmp, n.Name)
		}
	}
	argNames := strings.Join(tmp, ", ")

	if len(args) > 0 {
		args = ", " + args
	}
	if len(argNames) > 0 {
		argNames = ", " + argNames
	}
	return &funcType{
		name:       name,
		defPrefix:  defPrefix,
		callPrefix: callPrefix,
		argNames:   argNames,
		args:       args,
	}, nil
}

func parseFuncCall(b []byte) (*funcType, error) {
	exprStr := string(b)
	expr, err := goparser.ParseExpr(exprStr)
	if err != nil {
		return nil, err
	}
	ce, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("missing function call")
	}
	callPrefix, name, err := getCallName(ce)
	if err != nil {
		return nil, err
	}
	argNames := exprStr[ce.Lparen : ce.Rparen-1]

	if len(argNames) > 0 {
		argNames = ", " + argNames
	}
	return &funcType{
		name:       name,
		callPrefix: callPrefix,
		argNames:   argNames,
	}, nil
}

func (f *funcType) DefStream(dst string) string {
	return fmt.Sprintf("%sstream%s(%s *quicktemplate.Writer%s)", f.defPrefix, f.name, dst, f.args)
}

func (f *funcType) CallStream(dst string) string {
	return fmt.Sprintf("%sstream%s(%s%s)", f.callPrefix, f.name, dst, f.argNames)
}

func (f *funcType) DefWrite(dst string) string {
	return fmt.Sprintf("%s%s%s(%s io.Writer%s)", f.defPrefix, f.prefixWrite(), f.name, dst, f.args)
}

func (f *funcType) CallWrite(dst string) string {
	return fmt.Sprintf("%s%s%s(%s%s)", f.callPrefix, f.prefixWrite(), f.name, dst, f.argNames)
}

func (f *funcType) DefString() string {
	args := f.args
	if len(args) > 0 {
		// skip the first ', '
		args = args[2:]
	}
	return fmt.Sprintf("%s%s(%s) string", f.defPrefix, f.name, args)
}

func (f *funcType) prefixWrite() string {
	s := "write"
	if isUpper(f.name[0]) {
		s = "Write"
	}
	return s
}

func getCallName(ce *ast.CallExpr) (string, string, error) {
	callPrefix := ""
	name := ""
	expr := ce.Fun
	for {
		switch x := expr.(type) {
		case *ast.Ident:
			if len(callPrefix) == 0 && len(name) == 0 {
				return "", x.Name, nil
			}
			callPrefix = x.Name + "." + callPrefix
			return callPrefix, name, nil
		case *ast.SelectorExpr:
			if len(name) == 0 {
				name = x.Sel.Name
			} else {
				callPrefix = x.Sel.Name + "." + callPrefix
			}
			expr = x.X
		default:
			return "", "", fmt.Errorf("unexpected function name")
		}
	}
}

func isReservedArgName(name string) bool {
	return name == "w" || name == "qw"
}
