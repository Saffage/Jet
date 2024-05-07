package checker

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/saffage/jet/ast"
	"github.com/saffage/jet/constant"
	"github.com/saffage/jet/types"
)

// Type checks 'expr' and returns its type.
// If error was occured, result is undefined
func (check *Checker) typeOfInternal(expr ast.Node) types.Type {
	switch node := expr.(type) {
	case nil:
		panic("got nil not for expr")

	case ast.Decl:
		panic("declaration must be handled somewhere else")

	case *ast.BadNode,
		*ast.Comment,
		*ast.CommentGroup,
		*ast.Else,
		*ast.List,
		*ast.ExprList,
		*ast.AttributeList:
		// *ast.Signature:
		panic("ill-formed AST")

	case *ast.Empty:
		return types.Unit

	case *ast.Ident:
		return check.typeOfIdent(node)

	case *ast.Literal:
		return check.typeOfLiteral(node)

	// case *ast.Operator:
	// 	panic("not implemented")

	case *ast.BuiltInCall:
		return check.typeOfBuiltInCall(node)

	case *ast.Call:
		return check.typeOfCall(node)

	case *ast.Index:
		return check.typeOfIndex(node)

	case *ast.ArrayType:
		return check.typeOfArrayType(node)

	case *ast.Signature:
		return check.typeOfSignature(node)

	case *ast.PrefixOp:
		return check.typeOfPrefixOp(node)

	case *ast.InfixOp:
		return check.typeOfInfixOp(node)

	case *ast.PostfixOp:
		return check.typeOfPostfixOp(node)

	case *ast.BracketList:
		return check.typeOfBracketList(node)

	case *ast.ParenList:
		return check.typeOfParenList(node)

	case *ast.CurlyList:
		return check.typeOfCurlyList(node)

	case *ast.If:
		return check.typeOfIf(node)

	case *ast.While:
		return check.typeOfWhile(node)

	// case *ast.Return, *ast.Break, *ast.Continue:
	// 	panic("not implemented")

	default:
		panic(fmt.Sprintf("type checking of %T is not implemented", expr))
	}
}

func (check *Checker) valueOfInternal(expr ast.Node) *TypedValue {
	switch node := expr.(type) {
	case *ast.Literal:
		value := constant.FromNode(node)
		type_ := types.FromConstant(value)
		return &TypedValue{type_, value}

		// case *ast.Ident:
		// 	panic("constants are not implemented")

		// case *ast.PrefixOp, *ast.PostfixOp, *ast.InfixOp:
		// 	panic("not implemented")
	}

	return nil
}

func (check *Checker) symbolOf(ident *ast.Ident) Symbol {
	if sym, _ := check.scope.Lookup(ident.Name); sym != nil {
		return sym
	}

	return nil
}

func (check *Checker) typeOfIdent(node *ast.Ident) types.Type {
	if sym := check.symbolOf(node); sym != nil {
		if sym.Type() != nil {
			return sym.Type()
		}

		check.errorf(node, "expression `%s` has no type", node.Name)
		return nil
	}

	check.errorf(node, "identifier `%s` is undefined", node.Name)
	return nil
}

func (check *Checker) typeOfLiteral(node *ast.Literal) types.Type {
	switch node.Kind {
	case ast.IntLiteral:
		return types.Primitives[types.UntypedInt]

	case ast.FloatLiteral:
		return types.Primitives[types.UntypedFloat]

	case ast.StringLiteral:
		return types.Primitives[types.UntypedString]

	default:
		panic(fmt.Sprintf("unhandled literal kind: '%s'", node.Kind.String()))
	}
}

func (check *Checker) typeOfBuiltInCall(node *ast.BuiltInCall) types.Type {
	builtIn := (*BuiltIn)(nil)
	idx := slices.IndexFunc(check.builtIns, func(b *BuiltIn) bool {
		return b.name == node.Name.Name
	})

	if idx != -1 {
		builtIn = check.builtIns[idx]
	}

	if builtIn == nil {
		check.errorf(node.Name, "unknown built-in function '@%s'", node.Name.Name)
		return nil
	}

	args, _ := node.Args.(*ast.ParenList)
	if args == nil {
		check.errorf(node.Args, "block as built-in function argument is not yet supported")
		return nil
	}

	tArgs := check.typeOfParenList(args)
	if tArgs == nil {
		return nil
	}

	if idx, err := builtIn.t.CheckArgs(tArgs.(*types.Tuple)); err != nil {
		n := ast.Node(args)

		if idx < len(args.Exprs) {
			n = args.Exprs[idx]
		}

		check.errorf(n, err.Error())
		return nil
	}

	value := builtIn.f(args, check.scope)
	if value == nil {
		return nil
	}

	return value.Type
}

func (check *Checker) typeOfCall(node *ast.Call) types.Type {
	tOperand := check.typeOf(node.X)
	if tOperand == nil {
		return nil
	}

	fn := types.AsFunc(tOperand)
	if fn == nil {
		check.errorf(node.X, "expression is not a function")
		return nil
	}

	tArgs := check.typeOfParenList(node.Args)
	if tArgs == nil {
		return nil
	}

	if idx, err := fn.CheckArgs(tArgs.(*types.Tuple)); err != nil {
		n := ast.Node(node.Args)

		if idx < len(node.Args.Exprs) {
			n = node.Args.Exprs[idx]
		}

		check.errorf(n, err.Error())
		return nil
	}

	return fn.Result()
}

func (check *Checker) typeOfIndex(node *ast.Index) types.Type {
	t := check.typeOf(node.X)
	if t == nil {
		return nil
	}

	if len(node.Args.Exprs) != 1 {
		check.errorf(node.Args.ExprList, "expected 1 argument")
		return nil
	}

	tIndex := check.typeOf(node.Args.Exprs[0])
	if tIndex == nil {
		return nil
	}

	if array := types.AsArray(t); array != nil {
		if !types.Primitives[types.I32].Equals(tIndex) {
			check.errorf(node.Args.Exprs[0], "expected type (i32) for index, got (%s) instead", tIndex)
			return nil
		}

		return array.ElemType()
	}

	tuple := types.AsTuple(t)
	if tuple == nil {
		check.errorf(node.X, "expression is not an array or tuple")
		return nil
	}

	// TODO use [Scope.ValueOf]
	lit, _ := node.Args.Exprs[0].(*ast.Literal)
	if lit == nil || lit.Kind != ast.IntLiteral {
		check.errorf(node.Args.Exprs[0], "expected integer literal")
		return nil
	}

	n, err := strconv.ParseInt(lit.Value, 0, 64)
	if err != nil {
		panic(err)
	}

	if n < 0 || n > int64(tuple.Len())-1 {
		check.errorf(node.Args.Exprs[0], "index must be in range 0..%d", tuple.Len()-1)
		return nil
	}

	return tuple.Types()[uint64(n)]
}

func (check *Checker) typeOfArrayType(node *ast.ArrayType) types.Type {
	if len(node.Args.Exprs) == 0 {
		check.errorf(node.Args, "slices are not implemented")
		return nil
	}

	if len(node.Args.Exprs) > 1 {
		check.errorf(node.Args, "expected 1 argument")
		return nil
	}

	value := check.valueOf(node.Args.Exprs[0])
	if value == nil {
		check.errorf(node.Args.Exprs[0], "array size cannot be infered")
		return nil
	}

	intValue := constant.AsInt(value.Value)
	if intValue == nil {
		check.errorf(node.Args.Exprs[0], "expected integer value for array size")
		return nil
	}

	if intValue.Sign() == -1 || intValue.Int64() > math.MaxInt {
		check.errorf(node.Args.Exprs[0], "size must be in range 0..9223372036854775807")
		return nil
	}

	elemType := check.typeOf(node.X)
	if elemType == nil {
		return nil
	}

	if !types.IsTypeDesc(elemType) {
		check.errorf(node.X, "expected type, got (%s)", elemType)
		return nil
	}

	size := int(intValue.Int64())
	t := types.NewArray(size, types.SkipTypeDesc(elemType))
	return types.NewTypeDesc(t)
}

func (check *Checker) typeOfSignature(node *ast.Signature) types.Type {
	tParams := check.typeOfParenList(node.Params)
	if tParams == nil {
		return nil
	}

	tResult := types.Unit

	if node.Result != nil {
		tActualResult := check.typeOf(node.Result)
		if tActualResult == nil {
			return nil
		}

		if !types.IsTypeDesc(tActualResult) {
			check.errorf(node.Result, "expected type, got (%s) instead", tActualResult)
			return nil
		}

		tResult = types.WrapInTuple(types.SkipTypeDesc(tActualResult))
	}

	t := types.NewFunc(tResult, tParams.(*types.Tuple))
	return types.NewTypeDesc(t)
}

func (check *Checker) typeOfPrefixOp(node *ast.PrefixOp) types.Type {
	tOperand := check.typeOf(node.X)
	if tOperand == nil {
		return nil
	}

	switch node.Opr.Kind {
	case ast.OperatorNeg:
		if p := types.AsPrimitive(tOperand); p != nil {
			switch p.Kind() {
			case types.UntypedInt, types.UntypedFloat, types.I32:
				return tOperand
			}
		}

		check.errorf(
			node.Opr,
			"operator '%s' is not defined for the type (%s)",
			node.Opr.Kind.String(),
			tOperand.String())
		return nil

	case ast.OperatorNot:
		if p, ok := tOperand.Underlying().(*types.Primitive); ok {
			switch p.Kind() {
			case types.UntypedBool, types.Bool:
				return tOperand
			}
		}

		check.errorf(
			node.X,
			"operator '%s' is not defined for the type (%s)",
			node.Opr.Kind.String(),
			tOperand.String())
		return nil

	case ast.OperatorAddr:
		if types.IsTypeDesc(tOperand) {
			t := types.NewRef(types.SkipTypeDesc(tOperand))
			return types.NewTypeDesc(t)
		}

		return types.NewRef(types.SkipUntyped(tOperand))

	case ast.OperatorMutAddr:
		panic("not implemented")

	default:
		panic("unreachable")
	}
}

func (check *Checker) typeOfInfixOp(node *ast.InfixOp) types.Type {
	tOperandX := check.typeOf(node.X)
	if tOperandX == nil {
		return nil
	}

	tOperandY := check.typeOf(node.Y)
	if tOperandY == nil {
		return nil
	}

	if !tOperandX.Equals(tOperandY) {
		check.errorf(node, "type mismatch (%s and %s)", tOperandX, tOperandY)
		return nil
	}

	if node.Opr.Kind == ast.OperatorAssign {
		return types.Unit
	}

	if primitive := types.AsPrimitive(tOperandX); primitive != nil {
		switch node.Opr.Kind {
		case ast.OperatorAdd, ast.OperatorSub, ast.OperatorMul, ast.OperatorDiv, ast.OperatorMod,
			ast.OperatorBitAnd, ast.OperatorBitOr, ast.OperatorBitXor, ast.OperatorBitShl, ast.OperatorBitShr:
			switch primitive.Kind() {
			case types.UntypedInt, types.UntypedFloat, types.I32:
				return tOperandX
			}

		case ast.OperatorEq, ast.OperatorNe, ast.OperatorLt, ast.OperatorLe, ast.OperatorGt, ast.OperatorGe:
			switch primitive.Kind() {
			case types.UntypedBool, types.UntypedInt, types.UntypedFloat:
				return types.Primitives[types.UntypedBool]

			case types.Bool, types.I32:
				return types.Primitives[types.Bool]
			}

		default:
			panic("unreachable")
		}
	}

	check.errorf(
		node.Opr,
		"operator '%s' is not defined for the type '%s'",
		node.Opr.Kind.String(),
		tOperandX.String())
	return nil
}

func (check *Checker) typeOfPostfixOp(node *ast.PostfixOp) types.Type {
	tOperand := check.typeOf(node.X)
	if tOperand == nil {
		return nil
	}

	switch node.Opr.Kind {
	case ast.OperatorUnwrap:
		if ref := types.AsRef(tOperand); ref != nil {
			return ref.Base()
		}

		check.errorf(node.X, "expression is not a reference type")
		return nil

	case ast.OperatorTry:
		panic("not inplemented")

	default:
		panic("unreachable")
	}
}

func (check *Checker) typeOfBracketList(node *ast.BracketList) types.Type {
	var elemType types.Type

	for _, expr := range node.Exprs {
		t := check.typeOf(expr)
		if t == nil {
			return nil
		}

		if elemType == nil {
			elemType = types.SkipUntyped(t)
			continue
		}

		if !elemType.Equals(t) {
			check.errorf(expr, "expected type (%s) for element, got (%s) instead", elemType, t)
			return nil
		}
	}

	size := len(node.Exprs)
	return types.NewArray(size, elemType)
}

func (check *Checker) typeOfParenList(node *ast.ParenList) types.Type {
	// Either typedesc or tuple contructor.

	if len(node.Exprs) == 0 {
		return types.Unit
	}

	elemTypes := []types.Type{}
	isTypeDescTuple := false

	t := check.typeOf(node.Exprs[0])
	if t == nil {
		return nil
	}

	if types.IsTypeDesc(t) {
		isTypeDescTuple = true
		elemTypes = append(elemTypes, types.SkipTypeDesc(t))
	} else {
		elemTypes = append(elemTypes, types.SkipUntyped(t))
	}

	for _, expr := range node.Exprs[1:] {
		t := check.typeOf(expr)
		if t == nil {
			return nil
		}

		if isTypeDescTuple {
			if !types.IsTypeDesc(t) {
				check.errorf(expr, "expected type, got '%s' instead", t)
				return nil
			}

			elemTypes = append(elemTypes, types.SkipTypeDesc(t))
		} else {
			if types.IsTypeDesc(t) {
				check.errorf(expr, "expected expression, got type '%s' instead", t)
				return nil
			}

			elemTypes = append(elemTypes, types.SkipUntyped(t))
		}
	}

	if isTypeDescTuple {
		return types.NewTypeDesc(types.NewTuple(elemTypes...))
	}

	return types.NewTuple(elemTypes...)
}

func (check *Checker) typeOfCurlyList(node *ast.CurlyList) types.Type {
	block := NewBlock(NewScope(check.scope))
	fmt.Printf(">>> push local\n")

	for _, node := range node.Nodes {
		ast.WalkTopDown(check.blockVisitor(block), node)
	}

	fmt.Printf(">>> pop local\n")
	return block.t
}

func (check *Checker) typeOfIf(node *ast.If) types.Type {
	// We check the body type before the condition to return the
	// body type in case the condition is not a boolean expression.
	tBody := check.typeOf(node.Body)
	if tBody == nil {
		return nil
	}

	if node.Else != nil {
		tElse := check.typeOfElse(node.Else, tBody)
		if tElse == nil {
			return tBody
		}
	}

	tCondition := check.typeOf(node.Cond)
	if tCondition == nil {
		return tBody
	}

	if !types.Primitives[types.Bool].Equals(tCondition) {
		check.errorf(
			node.Cond,
			"expected type (bool) for condition, got (%s) instead",
			tCondition)
		return tBody
	}

	return tBody
}

func (check *Checker) typeOfElse(node *ast.Else, expectedType types.Type) types.Type {
	tBody := check.typeOf(node.Body)
	if tBody == nil {
		return nil
	}

	if !expectedType.Equals(tBody) {
		// Find the last node in the body for better error message.
		lastNode := ast.Node(node.Body)

		switch body := node.Body.(type) {
		case *ast.CurlyList:
			lastNode = body.Nodes[len(body.Nodes)-1]

		case *ast.If:
			lastNode = body.Body.Nodes[len(body.Body.Nodes)-1]
		}

		check.errorf(
			lastNode,
			"all branches must have the same type with first branch (%s), got (%s) instead",
			expectedType,
			tBody)
		return nil
	}

	return tBody
}

func (check *Checker) typeOfWhile(node *ast.While) types.Type {
	tBody := check.typeOf(node.Body)
	if tBody == nil {
		return nil
	}

	if !types.Unit.Equals(tBody) {
		check.errorf(node.Body, "while loop body must have no type, but got (%s)", tBody)
		return nil
	}

	tCond := check.typeOf(node.Cond)
	if tCond == nil {
		return nil
	}

	if !types.Primitives[types.Bool].Equals(tCond) {
		check.errorf(node.Cond, "expected type 'bool' for condition, got (%s) instead", tCond)
		return nil
	}

	return types.Unit
}