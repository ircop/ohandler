package streamer

import (
	"github.com/golang/protobuf/proto"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/logger"
	nats "github.com/nats-io/go-nats-streaming"
	"github.com/sasha-s/go-deadlock"
	"sync"
	"time"
)

//var TaskPool	sync.Map
var BoxPool	sync.Map
type WaitingBox struct {
	RequestID		string
	//Type			dproto.PacketType
	Timer			*time.Timer

	ErrorCB			func(string)
	AnswerCB		func(response dproto.BoxResponse)

	MX				deadlock.Mutex
}

/*
We have recieved a message. We must:
1) read message into 'packet' type
2) ensure packetType is BOX REPLY
3) unmarshal any to box reply and work with box reply
 */
func taskReply(msg *nats.Msg) {
	defer msg.Ack()

	var packet dproto.DPacket
	err := proto.Unmarshal(msg.Data, &packet)
	if err != nil {
		logger.Err("Failed to parse dproto packet: %s", err.Error())
		return
	}

	if packet.PacketType != dproto.PacketType_BOX_REPLY {
		logger.Err("Unknown packet type")
		return
	}

	// unmarshal payload into box reply
	var reply dproto.BoxResponse
	if err = proto.Unmarshal(packet.Payload.Value, &reply); err != nil {
		logger.Err("Failed to unmarshal box response: %s", err.Error())
		return
	}

	waitingInterface, ok := BoxPool.Load(reply.ReplyID)
	if !ok {
		logger.Err("Got unknown reply ID: %s", reply.ReplyID)
		return
	}
	BoxPool.Delete(reply.ReplyID)

	waitingTask := waitingInterface.(*WaitingBox)
	waitingTask.MX.Lock()
	if waitingTask.Timer != nil {
		waitingTask.Timer.Stop()
		waitingTask.Timer = nil
	}
	waitingTask.MX.Unlock()

	if reply.Error != "" {
		if waitingTask.ErrorCB != nil {
			waitingTask.ErrorCB(reply.Error)
		}
		return
	}

	if waitingTask.AnswerCB != nil {
		waitingTask.AnswerCB(reply)
	}
}
/*
func taskReply2(msg *nats.Msg) {
	defer msg.Ack()

	var response dproto.BoxResponse
	// check if this is an error
	err := proto.Unmarshal(msg.Data, &response)
	if err != nil {
		logger.Err("Failed to unmarshal task response: %s", err.Error())
		return
	}

	id := response.ReplyID
	wtInt, ok := TaskPool.Load(id)
	if !ok {
		logger.Err("Got unknown reply id: %s", id)
		return
	}
	// remove wt from pool
	TaskPool.Delete(id)
	wt := wtInt.(*WaitingTask)
	wt.MX.Lock()
	if wt.Timer != nil {
		wt.Timer.Stop()
		wt.Timer = nil
	}
	wt.MX.Unlock()

	if response.Type == dproto.PacketType_ERROR {
		if wt.ErrorCB != nil {
			// call 'error callback'
			wt.ErrorCB(response.Error)
		}
		return
	}

	if wt.AnswerCB != nil {
		// call regular answer callback
		wt.AnswerCB(response)
		return
	}
}
*/