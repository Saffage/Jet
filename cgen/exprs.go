package cgen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/saffage/jet/ast"
	"github.com/saffage/jet/checker"
	"github.com/saffage/jet/constant"
	"github.com/saffage/jet/internal/report"
	"github.com/saffage/jet/types"
)

func (gen *generator) exprString(expr ast.Node) string {
	if _, isDecl := expr.(*ast.Decl); isDecl {
		return "ERROR_CGEN__EXPR_IS_DECL"
	}

	report.Debugf("expr = %s", expr.Repr())
	exprStr := ""

	switch node := expr.(type) {
	case *ast.Empty:
		return ""

	case *ast.BuiltInCall:
		exprStr = gen.BuiltInCall(node)

	case *ast.Ident:
		switch sym := gen.SymbolOf(node).(type) {
		case *checker.Var, *checker.Func:
			return gen.name(sym)

		case *checker.Const:
			return gen.constant(sym.Value())

		case nil:
			report.TaggedErrorf("cgen", "expression `%s` have no uses", expr)
			panic("")

		default:
			panic(fmt.Sprintf("idk (%T) %s", sym, sym.Node().Repr()))
		}

	case *ast.Literal:
		typedValue, ok := gen.Types[expr]
		if !ok {
			report.Warningf("literal without type '%[1]T': %[1]s", expr)
			return "ERROR_CGEN__EXPR"
		}

		if typedValue.Value != nil {
			return gen.constant(typedValue.Value)
		}

	case *ast.Dot:
		tv := gen.Types[node.X]
		if tv == nil {
			// Defined in another module?
			panic("idk")
		}

		if types.IsTypeDesc(tv.Type) {
			ty := types.SkipTypeDesc(tv.Type)

			if _enum := types.AsEnum(ty); _enum != nil {
				// if tyY := gen.TypeOf(node.Y); tyY != nil && tyY.Equals(ty) {
				// 	// Enum field.
				// }
				return gen.TypeString(_enum) + "__" + node.Y.Name
				// return "ERROR_CGEN__INVALID_ENUM_FIELD"
			} else {
				return "ERROR_CGEN__INVALID_MEMBER_ACCESS"
			}
		}

		return gen.exprString(node.X) + "." + node.Y.Name

	case *ast.Deref:
		return gen.unary(node.X, nil, ast.OperatorStar)

	// case *ast.PrefixOp:
	// 	typedValue := gen.Types[node]

	// 	if typedValue == nil {
	// 		typedValue = gen.Types[expr]
	// 	}

	// 	if typedValue == nil {
	// 		panic("cannot get a type of the expression")
	// 	}

	// 	return gen.unary(node.X, typedValue.Type, node.Opr.Kind)

	case *ast.Op:
		tv := gen.Types[expr]
		if tv == nil {
			panic("cannot get a type of the expression")
		}

		if node.Y == nil {
			if node.X == nil {
				panic("unreachable")
			}

			return gen.unary(node.Y, tv.Type, node.Kind)
		}

		if node.X == nil {
			return gen.unary(node.Y, tv.Type, node.Kind)
		}

		return gen.binary(node.X, node.Y, tv.Type, node.Kind)

	case *ast.Call:
		tv := gen.Types[node.X]
		if tv == nil {
			// Defined in another module?
			panic("idk")
		}

		if types.IsTypeDesc(tv.Type) {
			ty := types.SkipTypeDesc(tv.Type)
			tmp := gen.tempVar(ty)

			if _struct := types.AsStruct(ty); _struct != nil {
				gen.structInit(gen.name(tmp), node, _struct)
				return gen.name(tmp)
				// buf.WriteString(fmt.Sprintf("(%s){\n", gen.TypeString(_struct)))
				// gen.indent++
				// buf.WriteString(gen.structInit(_struct, node.Args.List))
				// gen.indent--
				// gen.indent(&buf)
				// buf.WriteString("}")
				// return buf.String()
			} else if _enum := types.AsEnum(ty); _enum != nil {
				panic("todo")
				// return gen.TypeString(_enum) + "__" + node.Args.Repr()
			} else {
				return "ERROR_CGEN__INVALID_MEMBER_ACCESS"
			}
		} else if fn := types.AsFunc(tv.Type); fn != nil {
			buf := strings.Builder{}
			buf.WriteString(gen.exprString(node.X))
			buf.WriteByte('(')
			for i, arg := range node.Args.Nodes {
				if i != 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(gen.exprString(arg))
			}
			if types.IsArray(fn.Result()) {
				if len(node.Args.Nodes) > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString("/*RESULT*/")
			}
			buf.WriteByte(')')
			return buf.String()
		}

	case *ast.Index:
		buf := strings.Builder{}
		buf.WriteString(gen.exprString(node.X))
		buf.WriteByte('[')

		if len(node.Args.Nodes) != 1 {
			panic("idk how to handle it")
		}

		buf.WriteString(gen.exprString(node.Args.Nodes[0]))
		buf.WriteByte(']')
		return "(" + buf.String() + ")"

	case *ast.BracketList:
		// NOTE when array is used not in assignment they
		// must be prefixes with the type.
		tv := gen.Types[expr]
		if tv == nil || !types.IsArray(tv.Type) {
			// Defined in another module?
			panic("idk")
		}
		ty := types.AsArray(types.SkipUntyped(tv.Type))
		tmp := gen.tempVar(ty)
		gen.arrayInit(gen.name(tmp), node, ty)
		return gen.name(tmp)

	case *ast.If:
		ty := gen.TypeOf(expr)
		if ty == nil {
			panic("if expression have no type")
		}

		tmpVar := gen.tempVar(types.SkipUntyped(ty))
		gen.ifExpr(node, tmpVar)
		return gen.name(tmpVar)

	case *ast.CurlyList:
		ty := gen.TypeOf(expr)
		if ty == nil {
			panic("if expression have no type")
		}
		tmpVar := gen.tempVar(types.SkipUntyped(ty))
		gen.block(node.StmtList, tmpVar)
		return gen.name(tmpVar)

	default:
		fmt.Printf("not implemented '%T'\n", node)
	}

	if exprStr == "" {
		report.Warningf("empty expr at node '%T'", expr)
		return "ERROR_CGEN__EXPR"
	}

	return exprStr
}

func (gen *generator) unary(x ast.Node, _ types.Type, op ast.OperatorKind) string {
	switch op {
	case ast.OperatorAddrOf:
		return fmt.Sprintf("(&%s)", gen.exprString(x))

	case ast.OperatorStar:
		return fmt.Sprintf("(*%s)", gen.exprString(x))

	case ast.OperatorNot:
		return fmt.Sprintf("(!%s)", gen.exprString(x))

	case ast.OperatorNeg:
		return fmt.Sprintf("(-%s)", gen.exprString(x))

	default:
		panic(fmt.Sprintf("not a unary operator: '%s'", op))
	}
}

func (gen *generator) binary(x, y ast.Node, t types.Type, op ast.OperatorKind) string {
	t = types.SkipUntyped(t)

	switch op {
	case ast.OperatorBitAnd,
		ast.OperatorBitOr,
		ast.OperatorBitXor,
		ast.OperatorBitShl,
		ast.OperatorBitShr:
		return fmt.Sprintf("(%[3]s)((%[1]s) %[4]s (%[2]s))",
			gen.exprString(x),
			gen.exprString(y),
			gen.TypeString(t),
			op,
		)

	case ast.OperatorAdd,
		ast.OperatorSub,
		ast.OperatorMul,
		ast.OperatorDiv,
		ast.OperatorMod:
		return fmt.Sprintf("((%[3]s)(%[1]s) %[4]s (%[3]s)(%[2]s))",
			gen.exprString(x),
			gen.exprString(y),
			gen.TypeString(t),
			op,
		)

	case ast.OperatorEq,
		ast.OperatorNe,
		ast.OperatorGt,
		ast.OperatorGe,
		ast.OperatorLt,
		ast.OperatorLe:
		return fmt.Sprintf("((%[1]s) %[3]s (%[2]s))",
			gen.exprString(x),
			gen.exprString(y),
			op,
		)

	case ast.OperatorAnd:
		return fmt.Sprintf("((%[3]s)(%[1]s) && (%[3]s)(%[2]s))",
			gen.exprString(x),
			gen.exprString(y),
			gen.TypeString(t),
		)

	case ast.OperatorOr:
		return fmt.Sprintf("((%[3]s)(%[1]s) || (%[3]s)(%[2]s))",
			gen.exprString(x),
			gen.exprString(y),
			gen.TypeString(t),
		)

	case ast.OperatorAs:
		return fmt.Sprintf("((%s)%s)",
			gen.TypeString(t),
			gen.exprString(x),
		)

	case ast.OperatorAssign:
		ty := gen.Module.TypeOf(y)

		if array := types.AsArray(ty); array != nil {
			gen.arrayAssign(gen.exprString(x), y, array)
			return ""
		}

		return fmt.Sprintf("%s = %s",
			gen.exprString(x),
			gen.exprString(y),
		)

	case ast.OperatorAddAndAssign:
		return fmt.Sprintf("%s += %s",
			gen.exprString(x),
			gen.exprString(y),
		)

	case ast.OperatorSubAndAssign:
		return fmt.Sprintf("%s -= %s",
			gen.exprString(x),
			gen.exprString(y),
		)

	case ast.OperatorMultAndAssign:
		return fmt.Sprintf("%s *= %s",
			gen.exprString(x),
			gen.exprString(y),
		)

	case ast.OperatorDivAndAssign:
		return fmt.Sprintf("%s /= %s",
			gen.exprString(x),
			gen.exprString(y),
		)

	case ast.OperatorModAndAssign:
		return fmt.Sprintf("%s %%= %s",
			gen.exprString(x),
			gen.exprString(y),
		)

	default:
		panic(fmt.Sprintf("not a binary operator: '%s'", op))
	}
}

func (gen *generator) assign(dest string, value ast.Node) {
	tv := gen.Types[value]
	if tv == nil {
		panic("cannot get a type of node")
	}
	switch ty := tv.Type.(type) {
	case *types.Array:
		gen.arrayAssign(dest, value, ty)

	case *types.Struct:
		gen.structAssign(dest, value, ty)

	default:
		gen.linef("%s = %s;\n", dest, gen.exprString(value))
	}
}

func (gen *generator) constant(value constant.Value) string {
	if value == nil {
		panic("nil constant value")
	}

	switch value.Kind() {
	case constant.Bool:
		if *constant.AsBool(value) {
			return "true"
		}
		return "false"

	case constant.Int:
		return (*constant.AsInt(value)).String()

	case constant.Float:
		return (*constant.AsFloat(value)).String()

	case constant.String:
		value := constant.AsString(value)
		return strconv.Quote(*value)

	default:
		panic("unreachable")
	}

	return "ERROR_CGEN__CONSTANT"
}

func (gen *generator) ifExpr(node *ast.If, result *checker.Var) {
	gen.linef("if (%s)\n", gen.exprString(node.Cond))
	gen.block(node.Body.StmtList, result)

	if node.Else != nil {
		gen.elseExpr(node.Else, result)
	}
}

func (gen *generator) elseExpr(node *ast.Else, result *checker.Var) {
	switch body := node.Body.(type) {
	case *ast.If:
		gen.linef("else if (%s)\n", gen.exprString(body.Cond))
		gen.block(body.Body.StmtList, result)

		if body.Else != nil {
			gen.elseExpr(body.Else, result)
		}

	case *ast.CurlyList:
		gen.line("else\n")
		gen.block(body.StmtList, result)

	default:
		panic("unreachable")
	}
}
