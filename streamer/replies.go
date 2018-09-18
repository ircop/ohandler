package streamer

import (
	"github.com/gogo/protobuf/proto"
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/logger"
	nats "github.com/nats-io/go-nats-streaming"
	"github.com/sasha-s/go-deadlock"
	"sync"
	"time"
)

var TaskPool	sync.Map
type WaitingTask struct {
	RequestID		string
	Type			dproto.PacketType
	Timer			*time.Timer

	ErrorCB			func(string)
	AnswerCB		func(response dproto.Response)

	MX				deadlock.Mutex
}

func taskReply(msg *nats.Msg) {
	defer msg.Ack()

	var response dproto.Response
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
