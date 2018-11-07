package taskparser

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/ircop/dproto"
	"github.com/ircop/discoverer/util/mac"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/pkg/errors"
	"net"
	"regexp"
	"strings"
)

/*
Parse LldpNeighbors discovered:
1) find all this neighbors chassis/port IDs in DB (chassis ID, port ID)
2) ensure LldpNeighborship is in db
3) also remove non-existant neighbors!
- assuming ChassisID is macaddr, and PortID is either port mac or port name
 */
func compareLldp(neighbors []*dproto.LldpNeighbor, mo *handler.ManagedObject, dbo models.Object) {
	reMac, err := regexp.Compile(`^(?i:)[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}$`)
	if err != nil {
		logger.Err("Cannot compile huawei mac regex: %s", err.Error())
		return
	}

	// map that will handle found, discovered, checked, existant in DB, neighborships
	discovered := make([]models.LldpNeighbor, 0)

	// make small map [chassisID]neighborID for not to select same information from DB multiple times
	neis := make(map[string]int64)
	for i, _ := range neighbors {
		instance := neighbors[i]

		if instance.ChassisID == "" || instance.PortID == "" {
			continue
		}

		// 0: find LOCAL port
		var localPort models.Interface
		err := db.DB.Model(&localPort).Where(`object_id = ?`, dbo.ID).
			WhereGroup(func(q *orm.Query) (*orm.Query, error) {
				q.Where(`name = ?`, instance.LocalPort).
					WhereOr(`shortname = ?`, instance.LocalPort)
				return q, nil
			}).Select()
		if err != nil {
			logger.Err("%s: Failed to select local interface '%s': %s", dbo.Name, instance.LocalPort, err.Error())
			continue
		}

		// 1: try to find this chassis + port in DB
		if reMac.Match([]byte(instance.ChassisID)) {
			instance.ChassisID = strings.Replace(instance.ChassisID, "-", ".", -1)
		}

		cidMac, err := net.ParseMAC(instance.ChassisID)
		if err != nil || cidMac == nil {
			logger.Err("%s: Cannot parse neighbor chassis ID as mac (%s)", dbo.Name, instance.ChassisID)
			continue
		}

		// try to find chassis by mac in DB
		var neighborID int64
		if id, ok := neis[cidMac.String()]; ok {
			neighborID = id
		} else {
			var cmak models.ObjectMac
			if err = db.DB.Model(&cmak).Where(`mac = ?`, cidMac.String()).Select(); err != nil && err != pg.ErrNoRows {
				logger.Err("%s: Cannot select neighbor mac '%s' from object_macs db: %s", dbo.Name, cidMac.String(), err.Error())
				continue
			}
			if err != nil && err == pg.ErrNoRows {
				logger.Debug("%s: Cannot find neighbot by chassis id '%s' on port '%s'", dbo.Name, cidMac.String(), localPort.Name)
				continue
			}
			neighborID = cmak.ObjectID
		}

		portID := instance.PortID
		if Mac.IsMac(instance.PortID) {
			portID = Mac.New(instance.PortID).String()
		}
		// neighbor found in DB. Next try to find port.
		var remoteIF models.Interface
		if err = db.DB.Model(&remoteIF).Where(`object_id = ?`, neighborID).
			WhereGroup(func(q *orm.Query) (*orm.Query, error) {
				q.Where(`lldp_id = ?`, portID).
					WhereOr(`shortname = ?`, portID).
					WhereOr(`name = ?`, portID)
				return q, nil
			}).Select(); err != nil && err != pg.ErrNoRows {
				logger.Err("%s: Failed to select neighbor port '%s' in db: %s", instance.PortID, err.Error())
				continue
			}
		if err != nil && err == pg.ErrNoRows {
			logger.Debug("%s: Port id '%s' (for neighbor %d on port %s) not found in DB: %s", dbo.Name, instance.PortID, neighborID, instance.LocalPort, err.Error())
			continue
		}

		// Here we have:
		// localPort: local object interface
		// neighborID: remote neighbor ID
		// remoteIF: remote port ID
		n := models.LldpNeighbor{
			ObjectID:dbo.ID,
			LocalInterfaceID:localPort.ID,
			NeighborID:neighborID,
			NeighborInterfaceID:remoteIF.ID,
		}
		discovered = append(discovered, n)
	}

	// Now we have set of discovered LldpNeighbors.
	// We should:
	// 1: add non-existing neighbors to DB
	// 2: remove non-discovered existing neighbors from DB
	// ? todo: ? should we delete neighbor immidietly?
	// ? todo: ? Maybe we should not remove this neighborshpis at all?
	// ? todo: ? We will make _LINKS_, and LINKS should be keept up-to-dated, but not unconfirmed neighborships
	var dbNeighbors []models.LldpNeighbor
	err = db.DB.Model(&dbNeighbors).Where(`object_id = ?`, dbo.ID).Select()
	if err != nil && err != pg.ErrNoRows {
		logger.Err("%s: Failed to select existing lldp neighbors from DB: %s", dbo.Name, err.Error())
		return
	}

	// add non-existing interfaces
	for i, _ := range discovered {
		nei := discovered[i]
		found := false
		for j, _ := range dbNeighbors {
			if dbNeighbors[j].NeighborID == nei.NeighborID &&
				dbNeighbors[j].LocalInterfaceID == nei.LocalInterfaceID &&
				dbNeighbors[j].NeighborInterfaceID == nei.NeighborInterfaceID {
					found = true
					break
			}
		}
		if !found {
			logger.Update("%s: Adding new LLDP neighbor for port id %d (nei %d/port %d)", dbo.Name, nei.LocalInterfaceID, nei.NeighborID, nei.NeighborInterfaceID)
			err = db.DB.Insert(&nei)
			if err != nil {
				logger.Err("%s: Failed to insert new lldp neighbor (localport, nei, neiport = %d/%d/%d): %s", dbo.Name, nei.LocalInterfaceID, nei.NeighborID, nei.NeighborInterfaceID, err.Error())
				logger.Update("%s: Failed to insert new lldp neighbor (localport, nei, neiport = %d/%d/%d): %s", dbo.Name, nei.LocalInterfaceID, nei.NeighborID, nei.NeighborInterfaceID, err.Error())
				return
			}
		}
	}

	// todo: maybe we should add 'trash' timer for non-existing neighbors? Something like `updated_at` column.
	// todo: No. We should remove non-existing interfaces always, but ONLY IF THERE IS NOT LINKS for them.
	// todo: So first we should deal with links.
	processLinks(discovered, mo, dbo)
}

// Handle links stuff.
// What is link? Simplify, it's something like 'obj1_id, port1_id, obj2_id, port2_id'. BUT. What is obj1 and obj2?
// How we should determine which obj is '1' and which obj is '2'? :)
// This is very stupid, but looks like we should set 'obj1/obj2' randomly and select 'where ... or where ...', swapping obj1/obj2 :(
//
// Something like:
// 1: 	select lldp_neighbors, where object_id = nei.obj_id ; local_int = nei.remote_int ; remote_obj = dbo.obj ; remote_port = nei.local_port
//		i.e. select opposite lldp_neighborship.
// 2: if it doesnt exist, just 'continue'.
// 3: if exist, search actual link.
// N: select all db links for this object
func processLinks(neighbors []models.LldpNeighbor, mo *handler.ManagedObject, dbo models.Object) {
	for i, _ := range neighbors{
		//if dbo.Name == "J1" {
		//	logger.Debug("PROCESSING J1 link: %+#v", neighbors[i])
		//}
		nei := neighbors[i]
		// try to select 'remote' lldp neighbor
		// if there is no opposite LLDP record, just skip this neighbor.
		// link will be build if opposite neighbor will
		//logger.Debug("%s: searching opposite lldp neighbor for cid/pid %s/%s on port %s...", dbo.Name, )
		var remote models.LldpNeighbor
		err := db.DB.Model(&remote).Where(`neighbor_id = ?`, dbo.ID).Where(`neighbor_interface_id = ?`, nei.LocalInterfaceID).
			Where(`object_id = ?`, nei.NeighborID).Where(`local_interface_id = ?`, nei.NeighborInterfaceID).
			Select()
		if err != nil && err != pg.ErrNoRows {
			logger.Err("%s: failed to select opposite lldp neighborship for #%d: %s", dbo.Name, nei.ID, err.Error())
			continue
		}
		if err != nil && err == pg.ErrNoRows {
			continue
		}

		// if we got here, we have both local and opposite lldp instances.
		// we need to ensure that link is exist in DB, or create new link.

		// Something is selected. So we have local + remote confirmed lldp neighborship in DB.
		// 1: check if DB link exists for this interface.
		// 2: check if there is another links a) for local interface, b) for remote interface. Delete them.
		// 3: insert new link.
		var existingLink models.Link
		err = db.DB.Model(&existingLink).
			WhereGroup(func(q *orm.Query) (*orm.Query, error) {
				q.Where(`int1_id = ?`, nei.LocalInterfaceID).Where(`int2_id = ?`, nei.NeighborInterfaceID).Select()
				return q, nil
			}).
			WhereOrGroup(func(q *orm.Query) (*orm.Query, error) {
				q.Where(`int2_id = ?`, nei.LocalInterfaceID).Where(`int1_id = ?`, nei.NeighborInterfaceID).Select()
				return q, nil
			}).
			Select()

		if err != nil && err != pg.ErrNoRows {
			logger.Err("%s: failed to search existing links for local/remote interfaces '%d/%d' in db: %s", dbo.Name, nei.LocalInterfaceID, nei.NeighborInterfaceID, err.Error())
			continue
		}
		if err == nil {
			// we have existing link, nothing to do here.
			continue
		}


		//logger.Debug("%s: We have NO link between: int1_id=%d + int2=%d OR int2=%d + int1=%d", dbo.Name, nei.LocalInterfaceID, nei.NeighborInterfaceID, nei.LocalInterfaceID, nei.NeighborInterfaceID)
		//continue
		// We HAVE NO link with this ports. We will:
		// a) start transaction
		// b) find/remove all links for local port and for remote port
		// c) create link local<->remote ports
		// todo: bug. current link is also removed.
		logger.Update("%s: Adding link to %d:%d", dbo.Name, nei.NeighborID, nei.NeighborInterfaceID)
		txerr := db.DB.RunInTransaction(func(tx *pg.Tx) error {
			_, err := db.DB.Model(&models.Link{}).
				Where(`int1_id = ?`, nei.LocalInterfaceID).
				WhereOr(`int1_id = ?`, nei.NeighborInterfaceID).
				WhereOr(`int2_id = ?`, nei.LocalInterfaceID).
				WhereOr(`int2_id = ?`, nei.NeighborInterfaceID).
				Delete()
			if err != nil {
				return errors.Wrap(err, "Error deleting unconsistent links")
			}

			link := models.Link{
				Object1ID:nei.ObjectID,
				Int1ID:nei.LocalInterfaceID,
				Object2ID:nei.NeighborID,
				Int2ID:nei.NeighborInterfaceID,
				LinkType:"LLDP",
			}
			err = db.DB.Insert(&link)
			if err != nil {
				return errors.Wrap(err, "Error creating link")
			}

			return nil
		})
		if txerr != nil {
			logger.Err("%s: Failed to insert new link: %s", dbo.Name, err.Error())
			continue
		}
	}
}

