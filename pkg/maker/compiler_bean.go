package maker

import (
	"errors"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
	"github.com/team-ide/go-tool/util"
	"go.uber.org/zap"
	"reflect"
	"strings"
	"teamide/pkg/maker/modelers"
)

func (this_ *Compiler) GetOrCreateSpace(space string) (res *CompilerSpace) {
	res = this_.spaceCache[space]
	if res == nil {
		res = &CompilerSpace{
			Space:     space,
			Compiler:  this_,
			packCache: make(map[string]*CompilerPack),
		}
		this_.SpaceList = append(this_.SpaceList, res)
		this_.spaceCache[space] = res
	}
	return
}

type CompilerSpace struct {
	*Compiler
	Space     string
	PackList  []*CompilerPack
	packCache map[string]*CompilerPack
}

func (this_ *CompilerSpace) GetKey() (key string) {
	key = "space [" + this_.Space + "]"
	return
}

func (this_ *CompilerSpace) GetOrCreatePack(pack string) (res *CompilerPack) {
	res = this_.packCache[pack]
	if res == nil {
		res = &CompilerPack{
			Pack:          pack,
			CompilerSpace: this_,
			classCache:    make(map[string]*CompilerClass),
		}
		this_.PackList = append(this_.PackList, res)
		this_.packCache[pack] = res
	}
	return
}

type CompilerPack struct {
	*CompilerSpace
	Pack       string
	ClassList  []*CompilerClass
	classCache map[string]*CompilerClass
}

func (this_ *CompilerSpace) GetClass(path string, fileIsClass bool) (endName string, res *CompilerClass) {
	names := strings.Split(path, "/")
	size := len(names)
	endNameIndex := -1
	if !fileIsClass {
		endNameIndex = size - 1
	}
	var class []string
	if endNameIndex >= 0 {
		if endNameIndex > 0 {
			class = names[:endNameIndex]
		}
	} else {
		class = names
	}
	if len(class) == 0 {
		class = []string{"base"}
	}
	pack := this_.GetOrCreatePack("")
	res = pack.GetOrCreateClass(class)
	if endNameIndex >= 0 {
		endName = names[endNameIndex]
	}
	return
}

func (this_ *CompilerPack) GetKey() (key string) {
	key = this_.CompilerSpace.GetKey() + " pack [" + this_.Pack + "]"
	return
}

func (this_ *CompilerPack) GetOrCreateClass(class []string) (res *CompilerClass) {
	classKey := strings.Join(class, "_")
	res = this_.classCache[classKey]
	if res == nil {
		res = &CompilerClass{
			Class:            class,
			CompilerPack:     this_,
			importClassCache: make(map[*CompilerClass]*CompilerImport),
			importCache:      make(map[string]*CompilerImport),
			fieldCache:       make(map[string]*CompilerField),
			methodCache:      make(map[string]*CompilerMethod),
		}

		this_.ClassList = append(this_.ClassList, res)
		this_.classCache[classKey] = res
	}
	return
}

type CompilerClass struct {
	*CompilerPack
	Class            []string
	ImportList       []*CompilerImport
	importClassCache map[*CompilerClass]*CompilerImport
	importCache      map[string]*CompilerImport
	FieldList        []*CompilerField
	fieldCache       map[string]*CompilerField
	MethodList       []*CompilerMethod
	methodCache      map[string]*CompilerMethod
	Constant         *modelers.ConstantModel
	Error            *modelers.ErrorModel
	Struct           *modelers.StructModel
}

func (this_ *CompilerClass) GetKey() (key string) {
	key = this_.CompilerPack.GetKey() + " class [" + strings.Join(this_.Class, "_") + "]"
	return
}

func (this_ *CompilerClass) GetOrCreateClassImport(class *CompilerClass) (res *CompilerImport) {
	res = this_.importClassCache[class]
	if res == nil {
		res = &CompilerImport{
			Class: class,
		}
		this_.ImportList = append(this_.ImportList, res)
		this_.importClassCache[class] = res
	}
	return
}

func (this_ *CompilerClass) GetImport(imp string) (res *CompilerImport) {
	res = this_.importCache[imp]
	return
}
func (this_ *CompilerClass) GetOrCreateImport(imp string) (res *CompilerImport) {
	res = this_.importCache[imp]
	if res == nil {
		res = &CompilerImport{
			Import: imp,
		}
		this_.ImportList = append(this_.ImportList, res)
		this_.importCache[imp] = res
	}
	return
}

type CompilerImport struct {
	Class  *CompilerClass
	Import string
	AsName string
	IsThis bool
}

func (this_ *CompilerClass) addField(name string, valueType *ValueType) (res *CompilerField) {
	res = &CompilerField{
		CompilerClass: this_,
		Name:          name,
	}
	res.CompilerValueType = NewCompilerValueType(res, valueType)
	this_.FieldList = append(this_.FieldList, res)
	this_.fieldCache[name] = res

	return
}

type CompilerField struct {
	*CompilerClass
	Name string
	*CompilerValueType
	ConstantOption *modelers.ConstantOptionModel
	ErrorOption    *modelers.ErrorOptionModel
	StructField    *modelers.StructField
}

func (this_ *CompilerField) GetKey() (key string) {
	key = this_.CompilerClass.GetKey() + " field [" + this_.Name + "]"
	return
}

func (this_ *CompilerClass) GetMethod(name string) (res *CompilerMethod) {
	res = this_.methodCache[name]
	return
}

func (this_ *CompilerClass) CreateMethod(name string, args []*modelers.ArgModel) (res *CompilerMethod, err error) {

	res = &CompilerMethod{
		CompilerClass:                   this_,
		Method:                          name,
		paramCache:                      make(map[string]*CompilerMethodParam),
		varCache:                        make(map[string]*CompilerMethodVar),
		BindingCache:                    make(map[*ast.Binding]*CompilerMethodVar),
		CallCache:                       make(map[*ast.CallExpression]interface{}),
		CallScriptCache:                 make(map[*ast.CallExpression]string),
		BindingScriptCache:              make(map[*ast.Binding]string),
		AssignExpressionScriptCache:     make(map[*ast.AssignExpression]string),
		AssignExpressionScriptTypeCache: make(map[*ast.AssignExpression]*CompilerValueType),
	}

	for _, arg := range args {
		var t *ValueType
		t, err = this_.GetValueType(arg.Type)
		if err != nil {
			return
		}
		res.addParam(arg.Name, t)
		if err != nil {
			return
		}
	}
	this_.MethodList = append(this_.MethodList, res)
	this_.methodCache[name] = res
	return
}

type CompilerMethod struct {
	*CompilerClass
	Method            string
	Comment           string
	ParamList         []*CompilerMethodParam
	paramCache        map[string]*CompilerMethodParam
	VarList           []*CompilerMethodVar
	varCache          map[string]*CompilerMethodVar
	CallComponentList []*CompilerCall
	CallUtilList      []*CompilerCall
	CallFuncList      []*CompilerCall
	CallStorageList   []*CompilerCall
	CallServiceList   []*CompilerCall
	Result            *CompilerMethodResult
	Program           *ast.Program
	script            *Script
	code              string

	BindingCache                    map[*ast.Binding]*CompilerMethodVar
	CallCache                       map[*ast.CallExpression]interface{}
	CallScriptCache                 map[*ast.CallExpression]string
	BindingScriptCache              map[*ast.Binding]string
	AssignExpressionScriptCache     map[*ast.AssignExpression]string
	AssignExpressionScriptTypeCache map[*ast.AssignExpression]*CompilerValueType
}

func (this_ *CompilerMethod) GetKey() (key string) {
	key = this_.CompilerClass.GetKey() + " method [" + this_.Method + "]"
	return
}

func (this_ *CompilerMethod) getParam(name string) (res *CompilerValueType) {
	this_.fullImport(name)
	index := strings.Index(name, ".")
	var varName = name
	var subName = ""
	if index > 0 {
		varName = name[0:index]
		subName = name[index+1:]
	} else {
		index = strings.Index(name, "[\"")
		if index > 0 {
			varName = name[0:index]
			subName = name[index+2:]
		}
	}
	find := this_.paramCache[varName]
	if find != nil {
		if subName != "" {
			res = find.CompilerValueType.getVar(subName)
		} else {
			res = find.CompilerValueType
		}
	}
	return
}

func (this_ *CompilerMethod) fullImport(name string) {
	var importName = name
	index := strings.Index(name, ".")
	if index > 0 {
		importName = name[0:index]
	}
	if importName == this_.Space {
		return
	}
	switch importName {
	case "logger":
		this_.GetOrCreateImport(importName)
		break
	case "fmt":
		this_.GetOrCreateImport(importName)
		break
	case "context":
		this_.GetOrCreateImport(importName)
		break
	case "util":
		this_.GetOrCreateImport(importName)
		break
	case "constant":
		this_.GetOrCreateImport(importName)
		break
	case "error":
		this_.GetOrCreateImport(importName)
		break
	case "struct":
		this_.GetOrCreateImport(importName)
		break
	case "storage":
		this_.GetOrCreateImport(importName)
		break
	case "service":
		this_.GetOrCreateImport(importName)
		break
	case "func":
		this_.GetOrCreateImport(importName)
		break
	case "common":
		this_.GetOrCreateImport(importName)
		break
	default:
		if importName == "db" || strings.HasPrefix(importName, "db_") {
			this_.GetOrCreateImport("component_" + importName)
		}
		if importName == "redis" || strings.HasPrefix(importName, "redis_") {
			this_.GetOrCreateImport("component_" + importName)
		}
		if importName == "zk" || strings.HasPrefix(importName, "zk_") {
			this_.GetOrCreateImport("component_" + importName)
		}
		if importName == "kafka" || strings.HasPrefix(importName, "kafka_") {
			this_.GetOrCreateImport("component_" + importName)
		}
		if importName == "es" || strings.HasPrefix(importName, "es_") {
			this_.GetOrCreateImport("component_" + importName)
		}
	}
	return
}

func (this_ *CompilerMethod) addParam(name string, valueType *ValueType) (res *CompilerMethodParam) {
	util.Logger.Debug(this_.GetKey()+" set param ["+name+"] ", zap.Any("valueType", valueType))
	res = &CompilerMethodParam{
		CompilerMethod: this_,
		Name:           name,
	}
	res.CompilerValueType = NewCompilerValueType(res, valueType)
	this_.ParamList = append(this_.ParamList, res)
	this_.paramCache[name] = res
	if valueType == ValueTypeContext {
		this_.fullImport("context")
	}

	return
}

func (this_ *CompilerMethod) getVar(name string) (res *CompilerValueType) {
	this_.fullImport(name)
	index := strings.Index(name, ".")
	var varName = name
	var subName = ""
	if index > 0 {
		varName = name[0:index]
		subName = name[index+1:]
	} else {
		index = strings.Index(name, "[\"")
		if index > 0 {
			varName = name[0:index]
			subName = name[index+2:]
		}
	}
	find := this_.varCache[varName]
	if find != nil {
		if subName != "" {
			res = find.CompilerValueType.getVar(subName)
		} else {
			res = find.CompilerValueType
		}
	}
	return
}

func (this_ *CompilerMethod) addVar(name string, valueType *ValueType) (res *CompilerMethodVar) {
	util.Logger.Debug(this_.GetKey()+" set var ["+name+"] ", zap.Any("valueType", valueType))
	res = &CompilerMethodVar{
		CompilerMethod: this_,
		Name:           name,
	}
	res.CompilerValueType = NewCompilerValueType(res, valueType)
	this_.VarList = append(this_.VarList, res)
	this_.varCache[name] = res

	return
}

func (this_ *CompilerMethod) getByNameScript(nameScript string) (res interface{}) {
	names := strings.Split(nameScript, `.`)
	if len(names) == 0 {
		fmt.Println("name script [", nameScript, "] is empty")
		return
	}
	v := this_.script.dataContext[names[0]]
	if v == nil {
		return
	}
	switch tV := v.(type) {
	case *CompilerMethodVar:
		res = tV
		break
	case *CompilerMethodParam:
		res = tV
		break
	default:
		panic("getByNameScript [" + reflect.TypeOf(v).String() + "] not support")
	}
	return
}
func (this_ *CompilerMethod) FindType(name string) (find bool) {
	_, find = this_.varCache[name]
	if !find {
		_, find = this_.paramCache[name]
	}
	return
}

func (this_ *CompilerMethod) BindCode(script string) (err error) {
	this_.code = `(function (){
` + script + `
})()`
	this_.Program, err = goja.Parse("", this_.code)
	if err != nil {
		util.Logger.Error("compile script parse error", zap.Any("error", err))
		return
	}
	return
}

func (this_ *CompilerMethod) Error(msg string, node ast.Node) (err error) {
	err = errors.New(msg + ",code:" + this_.GetNodeCode(node))
	return
}
func (this_ *CompilerMethod) GetNodeCode(node ast.Node) (code string) {
	code = this_.code[node.Idx0()-1 : node.Idx1()-1]
	return
}

type CompilerMethodParam struct {
	*CompilerMethod
	Name string
	*CompilerValueType
}

type CompilerMethodVar struct {
	*CompilerMethod
	Name string
	*CompilerValueType
	Value string
}

type CompilerCall struct {
	*CompilerMethod
	name       string
	paramList  []*CompilerCallParam
	returnList []*ValueType
}

type CompilerCallParam struct {
	*CompilerCall
	Name string
	*CompilerValueType
	Value string
}

type CompilerValueType struct {
	Parent    interface{}
	valueType *ValueType
	types     []*ValueType

	subList  []*CompilerValueSub
	subCache map[string]*CompilerValueSub
}

type CompilerValueSub struct {
	Parent *CompilerValueType
	Name   string
	*CompilerValueType
	Value string
}

func (this_ *CompilerValueType) GetValueType() (res *ValueType) {
	res = this_.valueType
	return
}

func (this_ *CompilerValueType) getVar(name string) (res *CompilerValueType) {
	if this_.subCache == nil {
		this_.subCache = make(map[string]*CompilerValueSub)
	}
	index := strings.Index(name, ".")
	var varName = name
	var subName = ""
	if index > 0 {
		varName = name[0:index]
		subName = name[index+1:]
	} else {
		index = strings.Index(name, "[\"")
		if index > 0 {
			varName = name[0:index]
			subName = name[index+2:]
		}
	}
	if strings.HasSuffix(varName, "\"]") {
		varName = strings.TrimSuffix(varName, "\"]")
	}
	find := this_.subCache[varName]
	if find != nil {
		if subName != "" {
			res = find.CompilerValueType.getVar(subName)
		} else {
			res = find.CompilerValueType
		}
	} else {
		if this_.valueType == nil {
			this_.types = append(this_.types, ValueTypeMap)
			this_.valueType = ValueTypeMap
		}
		if this_.valueType.FieldTypes == nil {
			this_.valueType.FieldTypes = make(map[string]*ValueType)
		}
		t := this_.valueType.FieldTypes[varName]
		if t != nil {
			sub := this_.addVar(varName, t)
			res = sub.CompilerValueType
		} else if this_.valueType == ValueTypeMap {
			sub := this_.addVar(varName, nil)
			res = sub.CompilerValueType
		}
	}
	return
}
func (this_ *CompilerValueType) addVar(name string, valueType *ValueType) (res *CompilerValueSub) {
	res = &CompilerValueSub{
		Parent: this_,
		Name:   name,
	}
	res.CompilerValueType = NewCompilerValueType(res, valueType)
	this_.subList = append(this_.subList, res)
	this_.subCache[name] = res

	return
}

func NewCompilerValueType(parent interface{}, valueType *ValueType) (res *CompilerValueType) {
	res = &CompilerValueType{
		Parent: parent,
	}
	if valueType != nil {
		res.types = append(res.types, valueType)
		res.valueType = valueType
	}
	return
}
func (this_ *CompilerValueType) addValueTypes(valueTypes ...*ValueType) (err error) {

	for _, valueType := range valueTypes {
		var find bool
		for _, v := range this_.types {
			if v == valueType {
				find = true
			}
		}
		if !find {
			if this_.valueType == nil {
				this_.valueType = valueType
			} else {
				this_.valueType, err = this_.upgradeType(this_.valueType, valueType)
				if err != nil {
					return
				}
			}
			this_.types = append(this_.types, valueType)
		}
	}
	return
}

func (this_ *CompilerValueType) upgradeType(from *ValueType, to *ValueType) (endType *ValueType, err error) {
	if from == to {
		endType = to
		return
	}
	if from.IsNumber && to.IsNumber {
		endType = to
		return
	}
	if to == ValueTypeNull {
		endType = from
		return
	}
	err = errors.New("类型 [" + from.Name + "] [" + to.Name + "] 不一致")

	return
}

type CompilerMethodResult struct {
	*CompilerValueType
}
