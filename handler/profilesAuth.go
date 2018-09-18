package handler

import (
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"sync"
)

var AuthProfiles sync.Map

func StoreProfiles() error {
	profiles, err := models.AuthProfilesAll()
	if err != nil {
		return err
	}

	for _, p := range profiles {
		AuthProfiles.Store(p.ID, p)
	}

	logger.Log("Stored %d auth profiles", len(profiles))
	return nil
}
