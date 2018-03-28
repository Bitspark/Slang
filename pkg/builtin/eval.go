package builtin

import (
	"errors"
	"github.com/Bitspark/slang/pkg/core"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
	"math"
)

type EvaluableExpression struct {
	govaluate.EvaluableExpression
}

func getFlattenedObj(obj interface{}) map[string]interface{} {
	flatMap := make(map[string]interface{})

	if a, ok := obj.([]interface{}); ok {
		for k, val := range a {
			key := strconv.Itoa(k)
			var sub interface{}
			var ok bool

			if sub, ok = val.(map[string]interface{}); !ok {
				if sub, ok = val.([]interface{}); !ok {
					flatMap[key] = val
					continue
				}
			}

			for sKey, sVal := range getFlattenedObj(sub) {
				flatMap[key+"__"+sKey] = sVal
			}
		}
	} else if m, ok := obj.(map[string]interface{}); ok {
		for key, val := range m {
			var sub interface{}
			var ok bool

			if sub, ok = val.(map[string]interface{}); !ok {
				if sub, ok = val.([]interface{}); !ok {
					flatMap[key] = val
					continue
				}
			}

			for sKey, sVal := range getFlattenedObj(sub) {
				flatMap[key+"__"+sKey] = sVal
			}
		}

	} else {
		panic("obj must be list or map")
	}

	return flatMap
}

func newFlatMapParameters(m map[string]interface{}) govaluate.MapParameters {
	flatMap := getFlattenedObj(m)
	return govaluate.MapParameters(flatMap)
}

func evalFunctions() map[string]govaluate.ExpressionFunction {
	functions := map[string]govaluate.ExpressionFunction {
		"floor": func(args ...interface{}) (interface{}, error) {
			if fval, ok := args[0].(float64); ok {
				return math.Floor(fval), nil
			}
			if ival, ok := args[0].(int); ok {
				return ival, nil
			}
			return nil, errors.New("wrong type")
		},
		"ceil": func(args ...interface{}) (interface{}, error) {
			if fval, ok := args[0].(float64); ok {
				return math.Ceil(fval), nil
			}
			if ival, ok := args[0].(int); ok {
				return ival, nil
			}
			return nil, errors.New("wrong type")
		},
		"isNull": func(args ...interface{}) (interface{}, error) {
			if len(args) == 0 {
				return true, nil
			}
			return args[0] == nil, nil
		},
	}
	return functions
}

func newEvaluableExpression(expression string) (*EvaluableExpression, error) {
	expression = strings.Replace(expression, "\\.", "%DOT%", -1)
	expression = strings.Replace(expression, ".", "__", -1)
	expression = strings.Replace(expression, "%DOT%", ".", -1)
	goEvalExpr, err := govaluate.NewEvaluableExpressionWithFunctions(expression, evalFunctions())
	if err == nil {
		return &EvaluableExpression{*goEvalExpr}, nil
	}
	return nil, err
}

var evalOpCfg = &builtinConfig{
	oDef: core.OperatorDef{
		ServiceDefs: map[string]*core.ServiceDef{
			core.MAIN_SERVICE: {
				In: core.TypeDef{
					Type:    "generic",
					Generic: "paramsMap",
				},
				Out: core.TypeDef{
					Type: "primitive",
				},
			},
		},
	},
	oFunc: func(op *core.Operator) {
		expr, _ := newEvaluableExpression(op.Property("expression").(string))
		in := op.Main().In()
		out := op.Main().Out()
		for {
			i := in.Pull()

			if core.IsMarker(i) {
				out.Push(i)
				continue
			}

			if m, ok := i.(map[string]interface{}); ok {
				rlt, _ := expr.Eval(newFlatMapParameters(m))
				out.Push(rlt)
			} else {
				panic("invalid item")
			}
		}
	},
}
