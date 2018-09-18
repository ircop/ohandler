package models

import "time"

type Link struct {
	TableName struct{} `sql:"links"`

	ID				int64		`json:"id"`
	Object1ID		int64		`json:"object1_id" sql:"object1_id"`
	Object2ID		int64		`json:"object2_id" sql:"object2_id"`
	Int1ID			int64		`json:"int1_id" sql:"int1_id"`
	Int2ID			int64		`json:"int2_id" sql:"int2_id"`
	LinkType		string		`json:"link_type" sql:"link_type"`
	CreatedAt		*time.Time	`json:"created_at"`
}
