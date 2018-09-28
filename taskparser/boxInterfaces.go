package taskparser

import (
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/logger"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
)

// todo: handle multiple interfaces with same name/shortname =\
// done: db uniques
func compareInterfaces(news map[string]*dproto.Interface, mo *handler.ManagedObject, dbo models.Object) {
	oldIfArr := make([]models.Interface, 0)
	//oldPoArr := make([]models.PoMember, 0)
	err := db.DB.Model(&oldIfArr).Where(`object_id = ?`, dbo.ID).Select()
	if err != nil && err != pg.ErrNoRows {
		logger.Err("Failed to select object %d interfaces: %s", dbo.ID, err.Error())
		return
	}

	newIfs := make(map[string]*dproto.Interface, len(news))
	for _, i := range news {
		newIfs[i.Name] = i
	}

	oldIfs := make(map[string]models.Interface, len(oldIfArr))
	for _, i := range oldIfArr {
		// check 1: loop over existing interfaces. Delete if there is no existing ifs in new ifs map.
		oldIfs[i.Name] = i
		if _, ok := newIfs[i.Name]; !ok {
			logger.Update("%s: deleting interface %s", dbo.Name, i.Name)
			if err = db.DB.Delete(&i); err != nil {
				logger.Err("Failed to delete interface #%d (%s): %s", i.ID, i.Name, err.Error())
				logger.Update("Failed to delete interface #%d (%s): %s", i.ID, i.Name, err.Error())
			}
		}
	}

	// check 2: loop over new interfaces. Add if there is no new interface in olds map.
	for name, iface := range newIfs {
		//logger.Debug("-- NEWIFS.NAME: '%s'", name)
		// todo: compare new+old interface params (descr, lldp id)
		if old, ok := oldIfs[name]; !ok {
			logger.Update("%s: adding interface %s", dbo.Name, iface.Name)

			newIf := models.Interface{
				Name:iface.Name,
				Shortname:iface.Shortname,
				ObjectID:dbo.ID,
				Type:iface.Type.String(),
				Description:iface.Description,
				LldpID:iface.LldpID,
			}
			err := db.DB.Insert(&newIf)
			if err != nil {
				logger.Err("%s: Failed to insert interface %s: %s", dbo.Name, iface.Name, err.Error())
				logger.Update("%s: Failed to insert interface %s: %s", dbo.Name, iface.Name, err.Error())

			}
		} else {
			if iface.Description != old.Description || iface.LldpID != old.LldpID {
				logger.Update("%s: updating lldpID/descr for %s", dbo.Name, iface.Name)
				old.Description = iface.Description
				old.LldpID = iface.LldpID
				if err := db.DB.Update(&old); err != nil {
					logger.Err("%s: Failed to update %s lldpID/descr: %s", dbo.Name, iface.Name, err.Error())
					logger.Update("%s: Failed to update %s lldpID/descr: %s", dbo.Name, iface.Name, err.Error())
				}
			}
		}
	}

	// PART II: parse port-channels
	parsePortchannels(news, mo, dbo)

}

// todo: handle multiple PO members with same id =\
// done: db uniques
func parsePortchannels(news map[string]*dproto.Interface, mo *handler.ManagedObject, dbo models.Object) {
	for n, _ := range news {
		iface := news[n]
		if iface.Type != dproto.InterfaceType_AGGREGATED {
			continue
		}

		/*
		1: get po interface from DB
		2: get member interfaces from DB by name/shortname
		 */

		// port-channel here
		// select interface with this name:
		var po models.Interface
		err := db.DB.Model(&po).Where(`object_id = ?`, dbo.ID).
			WhereGroup(func(q *orm.Query) (*orm.Query, error) {
				q.Where(`name = ?`, iface.Name).WhereOr(`shortname = ?`, iface.Shortname)
				return q, nil
			}).Select()
		if err != nil {
			logger.Err("%s: cannot select port-channel '%s (%s)' from DB: %s", dbo.Name, iface.Name, iface.Shortname, err.Error())
			continue
		}

		// get member interfaces from DB: they MUST be there
		discoveredMembers := make(map[int64]models.Interface)	// just for simple search in hashmap
		for i, _ := range iface.PoMembers {
			name := iface.PoMembers[i]

			// This interface should be present in DB
			var dbIface models.Interface
			err = db.DB.Model(&dbIface).Where(`object_id = ?`, dbo.ID).
				WhereGroup(func(q *orm.Query) (*orm.Query, error) {
					q.Where(`name = ?`, name).WhereOr(`shortname = ?`, name)
					return q, nil
				}).Select()
			if err != nil {
				logger.Err("%s: cannot select port-channel '%s' member iface '%s': %s", dbo.Name, iface.Name, name, err.Error())
				return
			}
			discoveredMembers[dbIface.ID] = dbIface
		}

		// got all interfaces from DB.
		// Now we should select PoMembers from DB and compare with our ones.
		DBMembers := make([]models.PoMember, len(discoveredMembers))
		err = db.DB.Model(&DBMembers).
			Where(`po_id = ?`, po.ID).Select()
		if err != nil && err != pg.ErrNoRows {
			logger.Err("%s: cannot select port-channel members for po %d: %s", dbo.Name, po.ID, err.Error())
			return
		}
		// also make map
		dbMembersMap := make(map[int64]models.PoMember, len(DBMembers))
		for i, _ := range DBMembers {
			dbMembersMap[DBMembers[i].MemberID] = DBMembers[i]
		}

		// so for now we have:
		// iface: current port-channel, as dproto.Interface
		// po: current port-channel in DB
		// discoveredMembers: map[member_interface_id]models.Interface
		// dbMembersMap[member_interface_id]PoMember
		// Next we need just compare them and make db stuff if needed

		// first we will remove non-existing members from DB
		for id, dbMember := range dbMembersMap {
			if _, ok := discoveredMembers[id]; !ok {
				// we did not discovered current member, so delete it from DB
				logger.Update("%s: removing member %d from interface %s (%d)", dbo.Name, dbMember.MemberID, iface.Name, po.ID)
				err := db.DB.Delete(&dbMember)
				if err != nil && err != pg.ErrNoRows && err != pg.ErrMultiRows {
					logger.Err("%s: failed to remove member %d from portchannel %s: %s", dbo.Name, dbMember.ID, po.Name, err.Error())
					logger.Update("%s: failed to remove member %d from portchannel %s: %s", dbo.Name, dbMember.ID, po.Name, err.Error())
					return
				}
			}
		}

		// second, we should add to DB all discovered members, that are not exist in db yet
		for id, discovered := range discoveredMembers {
			if _, ok := dbMembersMap[id]; !ok {
				logger.Update("%s: adding member %s (%d) to port-channel %s (%d)", dbo.Name, discovered.Name, discovered.ID, po.Name, po.ID)
				newmember := models.PoMember{MemberID:discovered.ID, PoID:po.ID}
				if err := db.DB.Insert(&newmember); err != nil {
					logger.Err("%s: failed to insert new po (%s/%d) member (%s/%d): %s", dbo.Name, po.Name, po.ID, discovered.Name, discovered.ID, err.Error())
					logger.Update("%s: failed to insert new po (%s/%d) member (%s/%d): %s", dbo.Name, po.Name, po.ID, discovered.Name, discovered.ID, err.Error())
					return
				}
			}
		}

		// that's all, folks! lol
		// todo: revork this stuff, its overcomplicated
	}
}
