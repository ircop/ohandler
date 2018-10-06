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
	}

	logger.Log("Stored %d objects", len(objects))

	return nil
}
