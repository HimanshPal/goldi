package tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fgrosse/goldi"
	"github.com/fgrosse/goldi/tests/testAPI"
)

var _ = Describe("Container", func() {
	var (
		registry  goldi.TypeRegistry
		config    map[string]interface{}
		container *goldi.Container
	)

	BeforeEach(func() {
		registry = goldi.NewTypeRegistry()
		config = map[string]interface{}{}
		container = goldi.NewContainer(registry, config)
	})

	It("should panic if a type can not be resolved", func() {
		Expect(func() { container.Get("foo.bar") }).To(Panic())
	})

	It("should resolve simple types", func() {
		Expect(registry.RegisterType("goldi.test_type", testAPI.NewMockType)).To(Succeed())
		Expect(container.Get("goldi.test_type")).To(BeAssignableToTypeOf(&testAPI.MockType{}))
	})

	It("should pass static parameters as arguments when generating types", func() {
		typeID := "goldi.test_type"
		typeDef, err := goldi.NewType(testAPI.NewMockTypeWithArgs, "parameter1", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(registry.Register(typeID, typeDef)).To(Succeed())

		generatedType := container.Get("goldi.test_type")
		Expect(generatedType).NotTo(BeNil())
		Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.MockType{}))

		generatedMock := generatedType.(*testAPI.MockType)
		Expect(generatedMock.StringParameter).To(Equal("parameter1"))
		Expect(generatedMock.BoolParameter).To(Equal(true))
	})

	It("should be able to use parameters as arguments when generating types", func() {
		typeID := "goldi.test_type"
		typeDef, err := goldi.NewType(testAPI.NewMockTypeWithArgs, "%parameter1%", "%parameter2%")
		Expect(err).NotTo(HaveOccurred())
		Expect(registry.Register(typeID, typeDef)).To(Succeed())

		config["parameter1"] = "test"
		config["parameter2"] = true

		generatedType := container.Get("goldi.test_type")
		Expect(generatedType).NotTo(BeNil())
		Expect(generatedType).To(BeAssignableToTypeOf(&testAPI.MockType{}))

		generatedMock := generatedType.(*testAPI.MockType)
		Expect(generatedMock.StringParameter).To(Equal(config["parameter1"]))
		Expect(generatedMock.BoolParameter).To(Equal(config["parameter2"]))
	})
})