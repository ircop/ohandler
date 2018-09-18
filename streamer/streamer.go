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
}

var Nats NatsClient

//var NatsConn nats.Conn

func Init(url string, repliesChan string, tasksChan string) error {
	logger.Log("Initializing NATS client...")
	var err error

	hostname, err := os.Hostname()
	hostname = strings.Replace(hostname, ".", "-", -1) + "-client"
	if err != nil {
		return fmt.Errorf("Cannot discover hostname: %s", err.Error())
	}

	Nats.RepliesChan = repliesChan
	Nats.TasksChan = tasksChan
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

	return nil
}
