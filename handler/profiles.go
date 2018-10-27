package handler

import "github.com/ircop/ohandler/models"

func GetProfiles() (map[int64]models.DiscoveryProfile, map[int64]models.AuthProfile) {
	dprofiles := make(map[int64]models.DiscoveryProfile)
	DiscoveryProfiles.Range(func(key, dpInt interface{}) bool {
		dp := dpInt.(models.DiscoveryProfile)
		dprofiles[dp.ID] = dp
		return true
	})

	aprofiles := make(map[int64]models.AuthProfile)
	AuthProfiles.Range(func(key, dpInt interface{}) bool {
		ap := dpInt.(models.AuthProfile)
		aprofiles[ap.ID] = ap
		return true
	})

	return dprofiles, aprofiles
}
