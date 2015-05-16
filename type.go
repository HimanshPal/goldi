package goldi

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// A Type holds all information that is necessary to create a new instance of a type ID
type Type struct {
	generator          reflect.Value
	generatorType      reflect.Type
	generatorArguments []reflect.Value
}

// NewType creates a new Type and checks if the given factory method can be used get a go type
//
// This function will panic if the factoryFunction is no function, returns zero or more than
// one parameter or the return parameter is no pointer or interface type.
// If the number of given factoryParameters does not match the number of arguments of the
// factoryFunction this function will panic as well
func NewType(factoryFunction interface{}, factoryParameters ...interface{}) *Type {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("could not register type: %v", r))
		}
	}()

	generatorType := reflect.TypeOf(factoryFunction)
	if generatorType.Kind() != reflect.Func {
		panic(fmt.Errorf("kind was %v, not Func", generatorType.Kind()))
	}

	if generatorType.NumOut() != 1 {
		panic(fmt.Errorf("invalid number of return parameters: %d", generatorType.NumOut()))
	}

	kindOfGeneratedType := generatorType.Out(0).Kind()
	if kindOfGeneratedType != reflect.Interface && kindOfGeneratedType != reflect.Ptr {
		panic(fmt.Errorf("return parameter is no interface or pointer but a %v", kindOfGeneratedType))
	}

	if generatorType.NumIn() != len(factoryParameters) {
		panic(fmt.Errorf("invalid number of input parameters: got %d but expected %d", generatorType.NumIn(), len(factoryParameters)))
	}

	return &Type{
		generator:          reflect.ValueOf(factoryFunction),
		generatorType:      generatorType,
		generatorArguments: buildGeneratorCallArguments(generatorType, factoryParameters),
	}
}

func buildGeneratorCallArguments(generatorType reflect.Type, factoryParameters []interface{}) []reflect.Value {
	args := make([]reflect.Value, len(factoryParameters))
	for i, argument := range factoryParameters {
		expectedArgumentType := generatorType.In(i)
		args[i] = reflect.ValueOf(argument)
		if args[i].Kind() != expectedArgumentType.Kind() {
			if stringArg, isString := argument.(string); isString && isParameterOrTypeReference(stringArg) == false {
				panic(fmt.Errorf("input argument %d is of type %s but needs to be a %s", i+1, args[i].Kind(), expectedArgumentType.Kind()))
			}
		}
	}

	return args
}

// Generate will instantiate a new instance of the according type.
// The given configuration is used to resolve parameters that are used in the type factory method
// The type registry is used to lazily resolve type references
func (t *Type) Generate(config map[string]interface{}, registry TypeRegistry) interface{} {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("could not generate type: %v", r))
		}
	}()

	args := make([]reflect.Value, len(t.generatorArguments))
	for i, argument := range t.generatorArguments {
		args[i] = t.resolveParameter(i, argument, t.generatorType.In(i), config, registry)
	}

	result := t.generator.Call(args)
	if len(result) == 0 {
		panic(fmt.Errorf("no return parameter found. Seems like you did not use goldi.NewTypeGenerator to create this TypeGenerator"))
	}

	return result[0].Interface()
}

func (t *Type) resolveParameter(i int, argument reflect.Value, expectedArgument reflect.Type, config map[string]interface{}, registry TypeRegistry) reflect.Value {
	if argument.Kind() != reflect.String {
		return argument
	}

	stringArgument := argument.Interface().(string)
	if isParameterOrTypeReference(stringArgument) == false {
		return argument
	}

	if stringArgument[0] == '@' {
		return t.resolveTypeReference(i, stringArgument[1:], config, registry, expectedArgument)
	}

	parameterName := stringArgument[1 : len(stringArgument)-1]
	configuredValue, isConfigured := config[parameterName]
	if isConfigured == false {
		return argument
	}

	argument = reflect.New(expectedArgument).Elem()
	argument.Set(reflect.ValueOf(configuredValue))
	return argument
}

func (t *Type) resolveTypeReference(i int, typeID string, config map[string]interface{}, registry TypeRegistry, expectedArgument reflect.Type) reflect.Value {
	referencedType, typeDefined := registry[typeID]
	if typeDefined == false {
		panic(fmt.Errorf("the referenced type \"@%s\" has not been defined", typeID))
	}

	typeInstance := referencedType.Generate(config, registry)
	if reflect.TypeOf(typeInstance).AssignableTo(expectedArgument) == false {
		panic(t.invalidReferencedTypeErr(typeID, typeInstance, i))
	}

	// TODO check if this type is assignable and generate helpful error message if not
	argument := reflect.New(expectedArgument).Elem()
	argument.Set(reflect.ValueOf(typeInstance))
	return argument
}

func (t *Type) invalidReferencedTypeErr(typeID string, typeInstance interface{}, i int) error {
	factoryName := runtime.FuncForPC(t.generator.Pointer()).Name()
	factoryNameParts := strings.Split(factoryName, "/")
	factoryName = factoryNameParts[len(factoryNameParts)-1]

	n := t.generator.Type().NumIn()
	factoryArguments := make([]string, n)
	for i := 0; i < n; i++ {
		arg := t.generator.Type().In(i)
		factoryArguments[i] = arg.String()
	}

	err := fmt.Errorf("the referenced type \"@%s\" (type %T) can not be passed as argument %d to the function signature %s(%s)",
		typeID, typeInstance, i+1, factoryName, strings.Join(factoryArguments, ", "),
	)

	return err
}

// typeReferenceArguments is an internal function that returns all generator arguments that are type references
func (t *Type) typeReferenceArguments() []string {
	var typeRefParameters []string
	for _, argument := range t.generatorArguments {
		stringArgument := argument.Interface().(string)
		if isTypeReference(stringArgument) {
			typeRefParameters = append(typeRefParameters, stringArgument[1:])
		}
	}
	return typeRefParameters
}

// parameterArguments is an internal function that returns all generator arguments that are parameters
func (t *Type) parameterArguments() []string {
	var parameterArguments []string
	for _, argument := range t.generatorArguments {
		stringArgument := argument.Interface().(string)
		if isParameter(stringArgument) {
			parameterArguments = append(parameterArguments, stringArgument[1:len(stringArgument)-1])
		}
	}
	return parameterArguments
}

func isParameterOrTypeReference(p string) bool {
	return isParameter(p) || isTypeReference(p)
}

func isParameter(p string) bool {
	if len(p) < 2 {
		return false
	}

	return p[0] == '%' && p[len(p)-1] == '%'
}

func isTypeReference(p string) bool {
	if len(p) < 2 {
		return false
	}

	return p[0] == '@'
}
