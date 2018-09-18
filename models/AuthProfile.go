package models

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
)

// AuthProfile is struct for handling auth profiles in db
type AuthProfile struct {
	TableName struct{} `sql:"profiles_auth"`

	ID				int64		`json:"id"`
	Title			string		`json:"title"`
	Login			string		`json:"login"`
	Password		string		`json:"password"`
	Enable			string		`json:"enable"`
	RoCommunity		string		`json:"ro_community"`
	RwCommunity		string		`json:"rw_community"`
	CliType			CliType		`json:"cli_type"`
}

// AuthProfilesAll returns all profiles from DB or error
func AuthProfilesAll() ([]AuthProfile, error) {
	var models []AuthProfile
	err := db.DB.Model(&models).Select()
	if err != nil && err != pg.ErrNoRows {
		return models, err
	}

	return models, nil
}

// AuthProfilesByID selects AP by given ID and returns nil|result|error
func AuthProfilesByID(id int64) (*AuthProfile, error) {
	ap := AuthProfile{ID:id}
	err := db.DB.Model(&ap).Select()
	if err != nil && err != pg.ErrNoRows {
		return nil, err
	}

	if err == pg.ErrNoRows {
		return nil, nil
	}

	return &ap, nil
}
