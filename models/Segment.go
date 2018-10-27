package models

import "database/sql"

type Segment struct {
	TableName struct{} `sql:"segments" json:"-"`

	ID			int64			`json:"id" sql:"id"`
	Title		string			`json:"title" sql:"title"`
	ForeignID	sql.NullInt64	`json:"foreign_id"`
}
