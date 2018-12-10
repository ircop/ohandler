package models

import (
	"time"
	"github.com/ircop/ohandler/db"
)

type Vlan struct {
	TableName struct{} `sql:"vlans" json:"-"`

	ID				int64		`json:"id"`
	Vid				int64		`json:"vid"`
	Name			string		`json:"name"`
	Description		string		`json:"description"`
	CreatedAt		*time.Time	`json:"created_at" sql:"created_at"`
}

func VlanByVID(vid int64) (Vlan, error) {
	v := Vlan{}
	err := db.DB.Model(&v).Where(`vid = ?`, vid).First()
	if err != nil {
		return v, err
	}

	return v, nil
}