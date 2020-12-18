package parser

import (
	"fmt"

	cn "github.com/siddontang/go-mysql/canal"
)

// Categorize events and calls EventHandlerInterface instance's OnInsert/OnUpdate/OnDelete methods,
// which would be exposed through EventHandlerWrapper so that modules outside this lib can listen to
type baseEventHandler struct {
	cn.DummyEventHandler
	// Same instance as `EventHandlerWrapper.EventHandlerWrapper`
	EventHandlerInterface
	models ModelMap
	canal  *cn.Canal
}

// Implement OnRow https://pkg.go.dev/github.com/siddontang/go-mysql/canal#EventHandler.OnRow
//
// Categorize events and calls EventHandlerInterface instance's OnInsert/OnUpdate/OnDelete methods,
// which would be exposed through EventHandlerWrapper so that modules outside this lib can listen to
func (w *baseEventHandler) OnRow(e *cn.RowsEvent) error {

	if w.models[e.Table.Name] == "" || w.models[e.Table.Name] == nil {
		return fmt.Errorf("model is nil, make sure %v is defined in data model map", e.Table.Name)
	}
	model := w.models[e.Table.Name]

	// base value for canal.DeleteAction or canal.InsertAction
	var n = 0
	var k = 1

	if e.Action == cn.UpdateAction {
		n = 1
		k = 2
	}

	for i := n; i < len(e.Rows); i += k {
		switch e.Action {
		case cn.UpdateAction:
			w.OnUpdate(e.Table.Schema, e.Table.Name, getBinLogData(e, i-1, model), getBinLogData(e, i, model))
		case cn.InsertAction:
			w.OnInsert(e.Table.Schema, e.Table.Name, getBinLogData(e, i, model))
		case cn.DeleteAction:
			w.OnDelete(e.Table.Schema, e.Table.Name, getBinLogData(e, i, model))
		default:
			fmt.Printf("Unknown action")
		}

	}
	return nil
}
