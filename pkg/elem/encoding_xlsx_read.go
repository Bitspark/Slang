package elem

import (
	"github.com/Bitspark/slang/pkg/core"
	"github.com/tealeg/xlsx"
	"path/filepath"
)

var encodingXLSXReadCfg = &builtinConfig{
	opDef: core.OperatorDef{
		ServiceDefs: map[string]*core.ServiceDef{
			core.MAIN_SERVICE: {
				In: core.TypeDef{
					Type: "string",
				},
				Out: core.TypeDef{
					Type: "stream",
					Stream: &core.TypeDef{
						Type: "map",
						Map: map[string]*core.TypeDef{
							"name": {
								Type: "string",
							},
							"table": {
								Type: "stream",
								Stream: &core.TypeDef{
									Type: "stream",
									Stream: &core.TypeDef{
										Type: "string",
									},
								},
							},
						},
					},
				},
			},
		},
		DelegateDefs: map[string]*core.DelegateDef{},
	},
	opFunc: func(op *core.Operator) {
		in := op.Main().In()
		out := op.Main().Out()
		for !op.CheckStop() {
			filename, i := in.PullString()
			if i != nil {
				out.Push(i)
				continue
			}
			xlsxFile, err := xlsx.OpenFile(filepath.Join(core.WORKING_DIR, filename))
			if err != nil {
				panic(err)
			}
			out.PushBOS()
			outTable := out.Stream().Map("table")
			for _, sheet := range xlsxFile.Sheets {
				out.Stream().Map("name").Push(sheet.Name)
				outTable.PushBOS()
				for _, row := range sheet.Rows {
					outTable.Stream().PushBOS()
					for _, col := range row.Cells {
						outTable.Stream().Stream().Push(col.Value)
					}
					outTable.Stream().PushEOS()
				}
				outTable.PushEOS()
			}
			out.PushEOS()
		}
	},
}
