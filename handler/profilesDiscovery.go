package handler

import (
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"sync"
)

var DiscoveryProfiles sync.Map

func StoreDiscoveryProfiles() error {
	profiles, err := models.DiscoveryProfilesAll()
	if err != nil {
		return err
	}

	for _, p := range profiles {
		DiscoveryProfiles.Store(p.ID, p)
	}

	logger.Log("Stored %d disocvery profiles", len(profiles))

	return nil
}
