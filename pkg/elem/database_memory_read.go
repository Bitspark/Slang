package elem

import (
	"github.com/Bitspark/slang/pkg/core"
	"sync"
)

type memoryStore struct {
	mutex *sync.Mutex
	items map[string]interface{}
}

var memoryStores map[string]*memoryStore
var memoryMutex *sync.Mutex

func getMemoryStore(store string) *memoryStore {
	memoryMutex.Lock()
	ms, ok := memoryStores[store]
	if !ok {
		ms = &memoryStore{}
		ms.mutex = &sync.Mutex{}
		ms.items = make(map[string]interface{})
		memoryStores[store] = ms
	}
	memoryMutex.Unlock()
	return ms
}

var databaseMemoryReadCfg = &builtinConfig{
	opDef: core.OperatorDef{
		ServiceDefs: map[string]*core.ServiceDef{
			core.MAIN_SERVICE: {
				In: core.TypeDef{
					Type: "map",
					Map: map[string]*core.TypeDef{
						"key": {
							Type: "string",
						},
						"keyValue": {
							Type:    "generic",
							Generic: "keyType",
						},
					},
				},
				Out: core.TypeDef{
					Type:    "generic",
					Generic: "valueType",
				},
			},
		},
		DelegateDefs: map[string]*core.DelegateDef{
			"creator": {
				Out: core.TypeDef{
					Type:    "generic",
					Generic: "keyType",
				},
				In: core.TypeDef{
					Type:    "generic",
					Generic: "valueType",
				},
			},
		},
		PropertyDefs: map[string]*core.TypeDef{
			"store": {
				Type: "string",
			},
		},
	},
	opFunc: func(op *core.Operator) {
		in := op.Main().In()
		out := op.Main().Out()

		creatorOut := op.Delegate("creator").Out()
		creatorIn := op.Delegate("creator").In()

		// Get store
		store := op.Property("store").(string)
		ms := getMemoryStore(store)

		for {
			i := in.Pull()
			if core.IsMarker(i) {
				out.Push(i)
				continue
			}

			keyMap := i.(map[string]interface{})
			key := keyMap["key"].(string)
			keyValue := keyMap["keyValue"]

			ms.mutex.Lock()
			if value, ok := ms.items[key]; ok {
				out.Push(value)
			} else {
				creatorOut.Push(keyValue)
				value := creatorIn.Pull()
				ms.items[key] = value
				out.Push(value)
			}
			ms.mutex.Unlock()
		}
	},
}