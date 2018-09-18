package taskparser

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/pkg/errors"
)

// firts: we will create 2 maps [VID][INTERFACE_ID]discovered-vlan-mode and [VID][INTERFACE_ID]ObjectVlan
// second: compare them
func processVlans(discovered []*dproto.Vlan, mo *handler.ManagedObject, dbo models.Object) {
	// create discovered vlans map as VID-InterfaceID-Mode
	deviceVlans, err := getDeviceVlans(discovered, dbo)
	if err != nil {
		logger.Err("%s: %s", dbo.Name, err.Error())
		return
	}

	// create DB vlans map by VID -> intID -> objectVlan
	dbVlans, err := getDbVlans(dbo)
	if err != nil {
		logger.Err("%s: cannot select DB vlans: %s", dbo.Name, err.Error())
		return
	}

	// compare this maps...
	// first: delete DB vlans thet doesnt exists on device
	// second: if exist, compare ports and modes
	for vid, dbVlan := range dbVlans {
		devVlan, ok := deviceVlans[vid]
		if !ok {
			// delete DB vlans of this object with VID = vid
			logger.Update("%s: deleting vlan %d from object", dbo.Name, vid)
			_, err = db.DB.Model(&models.ObjectVlan{}).
				Where(`object_id = ?`, dbo.ID).Where(`vid = ?`, vid).
				Delete()
			if err != nil {
				logger.Err("%s: Failed to delete object_vlan %d: %s", dbo.Name, vid, err.Error())
				logger.Update("%s: Failed to delete object_vlan %d: %s", dbo.Name, vid, err.Error())
				return
			}
			continue
		}

		// vlan exist on device. Compare ports and modes.
		if err = comparePorts(vid, dbVlan, devVlan, dbo); err != nil {
			logger.Err(err.Error())
			return
		}
	}

	// create dbVlans that not exists
	for vid, devVlan := range deviceVlans {
		_, ok := dbVlans[vid]
		if ok {
			continue
		}

		// Vlan does not exists in DB for this object.
		// Loop over all ports and create ObjectVlans
		for ifid, mode := range devVlan {
			id, err := findOrCreateVlan(vid)
			if err != nil {
				logger.Err("%s: failed to find/create global vlan with vid=%d: %s", dbo.Name, vid, err.Error())
				return
			}
			logger.Update("%s: creating object_vlan %d for if %d", dbo.Name, vid, ifid)
			ovlan := models.ObjectVlan{
				VlanID:id,
				ObjectID:dbo.ID,
				Mode:mode,
				InterfaceID:ifid,
				VID:vid,
			}
			if err = db.DB.Insert(&ovlan); err != nil {
				logger.Update("%s: failed to create object_vlan: %s", dbo.Name, err.Error())
				logger.Err("%s: failed to create object_vlan: %s", dbo.Name, err.Error())
				return
			}
		}
	}
}

// dbVlan: map[INT_ID]ObjectVlan
// devVlan: map[INT_ID]mode
func comparePorts(vid int64, dbVlan map[int64]models.ObjectVlan, devVlan map[int64]string, dbo models.Object) error {
	// 1: loop over DB ports and find device port with same id. If none foud, delete. If found, compare/update mode.
	// 2: loop over device ports and find DB port with same id. If none, add.

	for ifid, ovlan := range dbVlan {
		mode, ok := devVlan[ifid]
		if !ok {
			// delete this from DB, because there is no this vlan on this interface on device
			logger.Update("%s: deleting vlan %d from interface %d", dbo.Name, vid, ifid)
			if err := db.DB.Delete(&ovlan); err != nil {
				logger.Update("%s: cannot delete vlan %d (id %d) from interface: %s", dbo.Name, vid, ovlan.ID, err.Error())
				return fmt.Errorf("%s: cannot delete vlan %d (id %d) from interface: %s", dbo.Name, vid, ovlan.ID, err.Error())
			}
			continue
		}
		// if mode differs, update it
		if mode != ovlan.Mode {
			ovlan.Mode = mode
			logger.Update("%s: setting vlan %d mode on iface %d to %s", dbo.Name, vid, ifid, mode)
			if err := db.DB.Update(&ovlan); err != nil {
				logger.Update("%s: error updating object_vlan: %s", dbo.Name, err.Error())
				return fmt.Errorf("%s: error updating object_vlan: %s", dbo.Name, err.Error())
			}
		}
	}


	for ifid, mode := range devVlan {
		_, ok := dbVlan[ifid]
		if !ok {
			// create new ObjectVlan
			logger.Update("%s: adding new ObjectVlan %d to interface %d", dbo.Name, vid, ifid)
			id, err := findOrCreateVlan(vid)
			if err != nil {
				logger.Update("%s: failed to find/create global vlan: %s", err.Error())
				return fmt.Errorf("%s: failed to find/create global vlan: %s", err.Error())
			}

			ovlan := models.ObjectVlan{
				VID:vid,
				InterfaceID:ifid,
				Mode:mode,
				ObjectID:dbo.ID,
				VlanID:id,
			}
			if err = db.DB.Insert(&ovlan); err != nil {
				logger.Update("%s: failed to add object_vlan %d: %s", dbo.Name, vid, err.Error())
				return fmt.Errorf("%s: failed to add object_vlan %d: %s", dbo.Name, vid, err.Error())
			}
		}
	}

	return nil
}

// return map[VID]map[INT_ID]ObjectVlan
func getDbVlans(dbo models.Object) (map[int64]map[int64]models.ObjectVlan, error) {
	result := make(map[int64]map[int64]models.ObjectVlan, 0)

	arr := make([]models.ObjectVlan, 0)
	if err := db.DB.Model(&arr).Where(`object_id = ?`, dbo.ID).Select(); err != nil {
		return result, err
	}

	for i, _ := range arr {
		vlan := arr[i]
		intMap, ok := result[vlan.VID]
		if !ok {
			intMap = make(map[int64]models.ObjectVlan, 0)
		}
		intMap[vlan.InterfaceID] = vlan
		result[vlan.VID] = intMap
	}

	return result, nil
}

// return map[VID]map[INT_ID]vlan_mode(string)
func getDeviceVlans(discovered []*dproto.Vlan, dbo models.Object) (map[int64]map[int64]string, error) {
	result := make(map[int64]map[int64]string, 0)

	// get interfaces: map[name]Interface and map[shortname]Interface
	//ifnames, err := getIfnames(dbo)
	ifnames, err := getIfnamesAll(dbo)
	if err != nil {
		return result, err
	}

	for i, _ := range discovered {
		vlan := discovered[i]
		vid := vlan.ID
		interfaces := make(map[int64]string, 0)

		for j, _ := range vlan.AccessPorts {
			ifname := vlan.AccessPorts[j]
			iface, ok := ifnames[ifname]
			if !ok {
				return result, fmt.Errorf("%d: Cannot find interface ID for '%s' (access)", vid, ifname)
			}
			interfaces[iface.ID] = models.VlanType_ACCESS.String()
		}
		for j, _ := range vlan.TrunkPorts {
			ifname := vlan.TrunkPorts[j]
			iface, ok := ifnames[ifname]
			if !ok {
				return result, fmt.Errorf("%d: Cannot find interface ID for '%s' (trunk)", vid, ifname)
			}
			interfaces[iface.ID] = models.VlanType_TRUNK.String()
		}
		result[vid] = interfaces
	}

	return result, nil
}

// return db id
// todo: name/description...
func findOrCreateVlan(vid int64) (int64, error) {
	var v models.Vlan
	err := db.DB.Model(&v).Where(`vid = ?`, vid).Select()
	if err != nil && err != pg.ErrNoRows {
		return 0, err
	}
	if err == pg.ErrNoRows {
		// create vlan
		v.Vid = vid
		logger.Update("Creating global vlan %d", vid)
		err = db.DB.Insert(&v)
		if err != nil {
			return 0, fmt.Errorf("Cannot create vlan %d: %s", vid, err.Error())
		}
		return v.ID, nil
	}

	return v.ID, nil
}

// todo: keep this somewhere, because we need this in multiple tasks during box
func getIfnames(dbo models.Object) (map[string]models.Interface, error) {
	ifNames := make(map[string]models.Interface, 0)

	ifArr := make([]models.Interface, 0)
	err := db.DB.Model(&ifArr).Where(`object_id = ?`, dbo.ID).
		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q.Where(`type = ?`, dproto.InterfaceType_PHISYCAL.String()).
				WhereOr(`type = ?`, dproto.InterfaceType_AGGREGATED.String())
			return q,nil
		}).
		Select()
	if err != nil {
		return ifNames, errors.Wrap(err, "Cannot select object interfaces")
	}
	for i, _ := range ifArr {
		ifNames[ifArr[i].Name] = ifArr[i]
		ifNames[ifArr[i].Shortname] = ifArr[i]
	}

	return ifNames, nil
}

func getIfnamesAll(dbo models.Object) (map[string]models.Interface, error) {
	ifNames := make(map[string]models.Interface, 0)

	ifArr := make([]models.Interface, 0)
	err := db.DB.Model(&ifArr).Where(`object_id = ?`, dbo.ID).
/*		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
		q.Where(`type = ?`, dproto.InterfaceType_PHISYCAL.String()).
			WhereOr(`type = ?`, dproto.InterfaceType_AGGREGATED.String()).
			WhereOr(`type = ?`, dproto.InterfaceType_SVI.String())
		return q,nil
	}).*/
		Select()
	if err != nil {
		return ifNames, errors.Wrap(err, "Cannot select object interfaces")
	}
	for i, _ := range ifArr {
		ifNames[ifArr[i].Name] = ifArr[i]
		ifNames[ifArr[i].Shortname] = ifArr[i]
	}

	return ifNames, nil
}