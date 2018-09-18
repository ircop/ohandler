package models

type PoMember struct {
	TableName struct{} `sql:"po_members"`

	ID			int64	`json:"id"`
	PoID		int64	`json:"po_id"`
	MemberID	int64	`json:"member_id"`
}