package handler

import (
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/sasha-s/go-deadlock"
	"sync"
	"time"
)

type ManagedObject struct {
	DbObject		models.Object
	BoxTimer		*time.Timer
	MX				deadlock.Mutex
}

var Objects sync.Map

func StoreObjects() error {
	objects, err := models.ObjectsAll()
	if err != nil {
		return err
	}

	for i, _ := range objects {
		mo := ManagedObject{DbObject:objects[i]}
		Objects.Store(objects[i].ID, &mo)
		continue
/*		// skip timer setup, if: profile is non-monitored | object is not alive
		//ap, ok1 := aps[o.AuthID]
		dp, ok2 := dps[o.DiscoveryID]

		mo := ManagedObject{DbObject:o}

		if /*!ok1 || */ /*!ok2 {
			logger.Err("No auth/discovery profile for object %d (%d/%d)", o.ID, o.AuthID, o.DiscoveryID)
			Objects.Store(o.ID, &mo)
			continue
		}

		if !o.Alive || !dp.Monitored {
			Objects.Store(o.ID, &mo)
			continue
		}*/

/*		// set box discovery timer. Simplify this: just set 'from now'.
		// Todo: later we will store last-box and last-periodic times

		// Schedule BOX DISCOVERY ('all' packettype)
		boxInterval := time.Duration(dp.BoxInterval) * time.Second
		mo.BoxTimer = time.AfterFunc(boxInterval, func(){
			tasks.BoxDiscovery(&mo)
		})
*/
		//Objects.Store(o.ID, &mo)
	}

	logger.Log("Stored %d objects", len(objects))

	return nil
}
