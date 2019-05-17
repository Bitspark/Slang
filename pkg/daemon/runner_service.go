package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/Bitspark/go-funk"
	"github.com/Bitspark/slang/pkg/api"
	"github.com/Bitspark/slang/pkg/core"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type RunInstruction struct {
	Id     uuid.UUID       `json:"id"`
	Props  core.Properties `json:"props"`
	Gens   core.Generics   `json:"gens"`
	Stream bool            `json:"stream"`
}

type RunState struct {
	Handle string `json:"handle,omitempty"`
	URL    string `json:"url,omitempty"`
	Status string `json:"status"`
	Error  *Error `json:"error,omitempty"`
}

type runningOperator struct {
	// JSON
	Op     uuid.UUID
	Handle string
	URL    string

	op       *core.Operator
	incoming chan interface{}
	outgoing chan portMessage
	inStop   chan bool
	outStop  chan bool
}

type portMessage struct {
	Port *core.Port
	Msg  interface{}
}

type runningOperatorManager struct {
	ops map[string]*runningOperator
}

var rnd = rand.New(rand.NewSource(99))
var runningOperators = &runningOperatorManager{make(map[string]*runningOperator)}

type httpDefLoader struct {
	httpDef *core.OperatorDef
}

func (rom *runningOperatorManager) newRunningOperator(op *core.Operator) *runningOperator {
	handle := strconv.FormatInt(rnd.Int63(), 16)
	url := "/instance/" + handle + "/"
	runningOp := &runningOperator{op.Id(), handle, url, op, make(chan interface{}, 0), make(chan portMessage, 0), make(chan bool, 0), make(chan bool, 0)}
	rom.ops[handle] = runningOp
	op.Main().Out().Bufferize()
	op.Start()
	log.Printf("operator %s (id: %s) started", op.Name(), handle)
	return runningOp
}

func (rom *runningOperatorManager) Run(op *core.Operator) *runningOperator {
	runningOp := rom.newRunningOperator(op)
	go func() {
	loop:
		for {
			select {
			case incoming := <-runningOp.incoming:
				op.Main().In().Push(incoming)
			case <-runningOp.inStop:
				break loop
			}
		}
	}()

	op.Main().Out().AsyncWalkPrimitivePorts(func(p *core.Port) {
		for {
			if p.Closed() {
				break
			}
			i := p.Pull()
			if !core.IsMarker(i) {
				runningOp.outgoing <- portMessage{p, i}
			}
		}
	})
	return runningOp
}

func (rom *runningOperatorManager) Halt(handle string) error {
	// `Halt` to me suggest that there is a way to resume operations
	// which is not the case.
	ro, err := runningOperators.Get(handle)

	if err != nil {
		return err
	}

	go ro.op.Stop()
	ro.inStop <- true
	ro.outStop <- true
	delete(rom.ops, handle)

	return nil
}

func (rom runningOperatorManager) Get(handle string) (*runningOperator, error) {
	if runningOp, ok := rom.ops[handle]; ok {
		return runningOp, nil
	}
	return nil, fmt.Errorf("unknown handle value: %s", handle)
}

func (l *httpDefLoader) List() ([]uuid.UUID, error) {
	httpDefId, _ := uuid.Parse(l.httpDef.Id)
	return []uuid.UUID{httpDefId}, nil
}

func (l *httpDefLoader) Has(opId uuid.UUID) bool {
	httpDefId, _ := uuid.Parse(l.httpDef.Id)
	return httpDefId == opId
}

func (l *httpDefLoader) Load(opId uuid.UUID) (*core.OperatorDef, error) {
	if !l.Has(opId) {
		return nil, fmt.Errorf("")
	}
	return l.httpDef, nil
}

var InstanceService = &Service{map[string]*Endpoint{
	"/": {func(w http.ResponseWriter, r *http.Request) {

		type outJSON struct {
			Objects []runningOperator `json:"objects"`
			Status  string            `json:"status"`
			Error   *Error            `json:"error,omitempty"`
		}

		if r.Method == "GET" {
			writeJSON(w, funk.Values(runningOperators.ops))
		}
	}},
}}

var RunningInstanceService = &Service{map[string]*Endpoint{
	"/{handle:\\w+}/": {func(w http.ResponseWriter, r *http.Request) {
		handle := mux.Vars(r)["handle"]

		runningIns, err := runningOperators.Get(handle)
		if err != nil {
			w.WriteHeader(404)
			return
		}

		if r.Method == "POST" {
			r.ParseForm()
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)

			var idat interface{}
			err := json.Unmarshal(buf.Bytes(), &idat)

			if err != nil {
				w.WriteHeader(400)
				return
			}

			runningIns.incoming <- idat

			writeJSON(w, &runningIns)
		}

	}},
}}

var RunnerService = &Service{map[string]*Endpoint{
	"/": {Handle: func(w http.ResponseWriter, r *http.Request) {
		hub := GetHub(r)
		st := GetStorage(r)
		if r.Method == "POST" {
			var data RunState
			var ri RunInstruction

			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&ri)
			if err != nil {
				data = RunState{Status: "error", Error: &Error{Msg: err.Error(), Code: "E000X"}}
				writeJSON(w, &data)
				return
			}

			opId := ri.Id
			op, err := api.BuildAndCompile(opId, ri.Gens, ri.Props, st)
			if err != nil {
				data = RunState{Status: "error", Error: &Error{Msg: err.Error(), Code: "E000X"}}
				writeJSON(w, &data)
				return
			}

			runOp := runningOperators.Run(op)

			// --- Handle incoming and outgoing data
			go func() {
			loop:
				for {
					select {
					case outgoing := <-runOp.outgoing:
						hub.broadCastTo(Root, fmt.Sprintf("====> %v", outgoing))
					case <-runOp.outStop:
						break loop
					}
				}
			}()
			// --- Handle incoming and outgoing data [END]

			data.Status = "success"
			data.Handle = runOp.Handle
			data.URL = runOp.URL

			writeJSON(w, &data)

		} else if r.Method == "DELETE" {
			type stopInstructionJSON struct {
				Handle string `json:"handle"`
			}

			type outJSON struct {
				Status string `json:"status"`
				Error  *Error `json:"error,omitempty"`
			}

			var data outJSON

			decoder := json.NewDecoder(r.Body)
			var si stopInstructionJSON
			err := decoder.Decode(&si)
			if err != nil {
				data = outJSON{Status: "error", Error: &Error{Msg: err.Error(), Code: "E000X"}}
				writeJSON(w, &data)
				return
			}

			if err := runningOperators.Halt(si.Handle); err == nil {
				data.Status = "success"
			} else {
				data = outJSON{Status: "error", Error: &Error{Msg: "Unknown handle", Code: "E000X"}}
			}

			writeJSON(w, &data)
		}
	}},
}}
