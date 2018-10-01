package taskparser

import (
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/db"
	"net"
	"fmt"
	"strings"
)

/*
TODO on db triggers:
- on insert IP: discover/create apropriate network and place IP to this network
- on delete IP: check if it's network contains any other networks/ips and delete, if not AND if network is not 'manual'
-- ^ recursive for all parent networks
 */

// parse, compare, store, delete ip interfaces
func processIpifs(discovered []*dproto.Ipif, mo *handler.ManagedObject, dbo models.Object) {
	// 1: get ifnames: map[ifname]interface ; map[shortname]interface
	dbIfs, err := getIfnamesAll(dbo)
	if err != nil {
		logger.Err("%s: processIpifs: cannot get interface names: %s", dbo.Name, err.Error())
		return
	}

	// get DB ip addresses for this object.
	var dbIps []models.Ipif
	if err = db.DB.Model(&dbIps).Where(`object_id = ?`, dbo.ID).Select(); err != nil {
		logger.Err("%s: cannot select DB ip addresses for object: %s", dbo.Name, err.Error())
		return
	}
	// map[ipCidr]ipif
	dbMap := make(map[string]models.Ipif)
	for i, _ := range dbIps {
		if !strings.Contains(dbIps[i].Addr, "/") {
			dbIps[i].Addr = dbIps[i].Addr + "/32"
		}
		dbMap[dbIps[i].Addr] = dbIps[i]
	}
	discMap := make(map[string]*dproto.Ipif)


	// step 1: add discovered ipifs, that are not in db yet
	for i, _ := range discovered {
		iface := discovered[i]
		//logger.Debug("%+#v\n", discovered[i])
		dbIf, ok := dbIfs[iface.Interface]
		if !ok {
			logger.Err("%s: ipif '%s': cannot find this interface in DB interfaces!", dbo.Name, iface.Interface)
			continue
		}

		bits, err := mask2bits(iface.Mask)
		if err != nil {
			logger.Err("%s: %s", dbo.Name, err.Error())
			continue
		}
		ipstring := fmt.Sprintf("%s/%d", iface.IP, bits)
		discMap[ipstring] = iface

		// check if there is such ip in DB; compare interfaces; update if needed.
		dbip, ok := dbMap[ipstring]
		if !ok {
			// create new IP interface in DB
			newone := models.Ipif{
				Addr:ipstring,
				InterfaceID:dbIf.ID,
				ObjectID:dbo.ID,
				Type:models.IpifType_DISCOVERED.String(),
				Description:"",
			}
			logger.Update("%s: Adding IP interface %s to %s", dbo.Name, ipstring, dbIf.Name)
			if err = db.DB.Insert(&newone); err != nil {
				logger.Update("%s: Failed to insert interface %s: %s", dbo.Name, ipstring, err.Error())
				logger.Err("%s: Failed to insert interface %s: %s", dbo.Name, ipstring, err.Error())
				return
			}
			continue
		}
		if ok {
			// there is already such IPIF in db. Compare interface and fix if needed.
			if dbIf.ID != dbip.InterfaceID {
				logger.Update("%s: moving IP interface %s to interface %s", dbo.Name, ipstring, dbIf.Name)
				dbip.InterfaceID = dbIf.ID
				if err = db.DB.Update(&dbip); err != nil {
					logger.Err("%s: Failed to update IPIF interface: %s", dbo.Name, err.Error())
					logger.Update("%s: Failed to update IPIF interface: %s", dbo.Name, err.Error())
					return
				}
			}
		}
	}

	// step 2: remove DB ipifs, that was not discovered (i.e. deleted from device)
	for ipstring, ipif  := range dbMap {
		if _, ok := discMap[ipstring]; !ok {
			logger.Update("%s: removing ip interface %s from DB", dbo.Name, ipstring)
			if err = db.DB.Delete(&ipif); err != nil {
				logger.Err("%s: failed to remove ipif %s: %s", dbo.Name, ipstring, err.Error())
				logger.Update("%s: failed to remove ipif %s: %s", dbo.Name, ipstring, err.Error())
				return
			}
		}
	}
}

func mask2bits(s string) (int,error) {
	var pfx int

	mask := net.IPMask(net.ParseIP(s).To4())
	pfx, _ = mask.Size()

	if pfx == 0 {
		return 0, fmt.Errorf("Failed to parse mask %s", s)
	}

	return pfx, nil
}