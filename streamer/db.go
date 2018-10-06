package streamer

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/uuid"
	"github.com/ircop/dproto"
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

// Send update event when:
// - changed mgmt addr
// - changed ping interval
// - changed alive?
func UpdatedObject(dbo models.Object, pingInterval int64, removed bool) {
	logger.Debug("Broadcasting object #%d (%s) update", dbo.ID, dbo.Name)
	o := dproto.DBObject{
		Addr:dbo.Mgmt,
		ID:dbo.ID,
		PingInterval:pingInterval,
		Alive:dbo.Alive,
		Removed:removed,
	}

	update := dproto.DBUpdate{
		Objects:[]*dproto.DBObject{&o},
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
	dprofiles := make(map[int64]models.DiscoveryProfile, 0)
	dbObjects := make([]*dproto.DBObject, 0)
	// store dprofiles into map
	handler.DiscoveryProfiles.Range(func(key, dpInt interface{}) bool {
		dp := dpInt.(models.DiscoveryProfile)
		dprofiles[dp.ID] = dp
		return true
	})

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

		obj := dproto.DBObject{
			ID:dbo.ID,
			Alive:dbo.Alive,
			Addr:dbo.Mgmt,
			PingInterval:dp.PingInterval,
			Removed:false,
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
