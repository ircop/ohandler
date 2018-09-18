package main

import (
	"flag"
	"github.com/ircop/ohandler/cfg"
	"fmt"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/streamer"
	"github.com/ircop/ohandler/tasks"
	"math/rand"
	"time"
)

/*
Start
Read config
Start logger
Connect to DB

Read profiles and remember them in-memory
Read objects
 */
func main() {
	rand.NewSource(time.Now().UnixNano())

	configPath := flag.String("c", "./ohandler.toml", "Config file location")
	flag.Parse()

	// config
	config, err := cfg.NewCfg(*configPath)
	if err != nil {
		fmt.Printf("[FATAL]: Cannot read config: %s\n", err.Error())
		return
	}

	// logger
	if err := logger.InitLogger(config.LogDebug, config.LogDir); err != nil {
		fmt.Printf("[FATAL]: %s", err.Error())
		return
	}

	// db
	if err := db.InitDB(config.DBHost, config.DBPort, config.DBName,
		config.DBUser, config.DBPassword); err != nil {
			logger.Err("Cannot initialize DB: %s", err.Error())
		}

	logger.Log("Starting object handler instance")

	if err = handler.StoreProfiles(); err != nil {
		logger.Err("Failed to store auth profiles: %s", err.Error())
		return
	}
	if err = handler.StoreDiscoveryProfiles(); err != nil {
		logger.Err("Failed to store discovery profiles: %s", err.Error())
		return
	}
	if err = streamer.Init(config.NatsURL, config.NatsReplies, config.NatsTasks); err != nil {
		logger.Err("Failed to init NATS-client: %s", err.Error())
		return
	}

	/*
	- select all object from DB
	- handle them to some struct, like 'type Object struct { mgmt: string, id: string, authProfile: string, profile: int?, timer: afterFunc ( ... discover ... )  }
	- set timers: X timeout +- some random value
	 */
	 if err = handler.StoreObjects(); err != nil {
		logger.Err("Failed to store objects: %s", err.Error())
		return
	 }

	 tasks.ScheduleObjects()

	 select{}
}
