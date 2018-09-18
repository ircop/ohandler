package streamer

import (
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/logger"
	"github.com/sasha-s/go-deadlock"
	"time"
)

var SendLock deadlock.Mutex

func SendTask(taskType dproto.PacketType, host string, protocol dproto.Protocol, profile dproto.ProfileType,
		login string, password string, enable string,community string,
		errorCB func(string), answerCB func(dproto.Response), timeoutCB func(),
	) {

	id, err := uuid.NewRandom()
	if err != nil {
		logger.Err("Cannot generate task uuid: %s", err.Error())
		return
	}

	port := 22
	if protocol == dproto.Protocol_TELNET {
		port = 23
	}

	message := dproto.TaskRequest{
		RequestID: 	id.String(),
		Timeout:   	120,		// todo: do we need this?
		Login:     	login,
		Password:	password,
		Type:		taskType,
		Profile:	profile,
		Host:		host,
		Proto:		protocol,
		Enable:		enable,
		Port:		int32(port),
	}

	bts, err := proto.Marshal(&message)
	if err != nil {
		logger.Err("Cannot marshal task message: %s", err.Error())
		return
	}

	// Create WaitingTask? Or pass it as argument, and create upper-level?
	wt := WaitingTask{
		RequestID:id.String(),
		Type:taskType,
		ErrorCB:errorCB,
		AnswerCB:answerCB,
	}
	wt.Timer = time.AfterFunc(time.Minute * 15, timeoutCB)	// todo: configure this timer
	TaskPool.Store(id.String(), &wt)

	// send task
	SendLock.Lock()
	defer SendLock.Unlock()
	_, err = Nats.Conn.PublishAsync(Nats.TasksChan, bts, func(g string, err error){
		if err != nil {
			logger.Err("Error receiving NATS ACK: %s", err.Error())
		} else {
			logger.Debug("NATS ACK: %s", g)
		}
	})
}
