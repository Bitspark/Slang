package builtin

import (
	"github.com/stretchr/testify/require"
	"github.com/Bitspark/slang/pkg/core"
	"github.com/Bitspark/slang/tests/assertions"
	"testing"
)

func TestBuiltin_Fork__CreatorFuncIsRegistered(t *testing.T) {
	a := assertions.New(t)

	ocFork := getBuiltinCfg("slang.fork")
	a.NotNil(ocFork)
}

func TestBuiltin_Fork__InPorts(t *testing.T) {
	a := assertions.New(t)

	o, err := MakeOperator(
		core.InstanceDef{
			Operator: "slang.fork",
			Generics: map[string]*core.TypeDef{
				"itemType": {
					Type: "primitive",
				},
			},
		},
	)
	require.NoError(t, err)

	a.NotNil(o.Main().In().Stream().Map("item"))
	a.NotNil(o.Main().In().Stream().Map("select"))
	a.Equal(core.TYPE_PRIMITIVE, o.Main().In().Stream().Map("item").Type())
	a.Equal(core.TYPE_BOOLEAN, o.Main().In().Stream().Map("select").Type())
}

func TestBuiltin_Fork__OutPorts(t *testing.T) {
	a := assertions.New(t)

	o, err := MakeOperator(
		core.InstanceDef{
			Operator: "slang.fork",
			Generics: map[string]*core.TypeDef{
				"itemType": {
					Type: "primitive",
				},
			},
		},
	)
	require.NoError(t, err)

	a.NotNil(o.Main().Out().Map("true").Stream())
	a.NotNil(o.Main().Out().Map("false").Stream())
	a.Equal(core.TYPE_PRIMITIVE, o.Main().Out().Map("true").Stream().Type())
	a.Equal(core.TYPE_PRIMITIVE, o.Main().Out().Map("false").Stream().Type())
}

func TestBuiltin_Fork__Correct(t *testing.T) {
	a := assertions.New(t)

	o, err := MakeOperator(
		core.InstanceDef{
			Operator: "slang.fork",
			Generics: map[string]*core.TypeDef{
				"itemType": {
					Type: "primitive",
				},
			},
		},
	)
	require.NoError(t, err)

	o.Main().Out().Bufferize()
	o.Start()

	o.Main().In().Push([]interface{}{
		map[string]interface{}{
			"item":   "hallo",
			"select": true,
		},
		map[string]interface{}{
			"item":   "welt",
			"select": false,
		},
		map[string]interface{}{
			"item":   100,
			"select": true,
		},
		map[string]interface{}{
			"item":   101,
			"select": false,
		},
	})

	a.PortPushesAll([]interface{}{[]interface{}{"hallo", 100}}, o.Main().Out().Map("true"))
	a.PortPushesAll([]interface{}{[]interface{}{"welt", 101}}, o.Main().Out().Map("false"))
}

func TestBuiltin_Fork__ComplexItems(t *testing.T) {
	a := assertions.New(t)
	o, err := MakeOperator(
		core.InstanceDef{
			Operator: "slang.fork",
			Generics: map[string]*core.TypeDef{
				"itemType": {
					Type: "map",
					Map: map[string]*core.TypeDef{
						"a": {Type: "number"},
						"b": {Type: "string"},
					},
				},
			},
		},
	)
	a.NoError(err)

	o.Main().Out().Bufferize()
	o.Start()

	o.Main().In().Push([]interface{}{
		map[string]interface{}{
			"item":   map[string]interface{}{"a": "1", "b": "hallo"},
			"select": true,
		},
		map[string]interface{}{
			"item":   map[string]interface{}{"a": "2", "b": "slang"},
			"select": false,
		},
	})

	a.PortPushesAll([]interface{}{[]interface{}{map[string]interface{}{"a": "1", "b": "hallo"}}}, o.Main().Out().Map("true"))
	a.PortPushesAll([]interface{}{[]interface{}{map[string]interface{}{"a": "2", "b": "slang"}}}, o.Main().Out().Map("false"))
}