package models

type Interface struct {
	TableName struct{} `sql:"interfaces"`

	ID 			int64	`json:"id"`
	ObjectID	int64	`json:"object_id"`
	Type		string	`json:"type"`
	Name		string	`json:"name"`
	Shortname	string	`json:"shortname"`
	Description	string	`json:"description"`
	LldpID		string	`json:"lldp_id"`
}
