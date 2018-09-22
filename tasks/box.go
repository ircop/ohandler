package tasks

import (
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/streamer"
	"math/rand"
	"time"
	"github.com/ircop/ohandler/taskparser"
	"runtime/debug"
)

// BoxDiscovery runs every time on object`s timer fires an event.
// 1) Send 'all' packet type to NATS
// 2) wait for answer (in another, upper-lewel, routine)
// 3) on answer, or on wait finished, re-schedule box discovery
func BoxDiscovery(obj *handler.ManagedObject) {
	obj.MX.Lock()
	dbo := obj.DbObject
	obj.MX.Unlock()

	logger.Debug("Running box discovery for %s (%s)", dbo.Name, dbo.Mgmt)
	if !dbo.Alive {
		logger.Log("Skipping box discovery for %s: !alive", dbo.Name)
		scheduleBox(obj, false)
		return
	}

	// get access profile
	apInt, ok := handler.AuthProfiles.Load(dbo.AuthID)
	if !ok {
		logger.Err("No auth profile for object %d found!", dbo.ID)
		scheduleBox(obj, false)
		return
	}
	ap := apInt.(models.AuthProfile)

	proto := dproto.Protocol_TELNET
	if ap.CliType.String() == "ssh" {
		proto = dproto.Protocol_SSH
	}

	// determine object profile
	profile, err := dbo.GetProfile()
	if err != nil {
		logger.Err("%s: %s", dbo.Name, err.Error())
		scheduleBox(obj, false)
		return
	}

	//proto := dproto.Protocol_TELNET
	streamer.SendTask(dproto.PacketType_ALL, dbo.Mgmt, proto, profile, ap.Login, ap.Password, ap.Enable, ap.RoCommunity,
		func(s string) {
			BoxErrorCallback(s, obj)
			scheduleBox(obj, false)
		},
		func(response dproto.Response) {
			BoxAnswerCallback(response, obj)
			scheduleBox(obj, false)
		},
		func() {
			BoxTimeoutCallback(obj)
			scheduleBox(obj, false)
		})
}

// BoxErrorCallback called when task results with global error
func BoxErrorCallback(errorText string, mo *handler.ManagedObject) {
	mo.MX.Lock()
	dbo := mo.DbObject
	mo.MX.Unlock()
	logger.Err("Got error on box discovery for %s (%s)", dbo.Name, dbo.Mgmt)
}

// BoxAnswerCallback: will be called after answer for this packet/task is recievwd
func BoxAnswerCallback (response dproto.Response, mo *handler.ManagedObject) {
	mo.MX.Lock()
	dbo := mo.DbObject
	mo.MX.Unlock()
	logger.Log("Got answer on box discovery for %s (%s): %d errors", dbo.Name, dbo.Mgmt, len(response.Errors))
	for topic, e := range response.Errors {
		logger.Log("%s: %s: %s", dbo.Name, topic, e)
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Panic("Recovered in BoxAnswerCallback for %s/%s: %+v\n%s", dbo.Name, dbo.Mgmt, r, debug.Stack())
		}
	}()

	// todo: parse box discovery answer: platform ; interfaces ; lldp ; ips ; etc......
	taskparser.ParseBoxResult(response, mo, dbo)
}

// BoxTimeoutCallback: will be called after timeout waiting for NATS task reply
func BoxTimeoutCallback (mo *handler.ManagedObject) {
	mo.MX.Lock()
	dbo := mo.DbObject
	mo.MX.Unlock()
	logger.Err("Got operation ack-timeout on box discovery for %s (%s)", dbo.Name, dbo.Mgmt)
}

// todo: re-select this object to sync just in case ( alive , etc. )
// todo: later all of this updates will not be needed, because all will be changed via this ohandler
func scheduleBox(mo *handler.ManagedObject, urgent bool) {
	mo.MX.Lock()
	dboOld := mo.DbObject
	mo.MX.Unlock()

	// todo: we should handle various DB problems and re-schedule object withoud DB
	var dbo models.Object
	err := db.DB.Model(&dbo).Where(`id = ?`, dboOld.ID).Select()
	if err != nil {
		logger.Err("Scheduler: failed to re-select object '%s': %s", dboOld.Name, err.Error())
		dbo = dboOld
	}

	dpInt, ok := handler.DiscoveryProfiles.Load(dbo.DiscoveryID)
	if !ok {
		logger.Err("WARNING! No discovery profile for object %d (%s/%s)", dbo.ID, dbo.Mgmt, dbo.Name)
		return
	}

	dp := dpInt.(models.DiscoveryProfile)
	boxInterval := time.Duration(dp.BoxInterval) * time.Second
	boxInterval = 15 * time.Second
	if urgent {
		boxInterval = 5 * time.Second
	}
	// todo: ondemand scheduling

	// re-schedule only if time is in the past or is null or time - now > boxInterval
	now := time.Now().In(Location)
	curPlanned := dbo.NextBox.In(Location)
	//logger.Debug("CUR: %+#v", curPlanned.String())
	//logger.Debug("NOW: %+#v", now.String())
	if curPlanned.Unix() <= now.Unix()+15 || time.Until(curPlanned) > (time.Duration(boxInterval)) {
		next := now.Add(boxInterval).Add(time.Duration(rand.Intn(15)) * time.Second)	// random +-15 sec. Todo: +- 2-5 min.
		dbo.NextBox = next
		err := db.DB.Update(&dbo)
		if err != nil {
			// continue scheduling, otherwise all will fail after 10-sec DB problems
			logger.Err("Failed to update next_box in db: %s", err.Error())
		} else {
			mo.MX.Lock()
			mo.DbObject = dbo
			mo.MX.Unlock()
		}
	}

	mo.MX.Lock()
	defer mo.MX.Unlock()
	//mo.BoxTimer = time.AfterFunc(boxInterval, func(){
	mo.BoxTimer = time.AfterFunc(time.Until(dbo.NextBox), func(){
		BoxDiscovery(mo)
	})

	logger.Debug("Scheduled box discovery for %s (%s) at %+#v", dbo.Name, dbo.Mgmt, dbo.NextBox.In(Location).String())
}
