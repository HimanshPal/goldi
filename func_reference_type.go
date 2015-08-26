package goldi

import (
	"fmt"
	"reflect"
	"unicode"
)

type funcReferenceType struct {
	*TypeID
}

// NewFuncReferenceType returns a TypeFactory that returns a method of another type as function.
func NewFuncReferenceType(typeID, functionName string) TypeFactory {
	if functionName == "" || unicode.IsLower(rune(functionName[0])) {
		return newInvalidType(fmt.Errorf("can not use unexported method %q as second argument to NewFuncReferenceType", functionName))
	}

	return &funcReferenceType{NewTypeID("@"+typeID + "::" + functionName)}
}

func (t *funcReferenceType) Arguments() []interface{} {
	return []interface{}{"@" + t.ID}
}

func (t *funcReferenceType) Generate(resolver *ParameterResolver) (interface{}, error) {
	referencedType, err := resolver.Container.Get(t.ID)
	if err != nil {
		return nil, fmt.Errorf("could not generate func reference type %s : type %s does not exist", t.ID)
	}

	v := reflect.ValueOf(referencedType)
	method := v.MethodByName(t.FuncReferenceMethod)

	if method.IsValid() == false {
		return nil, fmt.Errorf("could not generate func reference type %s : method does not exist", t)
	}

	return method.Interface(), nil
}