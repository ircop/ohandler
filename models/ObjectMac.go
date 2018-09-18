package models

type ObjectMac struct {
	TableName struct{} `sql:"object_macs"`

	ID			int64		`json:"id"`
	ObjectID	int64		`json:"object_id"`
	Mac      	string 		`json:"mac",sql:"type:macaddr;"`
}
