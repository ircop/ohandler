package models

type Segment struct {
	TableName struct{} `sql:"segments" json:"-"`

	ID		int64		`json:"id" sql:"id"`
	Title	string		`json:"title" sql:"title"`
}
