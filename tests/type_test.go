package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"github.com/fgrosse/goldi"
	"github.com/fgrosse/goldi/tests/testAPI"
)

var _ = Describe("Type", func() {
	var typeDef *goldi.Type

	Describe("NewType()", func() {
		Context("with invalid factory function", func() {
			It("should panic if the generator is no function", func() {
				Expect(func() { goldi.NewType(42) }).To(Panic())
			})

			It("should panic if the generator has no output parameters", func() {
				Expect(func() { goldi.NewType(func() {}) }).To(Panic())
			})

			It("should panic if the generator has more than one output parameter", func() {
				Expect(func() { goldi.NewType(func() (*testAPI.MockType, *testAPI.MockType) { return nil, nil }) }).To(Panic())
			})

			It("should panic if the return parameter is no pointer", func() {
				Expect(func() { goldi.NewType(func() testAPI.MockType { return testAPI.MockType{} }) }).To(Panic())
			})

			It("should not panic if the return parameter is an interface", func() {
				Expect(func() { goldi.NewType(func() interface{} { return testAPI.MockType{} }) }).NotTo(Panic())
			})
		})

		Context("with factory functions without arguments", func() {
			Context("when no factory argument is given", func() {
				It("should create the type", func() {
					typeDef = goldi.NewType(testAPI.NewMockType)
					Expect(typeDef).NotTo(BeNil())
				})
			})

			Context("when any argument is given", func() {
				It("should panic", func() {
					Expect(func() { goldi.NewType(testAPI.NewMockType, "foo") }).To(Panic())
				})
			})
		})

		Context("with factory functions with one or more arguments", func() {
			Context("when an invalid number of arguments is given", func() {
				It("should panic", func() {
					Expect(func() { goldi.NewType(testAPI.NewMockTypeWithArgs) }).To(Panic())
					Expect(func() { goldi.NewType(testAPI.NewMockTypeWithArgs, "foo") }).To(Panic())
					Expect(func() { goldi.NewType(testAPI.NewMockTypeWithArgs, "foo", false, 42) }).To(Panic())
				})
			})

			Context("when the wrong argument types are given", func() {
				It("should panic", func() {
					Expect(func() { goldi.NewType(testAPI.NewMockTypeWithArgs, "foo", "bar") }).To(Panic())
					Expect(func() { goldi.NewType(testAPI.NewMockTypeWithArgs, true, "bar") }).To(Panic())
				})
			})

			Context("when the correct argument number and types are given", func() {
				It("should create the type", func() {
					typeDef = goldi.NewType(testAPI.NewMockTypeWithArgs, "foo", true)
					Expect(typeDef).NotTo(BeNil())
				})
			})
		})
	})

	Describe("Generate()", func() {
		var (
			config       = map[string]interface{}{}
			typeRegistry goldi.TypeRegistry
		)

		BeforeEach(func() {
			typeRegistry = goldi.NewTypeRegistry()
		})

		Context("with factory functions without arguments", func() {
			It("should generate the type", func() {
				typeDef = goldi.NewType(testAPI.NewMockType)
				Expect(typeDef.Generate(config, typeRegistry)).To(BeAssignableToTypeOf(&testAPI.MockType{}))
			})
		})

		Context("with factory functions with one or more arguments", func() {
			It("should generate the type", func() {
				typeDef = goldi.NewType(testAPI.NewMockTypeWithArgs, "foo", true)

				generatedType := typeDef.Generate(config, typeRegistry)
				Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.MockType{}))

				generatedMock := generatedType.(*testAPI.MockType)
				Expect(generatedMock.StringParameter).To(Equal("foo"))
				Expect(generatedMock.BoolParameter).To(Equal(true))
			})

			Context("when a type reference is given", func() {
				Context("and its type matches the function signature", func() {
					It("should generate the type", func() {
						err := typeRegistry.RegisterType("foo", testAPI.NewMockType)
						Expect(err).NotTo(HaveOccurred())

						typeDef = goldi.NewType(testAPI.NewTypeForServiceInjection, "@foo")
						generatedType := typeDef.Generate(config, typeRegistry)
						Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.TypeForServiceInjection{}))

						generatedMock := generatedType.(*testAPI.TypeForServiceInjection)
						Expect(generatedMock.InjectedType).To(BeAssignableToTypeOf(&testAPI.MockType{}))
					})
				})

				Context("and its type does not match the function signature", func() {
					It("should panic with a helpful error message", func() {
						err := typeRegistry.RegisterType("foo", testAPI.NewFoo)
						Expect(err).NotTo(HaveOccurred())

						typeDef = goldi.NewType(testAPI.NewTypeForServiceInjectionWithArgs, "@foo", "arg1", "arg2", true)

						defer func() {
							r := recover()
							Expect(r).NotTo(BeNil(), "Expected Generate to panic")
							Expect(r).To(BeAssignableToTypeOf(errors.New("")))
							err := r.(error)
							Expect(err.Error()).To(Equal("could not generate type: the referenced type \"@foo\" (type *testAPI.Foo) can not be passed as argument 1 to the function signature testAPI.NewTypeForServiceInjectionWithArgs(*testAPI.MockType, string, string, bool)"))
						}()

						typeDef.Generate(config, typeRegistry)
					})
				})
			})
		})
	})
})
