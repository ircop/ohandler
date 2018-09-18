package models

import "time"

type Config struct {
	TableName struct{} `sql:"configs"`

	ID 			int64		`json:"id"`
	ObjectID	int64		`json:"object_id"`
	Config		string		`json:"config"`
	PrevDiff	string		`json:"prev_diff"`
	CreatedAt	*time.Time	`json:"created_at"`
}
