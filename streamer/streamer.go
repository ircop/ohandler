package streamer

import (
	"fmt"
	"github.com/ircop/ohandler/logger"
	nats "github.com/nats-io/go-nats-streaming"
	"os"
	"strings"
	"time"
)

type NatsClient struct {
	Conn			nats.Conn
	URL				string
	RepliesChan		string
	TasksChan		string
	DbChan			string
	SyncTimer		*time.Timer
}

var Nats NatsClient

//var NatsConn nats.Conn

func Init(url string, repliesChan string, tasksChan string, dbChan string) error {
	logger.Log("Initializing NATS client...")
	var err error

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("Cannot discover hostname: %s", err.Error())
	}
	hostname = strings.Replace(hostname, ".", "-", -1) + "-client"

	Nats.RepliesChan = repliesChan
	Nats.TasksChan = tasksChan
	Nats.DbChan = dbChan
	Nats.URL = url

	if Nats.Conn, err = nats.Connect("test-cluster", hostname, nats.NatsURL(Nats.URL)); err != nil {
		return err
	}

	// Subscribe to task-replies channel
	_, err = Nats.Conn.Subscribe(repliesChan, func(msg *nats.Msg) {
			// handle reply
			go taskReply(msg)
		},
		nats.DurableName(repliesChan),
		nats.MaxInflight(300),		// max. simultaneous updates. Todo: configure it
		nats.SetManualAckMode(),
		nats.AckWait(time.Minute * 15),	// todo: configure  this timeout
	)
	if err != nil {
		return err
	}

	// subscribe to DB channel
	_, err = Nats.Conn.Subscribe(dbChan, func(msg *nats.Msg) {
		go Nats.dbPacket(msg, dbChan)
	},
		nats.DurableName(dbChan),
		nats.MaxInflight(10),
		nats.SetManualAckMode(),
		nats.AckWait(time.Minute * 3),
	)

	// subscribe to pingupdates channel.
	// TODO: this should be made dynamicly, for each 'domain' (future)
	_, err = Nats.Conn.Subscribe("ping-default", func(msg *nats.Msg) {
		go PingUpdate(msg)
	},
		nats.DurableName("ping-default"),
		nats.MaxInflight(5000),
		nats.SetManualAckMode(),
		nats.AckWait(time.Minute * 1),
	)

	return nil
}
