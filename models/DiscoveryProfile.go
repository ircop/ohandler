package models

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
)

// DiscoveryProfile is struct for handling discovery profiles in db
type DiscoveryProfile struct {
	TableName struct{} `sql:"profiles_discovery"`

	ID					int64		`json:"id"`
	Title				string		`json:"title"`
	Monitored			bool		`json:"monitored"`
	BoxInterval			int64		`json:"box_interval"`
	PeriodicInterval	int64		`json:"periodic_interval"`
	PingInterval		int64		`json:"ping_interval"`
}

func DiscoveryProfilesAll() ([]DiscoveryProfile, error) {
	var models []DiscoveryProfile

	err := db.DB.Model(&models).Select()
	if err != nil && err != pg.ErrNoRows {
		return models, err
	}

	return models, nil
}