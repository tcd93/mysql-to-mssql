package parser

// EventHandlerInterface contains callbacks for insert/delete/update events from MySQL source
type EventHandlerInterface interface {
	// Callback when a record is inserted into table
	// 	func(schemaName string, table string, rec interface{}) {
	// 		var model := rec.(wrapperTest) // assert type back to type of `model` (2nd param)
	// 	}
	OnInsert(schemaName string, tableName string, rec interface{})
	// Callback when a record is updated in table, refer to `OnInsert` for example
	OnUpdate(schemaName string, tableName string, oldRec interface{}, newRec interface{})
	// Callback when a record is removed from table, refer to `OnInsert` for example
	OnDelete(schemaName string, tableName string, rec interface{})
}
