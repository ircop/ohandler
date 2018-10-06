package streamer

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	nats "github.com/nats-io/go-nats-streaming"
)

func PingUpdate(msg *nats.Msg) {
	defer msg.Ack()

	var packet dproto.DPacket
	err := proto.Unmarshal(msg.Data, &packet)
	if err != nil {
		logger.Err("Cannot unmarshal ping DPacket: %s", err.Error())
		return
	}

	if packet.PacketType == dproto.PacketType_PINGUPDATE {
		processUpdate(packet.Payload)
	}
}

func processUpdate(payload *any.Any) {
	var update dproto.Pingupdate
	if err := proto.Unmarshal(payload.Value, &update); err != nil {
		logger.Err("Cannot unmarshal PingUpdate payload: %s", err.Error())
		return
	}

	oid := update.OID
	alive := update.Alive
	//ts := update.Updated // todo: write/notify change time

	moInt, ok := handler.Objects.Load(oid)
	if !ok {
		// todo: send 'delete' for this object once again?
		logger.Err("Unable to find in-memory object %d", oid)
		return
	}
	mo := moInt.(*handler.ManagedObject)

	mo.MX.Lock()
	defer mo.MX.Unlock()
	dbo := mo.DbObject
	if dbo.Alive != alive {
		// update
		logger.Debug("Setting %s (#%d) state to %v", dbo.Name, oid, alive)
		dbo.Alive = alive
		if err := db.DB.Update(&dbo); err != nil {
			logger.Err("Failed to update object %s in DB: %s", dbo.Name, err.Error())
			return
		}
		mo.DbObject = dbo
	}
}
