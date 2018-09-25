package controllers

import (
	"fmt"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/tasks"
	"sort"
	"strings"
)

type DiscoveryProfileController struct {
	HTTPController
}

func (c *DiscoveryProfileController) GET(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err == nil {
		c.GetProfile(id, ctx)
		return
	}

	// get all profiles
	result := make(map[string]interface{})
	var dps []models.DiscoveryProfile
	handler.DiscoveryProfiles.Range(func(id, profInt interface{}) bool {
		dp := profInt.(models.DiscoveryProfile)
		dps = append(dps, dp)
		return true
	})

	sort.Slice(dps, func(i, j int) bool { return dps[i].Title < dps[j].Title })

	result["profiles"] = dps
	writeJSON(ctx.w, result)
}

func (c *DiscoveryProfileController) GetProfile(id int64, ctx *HTTPContext) {
	dp, ok := handler.DiscoveryProfiles.Load(id)
	if !ok {
		notFound(ctx.w)
		return
	}

	writeJSON(ctx.w, dp)
}


func (c *DiscoveryProfileController) POST(ctx *HTTPContext) {
	switch ctx.Params["what"] {
	case "save":
		c.Save(ctx)
		return
	case "add":
		c.Add(ctx)
		return
	case "delete":
		c.Delete(ctx)
		return
	default:
		returnError(ctx.w, "Unknown action", true)
		return
	}
}

func (c *DiscoveryProfileController) Delete(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong profile ID", true)
		return
	}

	cnt, err := db.DB.Model(&models.Object{}).Where(`discovery_id = ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, fmt.Sprintf("Cannot delete: there is %d objects with this discovery profile set.", cnt), true)
		return
	}

	if _, err = db.DB.Model(&models.DiscoveryProfile{}).Where(`id = ?`, id).Delete(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	handler.DiscoveryProfiles.Delete(id)

	returnOk(ctx.w)
}


func (c *DiscoveryProfileController) Save(ctx *HTTPContext) {
	required := []string{"id", "title", "monitored", "periodic_interval", "box_interval", "ping_interval"}
	if errors := c.CheckParams(ctx, required); len(errors) > 0 {
		returnError(ctx.w, fmt.Sprintf("Missing required parameters: %s", strings.Join(errors, ", ")), true)
		return
	}

	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong profile ID", true)
		return
	}
	boxInt, err := c.IntParam(ctx, "box_interval")
	if err != nil {
		returnError(ctx.w, "Wrong Box interval", true)
		return
	}
	perInt, err := c.IntParam(ctx, "periodic_interval")
	if err != nil {
		returnError(ctx.w, "Wrong Periodic interval", true)
		return
	}
	pingInt, err := c.IntParam(ctx, "ping_interval")
	if err != nil {
		returnError(ctx.w, "Wrong Ping interval", true)
		return
	}
	monitored := false
	if ctx.Params["monitored"] == "true" {
		monitored = true
	}

	// check for same title
	cnt, err := db.DB.Model(&models.DiscoveryProfile{}).Where(`title = ?`, ctx.Params["title"]).Where(`id != ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, fmt.Sprintf("There is already discovery profile named '%s'", ctx.Params["name"]), true)
		return
	}

	dpInt, ok := handler.DiscoveryProfiles.Load(id)
	if !ok {
		notFound(ctx.w)
		return
	}
	dp := dpInt.(models.DiscoveryProfile)

	oldBox := dp.BoxInterval
	dp.Title = strings.Trim(ctx.Params["title"], " ")
	dp.BoxInterval = boxInt
	dp.PeriodicInterval = perInt
	dp.PingInterval = pingInt
	dp.Monitored = monitored

	if err = db.DB.Update(&dp); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	} else {
		handler.DiscoveryProfiles.Store(dp.ID, dp)
	}

	// todo: if profile is changed, AND if discovery interval has been changed (if new interval is smaller then old one),
	// we need re-shedule stored objects.
	if oldBox > boxInt {
		var objects []models.Object
		if err := db.DB.Model(&objects).Where(`discovery_id = ?`, id).Select(); err != nil {
			returnError(ctx.w, err.Error(), true)
		}

		for i := range objects {
			moInt, ok := handler.Objects.Load(objects[i].ID)
			if !ok {
				returnError(ctx.w, fmt.Sprintf("Cannot find object %d in memory!", objects[i].ID), true)
				return
			}
			mo := moInt.(*handler.ManagedObject)
			tasks.ScheduleBox(mo, false)
		}
	}
	returnOk(ctx.w)
}

func (c *DiscoveryProfileController) Add(ctx *HTTPContext) {
	required := []string{"id", "title", "monitored", "periodic_interval", "box_interval", "ping_interval"}
	if errors := c.CheckParams(ctx, required); len(errors) > 0 {
		returnError(ctx.w, fmt.Sprintf("Missing required parameters: %s", strings.Join(errors, ", ")), true)
		return
	}

	boxInt, err := c.IntParam(ctx, "box_interval")
	if err != nil {
		returnError(ctx.w, "Wrong Box interval", true)
		return
	}
	perInt, err := c.IntParam(ctx, "periodic_interval")
	if err != nil {
		returnError(ctx.w, "Wrong Periodic interval", true)
		return
	}
	pingInt, err := c.IntParam(ctx, "ping_interval")
	if err != nil {
		returnError(ctx.w, "Wrong Ping interval", true)
		return
	}
	monitored := false
	if v, ok := ctx.Params["monitored"] ; ok && v == "true" {
		monitored = true
	}
	title := strings.Trim(ctx.Params["title"], " ")

	// check for same title
	cnt, err := db.DB.Model(&models.DiscoveryProfile{}).Where(`title = ?`, title).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, fmt.Sprintf("There is already discovery profile named '%s'", ctx.Params["name"]), true)
		return
	}

	dp := models.DiscoveryProfile{
		Monitored:monitored,
		PingInterval:pingInt,
		PeriodicInterval:perInt,
		BoxInterval:boxInt,
		Title:title,
	}

	if err = db.DB.Insert(&dp); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	handler.DiscoveryProfiles.Store(dp.ID, dp)
	returnOk(ctx.w)
}
