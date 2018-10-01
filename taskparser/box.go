package taskparser

import (
	//"github.com/ircop/discoverer/dproto"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/db"
	"net"
)

// ParseBoxResult parses response and updates object in DB and memory (if neded)
func ParseBoxResult(response dproto.BoxResponse, mo *handler.ManagedObject, dbo models.Object) {
	// check for global error, just in case
	//if response.Type == dproto.PacketType_ERROR {
	//	return
	//}

	// platform
	if e, ok := response.Errors[dproto.TaskType_PLATFORM.String()]; ok {
		logger.Err("%s: Error in %s: %s", dbo.Name, dproto.TaskType_PLATFORM.String(), e)
	} else {
		comparePlatform(response.Platform, mo, dbo)
	}

	// after parsing platform, DBO may be changed
	mo.MX.Lock()
	dbo = mo.DbObject
	mo.MX.Unlock()

	// Interfaces
	if e, ok := response.Errors[dproto.TaskType_INTERFACES.String()]; ok {
		logger.Err("%s: Error in %s: %s", dbo.Name, dproto.TaskType_INTERFACES.String(), e)
	} else {
		compareInterfaces(response.Interfaces, mo, dbo)
	}

	// Lldp
	if e, ok := response.Errors[dproto.TaskType_LLDP.String()]; ok {
		logger.Err("%s: Error in %s: %s", dbo.Name, dproto.TaskType_LLDP.String(), e)
	} else {
		compareLldp(response.LldpNeighbors, mo, dbo)
	}

	// Vlans
	if e, ok := response.Errors[dproto.TaskType_VLANS.String()]; ok {
		logger.Err("%s: Error in %s: %s", dbo.Name, dproto.TaskType_VLANS.String(), e)
	} else {
		processVlans(response.Vlans, mo, dbo)
	}

	// IPs
	if e, ok := response.Errors[dproto.TaskType_IPS.String()]; ok {
		logger.Err("%s: Error in %s: %s", dbo.Name, dproto.TaskType_IPS.String(), e)
	} else {
		processIpifs(response.Ipifs, mo, dbo)
	}

	// UPLINK
	if e, ok := response.Errors[dproto.TaskType_UPLINK.String()]; ok {
		logger.Err("%s: Error in %s: %s", dbo.Name, dproto.TaskType_UPLINK.String(), e)
	} else {
		processUplink(response.Uplink, mo, dbo)
	}

	if e, ok := response.Errors[dproto.TaskType_CONFIG.String()]; ok {
		logger.Err("%s: Error in %s: %s", dbo.Name, dproto.TaskType_CONFIG.String(), e)
	} else {
		processConfig(response.Config, mo, dbo)
	}
}


func comparePlatform(platform *dproto.Platform, mo *handler.ManagedObject, dbo models.Object) {
	// platform contains: model, version, revision, serial, macaddresses array
	if platform.Model == "" && platform.Serial == "" && platform.Revision == "" && platform.Serial == "" {
		// something wrong, skip it
		logger.Err("%s: skipping empty platform result", dbo.Name)
		return
	}

	// todo: should we write changes log into some db? Lets say, Clickhouse?
	mod := false
	if platform.Model != dbo.Model {
		mod = true
		//logger.Debug("%s: changing model '%s' => '%s'", dbo.Name, dbo.Model, platform.Model)
		logger.UpdateFormatted(dbo.Name, "Model", dbo.Model, platform.Model)
		dbo.Model = platform.Model
	}
	if platform.Revision != dbo.Revision {
		mod = true
		//logger.Debug("%s: changing revision '%s' => '%s'", dbo.Name, dbo.Revision, platform.Revision)
		logger.UpdateFormatted(dbo.Name, "Revision", dbo.Revision, platform.Revision)
		dbo.Revision = platform.Revision
	}
	if platform.Version != dbo.Version {
		mod = true
		//logger.Debug("%s: changing version '%s' => '%s'", dbo.Name, dbo.Version, platform.Version)
		logger.UpdateFormatted(dbo.Name, "Version", dbo.Version, platform.Version)
		dbo.Version = platform.Version
	}
	if platform.Serial != dbo.Serial {
		mod = true
		//logger.Debug("%s: changing serial '%s' => '%s'", dbo.Name, dbo.Serial, platform.Serial)
		logger.UpdateFormatted(dbo.Name, "Serial", dbo.Serial, platform.Serial)
		dbo.Serial = platform.Serial
	}

	// update if needed
	if mod {
		err := db.DB.Update(&dbo)
		if err != nil {
			logger.Err("Failed to update %s db model: %s", dbo.Name, err.Error())
			logger.Update("Failed to update %s db model: %s", dbo.Name, err.Error())
		} else {
			mo.MX.Lock()
			mo.DbObject = dbo
			mo.MX.Unlock()
			handler.Objects.Store(dbo.ID, mo)
		}
	}

	// macs...
	compareMacs(platform.Macs, mo, dbo)
}

func compareMacs(newMacsArr []string, mo *handler.ManagedObject, dbo models.Object) {
	oldMacsArr := make([]models.ObjectMac, 0)
	err := db.DB.Model(&oldMacsArr).Where(`object_id = ?`, dbo.ID).Select()
	if err != nil {
		logger.Err("Failed to select object %d macs: %s", dbo.ID, err.Error())
		return
	}

	// fill maps to simplify macs search by hash
	oldMacs := make(map[string]models.ObjectMac, len(oldMacsArr))
	newMacs := make(map[string]bool, len(newMacsArr))

	for _, mac := range oldMacsArr {
		m, e := net.ParseMAC(mac.Mac)
		if e != nil {
			logger.Err("%s: failed to parse old DB mac '%s': %s", dbo.Name, mac.Mac, e.Error())
			logger.Update("%s: failed to parse old DB mac '%s': %s", dbo.Name, mac.Mac, e.Error())
			return
		}
		oldMacs[m.String()] = mac
	}
	for _, mac := range newMacsArr {
		m, e := net.ParseMAC(mac)
		if e != nil {
			logger.Err("%s: failed to parse platform mac '%s': %s", dbo.Name, mac, e.Error())
			logger.Update("%s: failed to parse platform mac '%s': %s", dbo.Name, mac, e.Error())
			return
		}
		newMacs[m.String()] = true
	}

	// compare them
	// remove non-existing macs
	for mac, mdl := range oldMacs {
		if _, ok := newMacs[mac]; !ok {
			// remove old
			logger.Update("%s: removing mac '%s' from DB", dbo.Name, mac)
			err := db.DB.Delete(&mdl)
			if err != nil {
				logger.Err("%s: Failed to remove object mac '%s': %s", dbo.Name, mac, err.Error())
				logger.Update("%s: Failed to remove object mac '%s': %s", dbo.Name, mac, err.Error())
				return
			}
		}
	}

	// add newly discovered macs, thet are not in DB
	for mac, _ := range newMacs {
		if _, ok := oldMacs[mac]; !ok {
			// add new mac
			logger.Update("%s: adding chassis mac '%s'", dbo.Name, mac)
			mdl := models.ObjectMac{
				ObjectID:dbo.ID,
				Mac:mac,
			}
			err := db.DB.Insert(&mdl)
			if err != nil {
				logger.Err("%s: Failed to insert new chassis mac '%s': %s", dbo.Name, mac, err.Error())
				logger.Update("%s: Failed to insert new chassis mac '%s': %s", dbo.Name, mac, err.Error())
				return
			}
		}
	}
}
