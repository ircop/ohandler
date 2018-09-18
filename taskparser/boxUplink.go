package taskparser

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
)

func processUplink(uplink string, mo *handler.ManagedObject, dbo models.Object) {
	// select this interface from BD and compare with current set uplink
	if uplink == "" && dbo.UplinkID == 0 {
		return
	}
	if uplink == "" && dbo.UplinkID != 0 {
		logger.UpdateFormatted(dbo.Name, "Uplink", fmt.Sprintf("%d", dbo.UplinkID), "")
		dbo.UplinkID = 0
		err := db.DB.Update(&dbo)
		if err != nil {
			logger.Update("%s: Failed to update uplink: %s", dbo.Name, err.Error())
			logger.Err("%s: Failed to update uplink: %s", dbo.Name, err.Error())
			return
		}
		return
	}

	var iface models.Interface
	err := db.DB.Model(&iface).Where(`object_id = ?`, dbo.ID).
		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q.Where(`name = ?`, uplink).WhereOr(`shortname = ?`, uplink)
			return q, nil
		}).Select()
	if err != nil && err != pg.ErrNoRows {
		logger.Err("%s: Failed to select uplink interface '%s': %s", dbo.Name, uplink, err.Error())
		return
	}
	if err == pg.ErrNoRows {
		logger.Err("%s: Failed to find uplink interface '%s'", dbo.Name, uplink)
		return
	}

	if iface.ID != dbo.UplinkID {
		logger.UpdateFormatted(dbo.Name, "Uplink", fmt.Sprintf("%d", dbo.UplinkID), uplink)
		dbo.UplinkID = iface.ID
		err = db.DB.Update(&dbo)
		if err != nil {
			logger.Update("%s: Failed to update uplink: %s", dbo.Name, err.Error())
			logger.Err("%s: Failed to update uplink: %s", dbo.Name, err.Error())
			return
		}
	}
}
