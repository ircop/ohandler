package models

import "time"

type Network struct {
	TableName struct{} `sql:"networks"`

	ID			int64		`json:"id"`
	ParentID	int64		`json:"parent_id" sql:"parent_id"`
	Network		string		`json:"network" sql:"network"`
	Type		string		`json:"type" sql:"type"`
	CreatedAt	*time.Time	`json:"created_at" sql:"created_at"`
	Description	string		`json:"description" sql:"description"`
	Children	int64		`json:"children" sql:"-"`
}
