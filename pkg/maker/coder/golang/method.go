package golang

import (
	"fmt"
	"github.com/dop251/goja/ast"
	"github.com/team-ide/go-tool/util"
	"go.uber.org/zap"
	"reflect"
	"strings"
	"teamide/pkg/maker"
)

type MethodBuilder struct {
	*ClassBuilder
	*maker.CompilerMethod
	inIfTest              int
	inElseIf              int
	lastReturnRowNumber   int
	inVarScript           int
	inArgument            int
	inArgumentStruct      int
	inAssign              int
	inExpressionStatement int
}

func (this_ *MethodBuilder) Gen() (err error) {
	key := this_.GetKey()

	util.Logger.Debug("gen " + key + " start")

	methodName := util.FirstToUpper(this_.Method)
	this_.AppendTab()
	this_.AppendCode("// " + methodName + " ")
	this_.AppendComment(this_.Comment)
	this_.NewLine()
	var str string
	str += "func (this_ *" + this_.GetImplClassName() + ") " + methodName
	str += "("
	for i, param := range this_.ParamList {

		var typeS string
		typeS, err = this_.GetTypeStr(param.CompilerValueType.GetValueType())
		if err != nil {
			return
		}
		if i > 0 {
			str += ", "
		}

		str += param.Name + " " + typeS
	}
	str += ")"
	str += " ("
	if this_.Result.GetValueType() != nil {
		var typeS string
		typeS, err = this_.GetTypeStr(this_.Result.GetValueType())
		if err != nil {
			return
		}
		str += "res " + typeS
		str += ", "
	}
	str += "err error"
	str += ")"

	str += " { "
	this_.AppendTabLine(str)
	this_.Tab()

	err = this_.Statements(this_.Program.Body[0].(*ast.ExpressionStatement).Expression.(*ast.CallExpression).Callee.(*ast.FunctionLiteral).Body.List)
	if err != nil {
		return
	}

	if this_.lastReturnRowNumber != this_.GetRowNumber() {
		this_.AppendTabLine("return")
	}

	this_.Indent()
	this_.AppendTabLine("}")
	this_.NewLine()

	util.Logger.Debug("gen " + key + " end")

	return
}

func (this_ *MethodBuilder) Statements(statements []ast.Statement) (err error) {
	for _, statement := range statements {
		err = this_.Statement(statement)
		if err != nil {
			return
		}
	}
	return
}

func (this_ *MethodBuilder) Statement(statement ast.Statement) (err error) {
	if statement == nil {
		return
	}

	switch s := statement.(type) {
	case *ast.ExpressionStatement:
		err = this_.ExpressionStatement(s)
		break
	case *ast.IfStatement:
		err = this_.IfStatement(s)
		break
	case *ast.VariableStatement:
		err = this_.VariableStatement(s)
		break
	case *ast.BlockStatement:
		err = this_.BlockStatement(s)
		break
	case *ast.ThrowStatement:
		err = this_.ThrowStatement(s)
		break
	case *ast.ReturnStatement:
		err = this_.ReturnStatement(s)
		break
	default:
		err = this_.Error("statement ["+reflect.TypeOf(statement).String()+"] 不支持", statement)
		util.Logger.Error(this_.GetKey()+" Statement error", zap.Error(err))
		break
	}
	return
}

func (this_ *MethodBuilder) VariableDeclarations(variableDeclarations []*ast.VariableDeclaration) (err error) {

	for _, variableDeclaration := range variableDeclarations {
		err = this_.VariableDeclaration(variableDeclaration)
		if err != nil {
			return
		}
	}
	return
}

func (this_ *MethodBuilder) FunctionLiteral(expression *ast.FunctionLiteral) (err error) {
	err = this_.ParameterList(expression.ParameterList)
	if err != nil {
		return
	}
	err = this_.VariableDeclarations(expression.DeclarationList)
	if err != nil {
		return
	}
	err = this_.BlockStatement(expression.Body)
	if err != nil {
		return
	}
	return
}

func (this_ *MethodBuilder) ParameterList(parameterList *ast.ParameterList) (err error) {
	fmt.Println("TODO ParameterList:", util.GetStringValue(parameterList))
	return
}

func (this_ *MethodBuilder) BlockStatement(statement *ast.BlockStatement) (err error) {
	err = this_.Statements(statement.List)
	if err != nil {
		return
	}
	return
}

func (this_ *MethodBuilder) VariableStatement(statement *ast.VariableStatement) (err error) {
	err = this_.Bindings(statement.List)
	if err != nil {
		return
	}
	return
}

func (this_ *MethodBuilder) VariableDeclaration(variableDeclaration *ast.VariableDeclaration) (err error) {
	fmt.Println("TODO VariableDeclaration:", util.GetStringValue(variableDeclaration))
	fmt.Println("TODO VariableDeclaration code:", this_.GetNodeCode(variableDeclaration))
	return
}

func (this_ *MethodBuilder) Bindings(bindings []*ast.Binding) (err error) {
	for _, binding := range bindings {
		err = this_.Binding(binding)
		if err != nil {
			return
		}
	}
	return
}

func (this_ *MethodBuilder) Binding(binding *ast.Binding) (err error) {

	var hasError bool
	var hasInitializer bool
	if binding.Initializer != nil {
		hasInitializer = true
		if a, ok := binding.Initializer.(*ast.CallExpression); ok {
			hasError = this_.getMethodHasError(this_.CallCache[a])
		}
	}

	this_.AppendTab()
	this_.AppendCode("var ")

	this_.inVarScript = 1
	err = this_.Expression(binding.Target)
	this_.inVarScript = 0
	if err != nil {
		return
	}

	methodVar := this_.BindingCache[binding]
	valueType := methodVar.CompilerValueType.GetValueType()
	var typeS string
	typeS, err = this_.GetTypeStr(valueType)
	if err != nil {
		return
	}

	if !hasInitializer || hasError {
		this_.AppendCode(" " + typeS)
		this_.NewLine()
	}

	if hasInitializer {
		if hasError {
			this_.AppendTab()
			this_.inVarScript = 1
			err = this_.Expression(binding.Target)
			this_.inVarScript = 0

			this_.AppendCode(", err")
		}
		this_.AppendCode(" = ")
		err = this_.Expression(binding.Initializer)
		if err != nil {
			return
		}
		this_.NewLine()

		if hasError {
			this_.AppendTabLine("if err != nil {")
			this_.Tab()
			this_.AppendTabLine("return")
			this_.Indent()
			this_.AppendTabLine("}")
		}
	} else {
		if valueType == maker.ValueTypeMap || valueType.Struct != nil {
			this_.AppendTab()
			this_.inVarScript = 1
			err = this_.Expression(binding.Target)
			this_.inVarScript = 0
			this_.AppendCode(" = ")
			if valueType == maker.ValueTypeMap {
				this_.AppendCode("make(map[string]any)")
			} else if valueType.Struct != nil {
				this_.AppendCode("&" + strings.TrimPrefix(typeS, "*") + "{}")
			}
			this_.NewLine()
		}
	}

	return
}
func (this_ *MethodBuilder) ExpressionStatement(statement *ast.ExpressionStatement) (err error) {
	this_.AppendTab()
	this_.inExpressionStatement = 1
	err = this_.Expression(statement.Expression)
	this_.inExpressionStatement = 0
	this_.NewLine()
	return
}
func (this_ *MethodBuilder) ThrowStatement(statement *ast.ThrowStatement) (err error) {
	this_.AppendTab()
	this_.AppendCode("err = ")
	this_.inVarScript = 1
	err = this_.Expression(statement.Argument)
	this_.inVarScript = 0
	if err != nil {
		return
	}
	this_.NewLine()
	this_.AppendTabLine("return")
	this_.lastReturnRowNumber = this_.GetRowNumber()
	return
}

func (this_ *MethodBuilder) IfStatement(statement *ast.IfStatement) (err error) {

	if this_.inElseIf > 0 {
		this_.inElseIf--
		this_.AppendCode(" else if ")
	} else {
		this_.AppendTab()
		this_.AppendCode("if ")
	}
	this_.inIfTest++
	err = this_.Expression(statement.Test)
	this_.inIfTest--
	if err != nil {
		return
	}
	this_.AppendCode(" { ")
	this_.NewLine()
	this_.Tab()
	//fmt.Println("IfStatement Consequent:", reflect.ValueOf(statement.Consequent).String(), this_.GetNodeCode(statement.Consequent))
	err = this_.Statement(statement.Consequent)
	if err != nil {
		return
	}
	this_.Indent()
	this_.AppendTab()
	this_.AppendCode("}")
	if statement.Alternate != nil {
		if _, ok := statement.Alternate.(*ast.IfStatement); ok {
			this_.inElseIf++
			err = this_.Statement(statement.Alternate)
			if err != nil {
				return
			}
		} else {
			this_.AppendCode(" else { ")
			this_.NewLine()
			this_.Tab()
			err = this_.Statement(statement.Alternate)
			if err != nil {
				return
			}
			this_.Indent()
			this_.AppendTab()
			this_.AppendCode("}")
		}
	}
	this_.NewLine()
	return
}

func (this_ *MethodBuilder) ReturnStatement(statement *ast.ReturnStatement) (err error) {
	if statement.Argument != nil {
		this_.AppendTab()
		var hasError bool
		if a, ok := statement.Argument.(*ast.CallExpression); ok {
			hasError = this_.getMethodHasError(this_.CallCache[a])
		}
		if hasError {
			this_.AppendCode("res, err = ")
		} else {
			this_.AppendCode("res = ")
		}
		this_.inVarScript = 1
		err = this_.Expression(statement.Argument)
		this_.inVarScript = 0
		if err != nil {
			return
		}
		this_.NewLine()
	}
	this_.AppendTabLine("return")
	this_.lastReturnRowNumber = this_.GetRowNumber()
	return
}

func (this_ *MethodBuilder) Expression(expression ast.Expression) (err error) {
	if expression == nil {
		return
	}
	switch e := expression.(type) {
	case *ast.CallExpression:
		err = this_.CallExpression(e)
		break
	case *ast.FunctionLiteral:
		err = this_.FunctionLiteral(e)
		break
	case *ast.AssignExpression:
		err = this_.AssignExpression(e)
		break
	case *ast.BinaryExpression:
		err = this_.BinaryExpression(e)
		break
	case *ast.Identifier:
		err = this_.Identifier(e)
		break
	case *ast.NumberLiteral:
		err = this_.NumberLiteral(e)
		break
	case *ast.NullLiteral:
		err = this_.NullLiteral(e)
		break
	case *ast.StringLiteral:
		err = this_.StringLiteral(e)
		break
	case *ast.DotExpression:
		err = this_.DotExpression(e)
		break
	case *ast.BracketExpression:
		err = this_.BracketExpression(e)
		break
	case *ast.TemplateLiteral:
		err = this_.TemplateLiteral(e)
		break
	case *ast.ObjectLiteral:
		err = this_.ObjectLiteral(e)
		break
	default:
		err = this_.Error("expression ["+reflect.TypeOf(expression).String()+"] 不支持", expression)
		util.Logger.Error(this_.GetKey()+" Expression error", zap.Error(err))
		break

	}
	return
}

func (this_ *MethodBuilder) ArgumentList(method any, argumentList []ast.Expression) (err error) {

	for i, one := range argumentList {
		if i > 0 {
			this_.AppendCode(", ")
		}

		this_.inArgument = 1
		this_.inVarScript = 1
		err = this_.Expression(one)
		this_.inVarScript = 0
		this_.inArgument = 0
		if err != nil {
			return
		}
	}

	return
}

func (this_ *MethodBuilder) formatMethod(name string, obj interface{}) (script string) {
	names := strings.Split(name, ".")
	script = name
	if len(names) < 2 {
		return
	}

	switch toB := obj.(type) {
	case *maker.CompilerMethod:
		place := this_.getPackBuilder(toB.CompilerPack)
		class := this_.getClassBuilder(toB.CompilerClass)
		if this_.spacePack == place.spacePack {
			script = "this_"
		} else {
			script = place.spacePack
			script += "." + class.GetClassBeanName()
		}
		script += "." + util.FirstToUpper(names[len(names)-1])
		break
	default:
		script = this_.formatAssign(name)

	}
	return
}

func (this_ *MethodBuilder) formatAssign(name string) (script string) {
	names := strings.Split(name, ".")
	script = name
	if len(names) < 2 {
		return
	}
	script = names[0]
	_, asName := this_.GetImportAsName(names[0])
	if asName != "" {
		script = asName
	}

	for i := 1; i < len(names); i++ {
		script += "." + util.FirstToUpper(names[i])
	}

	return
}

func (this_ *MethodBuilder) getMethodHasError(obj any) (hasError bool) {
	if obj == nil {
		return
	}

	switch toB := obj.(type) {
	case *maker.CompilerMethod:
		hasError = true
		break
	case *maker.ComponentMethod:
		hasError = toB.HasError
		break
	default:

	}
	return
}

func (this_ *MethodBuilder) getMethodHasReturn(obj any) (hasReturn bool) {
	if obj == nil {
		return
	}

	switch toB := obj.(type) {
	case *maker.CompilerMethod:
		hasReturn = toB.Result.GetValueType() != nil
		break
	case *maker.ComponentMethod:
		hasReturn = toB.HasReturn
		break
	default:

	}
	return
}
func (this_ *MethodBuilder) CallExpression(expression *ast.CallExpression) (err error) {

	obj := this_.CallCache[expression]
	script := this_.CallScriptCache[expression]
	script = this_.formatMethod(script, obj)

	var hasError bool
	if this_.inAssign == 0 && this_.inExpressionStatement == 1 {
		hasError = this_.getMethodHasError(obj)
		if hasError {
			if this_.getMethodHasReturn(obj) {
				this_.AppendCode("_, ")
			}
			this_.AppendCode("err = ")
		}
	}
	this_.AppendCode(script)
	//err = this_.Expression(expression.Callee)
	//if err != nil {
	//	return
	//}
	this_.AppendCode("(")
	err = this_.ArgumentList(obj, expression.ArgumentList)
	if err != nil {
		return
	}
	this_.AppendCode(")")

	if hasError {
		this_.NewLine()
		this_.AppendTabLine("if err != nil {")
		this_.Tab()
		this_.AppendTabLine("return")
		this_.Indent()
		this_.AppendTabLine("}")
	}

	return
}

func (this_ *MethodBuilder) AssignExpression(expression *ast.AssignExpression) (err error) {
	script := this_.AssignExpressionScriptCache[expression]
	varType := this_.AssignExpressionScriptTypeCache[expression]
	this_.inAssign = 1
	script = this_.formatAssign(script)
	var hasError bool
	if a, ok := expression.Right.(*ast.CallExpression); ok {
		hasError = this_.getMethodHasError(this_.CallCache[a])
	}

	this_.AppendCode(script)
	if hasError {
		this_.AppendCode(", err")
	}
	this_.AppendCode(" " + expression.Operator.String() + " ")

	if _, ok := expression.Right.(*ast.NullLiteral); ok {
		valueType := varType.GetValueType()
		if valueType.IsNumber {
			this_.AppendCode("0")
		} else if valueType == maker.ValueTypeString {
			this_.AppendCode("\"\"")
		}

	} else {
		this_.inVarScript = 1
		err = this_.Expression(expression.Right)
		this_.inVarScript = 0
		if err != nil {
			return
		}
	}

	this_.inAssign = 0

	if hasError {
		this_.NewLine()
		this_.AppendTabLine("if err != nil {")
		this_.Tab()
		this_.AppendTabLine("return")
		this_.Indent()
		this_.AppendTabLine("}")
	}

	return
}

func (this_ *MethodBuilder) BinaryExpression(expression *ast.BinaryExpression) (err error) {
	this_.inVarScript = 1
	err = this_.Expression(expression.Left)
	this_.inVarScript = 0
	if err != nil {
		return
	}
	this_.AppendCode(" " + expression.Operator.String() + " ")
	this_.inVarScript = 1
	err = this_.Expression(expression.Right)
	this_.inVarScript = 0
	if err != nil {
		return
	}
	return
}

func (this_ *MethodBuilder) Identifier(expression *ast.Identifier) (err error) {
	name := expression.Name.String()
	if this_.inVarScript > 0 {
		if this_.inVarScript == 1 {
			_, asName := this_.GetImportAsName(name)
			if asName != "" {
				name = asName
			}
		} else {
			name = util.FirstToUpper(name)
		}
	}
	if this_.inArgument > 0 {
		if this_.inArgument == 1 {
			this_.inArgument++
			this_.inArgumentStruct = 0
			if name == this_.golang.GetStructPack() {
				this_.inArgumentStruct = 1
				name = "&" + name
			} else {
				_, asName := this_.GetImportAsName(name)
				if asName != "" {
					name = asName
				}
			}
		} else {
			if this_.inArgumentStruct == 1 {
				this_.inArgumentStruct = 0
				name = name + "{}"
			}
		}
	}
	this_.AppendCode(name)
	this_.inVarScript++
	return
}

func (this_ *MethodBuilder) NumberLiteral(expression *ast.NumberLiteral) (err error) {
	this_.AppendCode(util.GetStringValue(expression.Value))
	return
}

func (this_ *MethodBuilder) NullLiteral(expression *ast.NullLiteral) (err error) {
	this_.AppendCode("nil")
	return
}

func (this_ *MethodBuilder) StringLiteral(expression *ast.StringLiteral) (err error) {
	this_.AppendCode(`"`, expression.Value.String(), `"`)
	return
}

func (this_ *MethodBuilder) TemplateLiteral(expression *ast.TemplateLiteral) (err error) {

	var str = ""
	for i, element := range expression.Elements {
		str += element.Literal
		if i < len(expression.Elements)-1 {
			str += "%v"
		}
	}
	this_.AppendCode("fmt.Sprintf(`", str, "`")
	for _, e := range expression.Expressions {
		this_.AppendCode(`, `)
		err = this_.Expression(e)
		if err != nil {
			return
		}
	}
	this_.AppendCode(")")
	return
}
func (this_ *MethodBuilder) DotExpression(expression *ast.DotExpression) (err error) {
	err = this_.Expression(expression.Left)
	if err != nil {
		return
	}
	this_.AppendCode(".")
	err = this_.Identifier(&expression.Identifier)
	if err != nil {
		return
	}

	return
}
func (this_ *MethodBuilder) ObjectLiteral(expression *ast.ObjectLiteral) (err error) {

	if len(expression.Value) == 0 {
		this_.AppendCode("map[string]any{}")
	} else {
		this_.AppendCode("map[string]any{")
		for i, v := range expression.Value {
			if i > 0 {
				this_.AppendCode(", ")
			}
			err = this_.Property(v)
			if err != nil {
				return
			}
		}
		this_.AppendCode("}")
	}

	return
}

func (this_ *MethodBuilder) Property(property ast.Property) (err error) {
	if property == nil {
		return
	}
	switch e := property.(type) {
	case *ast.PropertyKeyed:
		err = this_.PropertyKeyed(e)
		break
	default:
		err = this_.Error("property ["+reflect.TypeOf(property).String()+"] 不支持", property)
		util.Logger.Error(this_.GetKey()+" Property error", zap.Error(err))
		break

	}
	return
}

func (this_ *MethodBuilder) PropertyKeyed(property *ast.PropertyKeyed) (err error) {
	err = this_.Expression(property.Key)
	if err != nil {
		return
	}
	this_.AppendCode(" : ")
	fmt.Println("PropertyKeyed Value", reflect.TypeOf(property.Value).String())
	err = this_.Expression(property.Value)
	if err != nil {
		return
	}
	return
}

func (this_ *MethodBuilder) BracketExpression(expression *ast.BracketExpression) (err error) {
	err = this_.Expression(expression.Left)
	if err != nil {
		return
	}
	this_.AppendCode("[")
	oldInVarScript := this_.inVarScript
	this_.inVarScript = 1
	err = this_.Expression(expression.Member)
	this_.inVarScript = oldInVarScript
	if err != nil {
		return
	}
	this_.AppendCode("]")

	return
}
