package models

import "time"

type Vlan struct {
	TableName struct{} `sql:"vlans" json:"-"`

	ID				int64		`json:"id"`
	Vid				int64		`json:"vid"`
	Name			string		`json:"name"`
	Description		string		`json:"description"`
	CreatedAt		*time.Time	`json:"created_at" sql:"created_at"`
}