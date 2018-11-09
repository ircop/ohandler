package models

import (
	"database/sql"
	"fmt"
	"github.com/go-pg/pg"
	//"github.com/ircop/discoverer/dproto"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/db"
	"time"
)

type Object struct {
	TableName struct{} `sql:"objects" json:"-"`

	ID			int64		`json:"id"`
	Mgmt		string		`json:"mgmt"`
	Name		string		`json:"name"`
	AuthID		int64		`json:"auth_id"`
	DiscoveryID	int64		`json:"discovery_id"`
	OsID		int64		`json:"os_id"`
	CreatedAT	time.Time	`json:"created_at"`
	DeletedAT	time.Time	`json:"deleted_at"`
	Alive		bool		`json:"alive"`
	ProfileID	int32		`json:"profile_id" sql:",notnull"`

	Model		string		`json:"model"`
	Revision	string		`json:"revision"`
	Version		string		`json:"version"`
 	Serial		string		`json:"serial"`
	UplinkID	int64		`json:"uplink_id", sql:"uplink_id"`
	ForeignID	sql.NullInt64	`json:"foreign_id"`

	NextBox		time.Time	`json:"next_box"`
}

func (o *Object) GetInterfacesCount(intType string) (int, error) {
	q := db.DB.Model(&Interface{}).Where(`object_id = ?`, o.ID)
	if intType != "" {
		q.Where(`type = ?`, intType)
	}

	return q.Count()
}

func ObjectsAll() ([]Object, error) {
	var objects []Object
	err := db.DB.Model(&objects).
		//Where(`mgmt = ?`, `109.206.159.150`).WhereOr(`mgmt = ?`, `172.31.31.199`).
		Select()
	if err != nil && err != pg.ErrNoRows {
		return objects, err
	}

	return objects, nil
}

func (o *Object) GetProfile() (dproto.ProfileType, error) {
	if _, ok := dproto.ProfileType_name[o.ProfileID]; !ok {
		return dproto.ProfileType(0), fmt.Errorf("Profile %d not found", o.ProfileID)
	}

	return dproto.ProfileType(o.ProfileID), nil
}
