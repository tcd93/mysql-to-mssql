package main

import (
	"time"

	dynamicstruct "github.com/ompluscator/dynamic-struct"
)

// generate StaffModel datamodel dynamically in runtime, note that fields must be public (to use with `reflect` package)
func generateStaffModel() interface{} {
	return dynamicstruct.NewStruct().
		AddField("StaffID", 0, `gorm:"column:staff_id;primaryKey"`).
		AddField("FirstName", "", `gorm:"column:first_name"`).
		AddField("LastName", "", `gorm:"column:last_name"`).
		AddField("Email", new(string), `gorm:"column:email"`).
		AddField("AddressID", 0, `gorm:"column:address_id"`).
		AddField("Picture", new([]byte), `gorm:"column:picture"`).
		AddField("StoreID", 0, `gorm:"column:store_id"`).
		AddField("Active", false, `gorm:"column:active"`).
		AddField("UserName", new(string), `gorm:"column:username"`).
		AddField("Password", new(string), `gorm:"column:password"`).
		AddField("LastUpdated", &time.Time{}, `gorm:"column:last_update"`).
		Build().
		New()
}
