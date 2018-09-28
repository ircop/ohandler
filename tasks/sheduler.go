package tasks

import (
	"fmt"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/models"
	"time"
)

var Location *time.Location

// ScheduleObjects makes initial scheduling for all objects
func ScheduleObjects() {
	// Load local location
	var err error
	Location, err = time.LoadLocation("Local")
	if nil != err {
		panic(fmt.Errorf("Cannot shedule anything, no local timezone: %s", err.Error()))
	}

	dps := make(map[int64]models.DiscoveryProfile)
	handler.DiscoveryProfiles.Range(func(key, val interface{}) bool {
		dp := val.(models.DiscoveryProfile)
		dps[dp.ID] = dp
		return true
	})

	handler.Objects.Range(func(key, val interface{}) bool {
		o := val.(*handler.ManagedObject)
		SheduleBox(o, false)
		return true
	})
}
