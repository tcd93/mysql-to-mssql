package main

import "time"

// StaffModel datamodel, note that fields must be public (to use with `reflect` package)
type StaffModel struct {
	StaffID     int       `gorm:"column:staff_id;primaryKey"`
	FirstName   string    `gorm:"column:first_name"`
	LastName    string    `gorm:"column:last_name"`
	AddressID   int       `gorm:"column:address_id"`
	Email       *string   `gorm:"column:email"`
	Picture     *[]byte   `gorm:"column:picture"`
	StoreID     int       `gorm:"column:store_id"`
	Active      bool      `gorm:"column:active"`
	UserName    *string   `gorm:"column:username"`
	Password    *string   `gorm:"column:password"`
	LastUpdated time.Time `gorm:"column:last_update"`
}
