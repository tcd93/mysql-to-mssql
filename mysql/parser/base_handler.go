package parser

import (
	"github.com/siddontang/go-log/log"
	cn "github.com/siddontang/go-mysql/canal"
)

type baseEventHandler struct {
	cn.DummyEventHandler
	// Same instance as `EventHandlerWrapper.EventHandlerWrapper`
	EventHandlerInterface
	models ModelMap
	canal  *cn.Canal
}

// Implement OnRow https://pkg.go.dev/github.com/siddontang/go-mysql/canal#EventHandler.OnRow
func (w *baseEventHandler) OnRow(e *cn.RowsEvent) error {

	if w.models[e.Table.Name] == "" || w.models[e.Table.Name] == nil {
		log.Errorf("baseEventHandler OnRow: model is nil, make sure %v is defined in Datamodels", e.Table.Name)
		return nil
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
		new := getBinLogData(e, i, model)
		if new != nil {
			switch e.Action {
			case cn.UpdateAction:
				old := getBinLogData(e, i-1, model)
				if old != nil {
					w.OnUpdate(e.Table.Schema, e.Table.Name, old, new)
				}
			case cn.InsertAction:
				w.OnInsert(e.Table.Schema, e.Table.Name, new)
			case cn.DeleteAction:
				w.OnDelete(e.Table.Schema, e.Table.Name, new)
			default:
				log.Errorf("baseEventHandler OnRow: Unknown action")
			}
		}
	}
	return nil
}
