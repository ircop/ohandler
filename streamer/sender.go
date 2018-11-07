package streamer

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/google/uuid"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/logger"
	"github.com/sasha-s/go-deadlock"
	"time"
)

var SendLock deadlock.Mutex

func SendBox(host string, protocol dproto.Protocol, profile dproto.ProfileType, login string, password string, enable string,
	errorCB func(string), answerCB func(response dproto.BoxResponse), timeoutCB func()) {

	// unique request ID
	id, err := uuid.NewRandom()
	if err != nil {
		logger.Err("Cannot generate task uuid: %s", err.Error())
		return
	}

	var port int64 = 22
	if protocol == dproto.Protocol_TELNET {
		port = 23
	}

	message := dproto.BoxRequest{
		RequestID:	id.String(),
		// todo: do we need this tout?
		Timeout:	120,
		Login:		login,
		Password:	password,
		Profile:	profile,
		Host:		host,
		Proto:		protocol,
		Enable:		enable,
		Port:		port,
	}

	// bytes of task struct. This will be marshaled into ANY
	bts, err := proto.Marshal(&message)
	if err != nil {
		logger.Err("Cannot marshal task message: %s", err.Error())
		return
	}

	wt := WaitingBox{
		RequestID:	id.String(),
		ErrorCB:	errorCB,
		AnswerCB:	answerCB,
	}
	wt.Timer = time.AfterFunc(time.Minute * 15, timeoutCB)	// todo: configure wait timer
	BoxPool.Store(id.String(), &wt)


	packet := dproto.DPacket{
		PacketType:dproto.PacketType_BOX_REQUEST,
		Payload: &any.Any{
			TypeUrl:dproto.PacketType_BOX_REQUEST.String(),
			Value:bts,
		},
	}

	packetBts, err := proto.Marshal(&packet)
	if err != nil {
		logger.Err("Cannot marshal dproto packet: %s", err.Error())
		return
	}

	// send this task
	logger.Log("Sending box request '%s'", id.String())
	SendLock.Lock()
	defer SendLock.Unlock()
	_, err = Nats.Conn.PublishAsync(Nats.TasksChan, packetBts, func(g string, e error) {
		if e != nil {
			logger.Err("Error recieving NATS ACK: %s", e.Error())
		}
	})
	if err != nil {
		logger.Err("Failed to send NATS BOX request: %s", err.Error())
	}
}
