package streamer

import (
	"github.com/go-pg/pg"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/uuid"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	nats "github.com/nats-io/go-nats-streaming"
	"time"
)

func (n *NatsClient) dbPacket(msg *nats.Msg, dbChan string) {
	defer msg.Ack()

	var packet dproto.DPacket
	err := proto.Unmarshal(msg.Data, &packet)
	if err != nil {
		logger.Err("Failed to parse dproto packet: %s", err.Error())
		return
	}

	// got and parsed DB packet.
	if packet.PacketType == dproto.PacketType_DB_REQUEST {
		logger.Debug("Got DBD Sync request")
		n.DbSync(dbChan)
		return
	}
}

func sendUpdate(objects []*dproto.DBObject) {
	if len(objects) < 1 {
		return
	}
	update := dproto.DBUpdate{
		Objects:objects,
	}
	bts, err := proto.Marshal(&update)
	if err != nil {
		logger.Err("Failed to marshal DB update: %s", err.Error())
		return
	}
	packet := dproto.DPacket{
		PacketType:dproto.PacketType_DB_UPDATE,
		Payload:&any.Any{
			TypeUrl:dproto.PacketType_DB_UPDATE.String(),
			Value:bts,
		},
	}
	packetBts, err := proto.Marshal(&packet)
	if err != nil {
		logger.Err("Failed to marshal DB Update packet: %s", err.Error())
		return
	}

	SendLock.Lock()
	defer SendLock.Unlock()
	_, err = Nats.Conn.PublishAsync(Nats.DbChan, packetBts, func(g string, e error) {
		if e != nil {
			logger.Err("Error recieving NATS DBUpdate ACK: %s", e.Error())
		}
	})
	if err != nil {
		logger.Err("Failed to send NATS DBUpdate: %s", err.Error())
	}
}

// Update banch of objects by ids.
// Separate this function for calling as gorouting
func UpdateObjects(objects []models.Object, removed bool) {
	dprofiles, aprofiles := handler.GetProfiles()


	DBObjects := make([]*dproto.DBObject, 0)
	for i := range objects {

		dbo := objects[i]
		ap, ok := aprofiles[dbo.AuthID]
		if !ok {
			logger.Err("Failed to find auth profile for %s (%d)", dbo.Name, dbo.AuthID)
			continue
		}
		dp, ok := dprofiles[dbo.DiscoveryID]
		if !ok {
			logger.Err("Failed to find discovery profile for %s (%d)", dbo.Name, dbo.AuthID)
			continue
		}

		interfaces := make([]*dproto.PollInterface, 0)
		o := dproto.DBObject{
			Addr:dbo.Mgmt,
			ID:dbo.ID,
			PingInterval:dp.PingInterval,
			PollInterval:dp.PeriodicInterval,
			RoCommunity:ap.RoCommunity,
			Alive:dbo.Alive,
			Removed:removed,
			Interfaces:interfaces,
		}

		ifs := make([]models.Interface, 0)
		if err := db.DB.Model(&ifs).Where(`object_id = ?`, dbo.ID).Select(); err != nil && err != pg.ErrNoRows {
			logger.Err("Cannot select object interfaces: ")
		} else {
			for i := range ifs {
				iface := dproto.PollInterface{
					ID:ifs[i].ID,
					Name:ifs[i].Name,
					Shortname:ifs[i].Shortname,
				}
				interfaces = append(interfaces, &iface)
			}
			o.Interfaces = interfaces
		}
		DBObjects = append(DBObjects, &o)
	}

	sendUpdate(DBObjects)
}

// Send update event when:
// - changed mgmt addr
// - changed ping interval
// - changed alive?
func UpdateObject(dbo models.Object, removed bool) {
	logger.Debug("Broadcasting object #%d (%s) update", dbo.ID, dbo.Name)

	dpInt, ok := handler.DiscoveryProfiles.Load(dbo.DiscoveryID)
	if !ok {
		logger.Err("UpdateObject: failed to find discovery profile for %s (%d)", dbo.Name, dbo.DiscoveryID)
		return
	}
	apInt, ok := handler.AuthProfiles.Load(dbo.AuthID)
	if !ok {
		logger.Err("UpdateObject: failed to find auth profile for %s (%d)", dbo.Name, dbo.AuthID)
		return
	}
	dp := dpInt.(models.DiscoveryProfile)
	ap := apInt.(models.AuthProfile)

	interfaces := make([]*dproto.PollInterface, 0)
	o := dproto.DBObject{
		Addr:dbo.Mgmt,
		ID:dbo.ID,
		PingInterval:dp.PingInterval,
		PollInterval:dp.PeriodicInterval,
		RoCommunity:ap.RoCommunity,
		Alive:dbo.Alive,
		Removed:removed,
		Interfaces:interfaces,
	}
	logger.Debug("SENDING DBO: %+#v", o)
	// get object interfaces
	ifs := make([]models.Interface, 0)
	if err := db.DB.Model(&ifs).Where(`object_id = ?`, dbo.ID).Select(); err != nil && err != pg.ErrNoRows {
		logger.Err("Cannot select object interfaces: ")
	} else {
		for i := range ifs {
			iface := dproto.PollInterface{
				ID:ifs[i].ID,
				Name:ifs[i].Name,
				Shortname:ifs[i].Shortname,
			}
			interfaces = append(interfaces, &iface)
		}
		o.Interfaces = interfaces
	}

	sendUpdate([]*dproto.DBObject{&o})
}

func (n *NatsClient) DbSync(dbChan string) {
	defer func() {
		// after end of sync, shedule next sync in 15 minutes.
		// todo: configure sync interval
		if n.SyncTimer != nil {
			n.SyncTimer.Stop()
			n.SyncTimer = nil
		}
		n.SyncTimer = time.AfterFunc(time.Minute * 15, func(){
			n.DbSync(dbChan)
		})
	}()

	// send all objects to channel.
	//dprofiles := make(map[int64]models.DiscoveryProfile, 0)
	dbObjects := make([]*dproto.DBObject, 0)
	// store dprofiles into map
	//handler.DiscoveryProfiles.Range(func(key, dpInt interface{}) bool {
	//	dp := dpInt.(models.DiscoveryProfile)
	//	dprofiles[dp.ID] = dp
	//	return true
	//})
	dprofiles, aprofiles := handler.GetProfiles()

	handler.Objects.Range(func(k, oInt interface{}) bool {
		mo := oInt.(*handler.ManagedObject)
		mo.MX.Lock()
		dbo := mo.DbObject
		mo.MX.Unlock()

		dp, ok := dprofiles[dbo.DiscoveryID]
		if !ok {
			logger.Err("DB SYNC: No discovery profile with id %d for %s", dbo.DiscoveryID, dbo.Name)
			return true
		}
		ap, ok := aprofiles[dbo.AuthID]
		if !ok {
			logger.Err("DB SYNC: No auth profile with id %d for %s", dbo.AuthID, dbo.Name)
			return true
		}

		interfaces := make([]*dproto.PollInterface, 0)
		obj := dproto.DBObject{
			Addr:dbo.Mgmt,
			ID:dbo.ID,
			PingInterval:dp.PingInterval,
			PollInterval:dp.PeriodicInterval,
			RoCommunity:ap.RoCommunity,
			Alive:dbo.Alive,
			Removed:false,
			Interfaces:interfaces,
		}

		// get object interfaces
		ifs := make([]models.Interface, 0)
		if err := db.DB.Model(&ifs).Where(`object_id = ?`, dbo.ID).Select(); err != nil && err != pg.ErrNoRows {
			logger.Err("Cannot select object interfaces: ")
		} else {
			for i := range ifs {
				iface := dproto.PollInterface{
					ID:ifs[i].ID,
					Name:ifs[i].Name,
					Shortname:ifs[i].Shortname,
				}
				interfaces = append(interfaces, &iface)
			}
			obj.Interfaces = interfaces
		}

		dbObjects = append(dbObjects, &obj)
		return true
	})

	// stored all objects ; send them to given channel
	id, err := uuid.NewRandom()
	if err != nil {
		logger.Err("Cannot generate uuid: %s", err.Error())
		return
	}
	reply := dproto.DBD{
		ReplyID:id.String(),
		Objects:dbObjects,
	}
	bts, err := proto.Marshal(&reply)
	if err != nil {
		logger.Err("Cannot marshal dproto DBD: %s", err.Error())
		return
	}

	packet := dproto.DPacket{
		PacketType:dproto.PacketType_DB,
		Payload:&any.Any{
			Value:bts,
			TypeUrl:dproto.PacketType_DB.String(),
		},
	}
	packetBts, err := proto.Marshal(&packet)
	if err != nil {
		logger.Err("Cannot marshal dproto DBD packet: %s", err.Error())
		return
	}

	SendLock.Lock()
	defer SendLock.Unlock()
	_, err = Nats.Conn.PublishAsync(dbChan, packetBts, func(g string, e error) {
		if e != nil {
			logger.Err("Error recieving NATS DBD ACK: %s", e.Error())
		}
	})
	if err != nil {
		logger.Err("Failed to send NATS DBD: %s", err.Error())
	}
}
