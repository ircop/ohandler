package controllers

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/tasks"
	"net"
	"strings"
)

type ObjectController struct {
	HTTPController
}

type objParams struct {
	Name	string
	Mgmt	string
	OsID	int64
	AuthID	int64
	DiscID	int64
}

func (c *ObjectController) GetInterfaces(ctx *HTTPContext, obj models.Object) {
	result := make(map[string]interface{})

	// select:
	// Physical/Aggregated
	// SVI / Tunnels
	// Virtual (tunnel/svi)
	// Other (loopback, null, mgmt)
	phis := make([]interface{}, 0)
	virtual := make([]interface{}, 0)
	other := make([]interface{}, 0)

	ints := make([]models.Interface, 0)
	/*err := db.DB.Model(&ints).Where(`object_id = ?`, obj.ID).
		Order(`natsort(name)`).
		Select()*/
	_, err := db.DB.Query(&ints, `select * from interfaces where object_id = ? order by natsort(name)`, obj.ID)
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	for n := range ints {
		iface := ints[n]
		item := make(map[string]interface{})
		item["id"] = iface.ID
		item["name"] = iface.Name
		item["shortname"] = iface.Shortname
		item["description"] = iface.Description
		//logger.Debug("if: %s", iface.Name)
		switch iface.Type {
		case dproto.InterfaceType_PHISYCAL.String(), dproto.InterfaceType_AGGREGATED.String(), dproto.InterfaceType_MANAGEMENT.String():
			phis = append(phis, item)
			break
		case dproto.InterfaceType_SVI.String(), dproto.InterfaceType_TUNNEL.String(), dproto.InterfaceType_LOOPBACK.String(), dproto.InterfaceType_NULL.String():
			virtual = append(virtual, item)
			break
		default:
			other = append(other, item)
			break
		}
	}

	result["phisycal"] = phis
	result["virtual"] = virtual
	result["other"] = other

	writeJSON(ctx.w, result)
}


func (c *ObjectController) GET(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	var obj models.Object
	err = db.DB.Model(&obj).Where(`id = ?`, id).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			notFound(ctx.w)
			return
		}
		returnError(ctx.w, fmt.Sprintf("Cannot select object: %s", err.Error()), true)
		return
	}

	switch ctx.Params["what"] {
	case "interfaces":
		c.GetInterfaces(ctx, obj)
		return
	}

	intCount, err := obj.GetInterfacesCount("")
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["object"] = obj
	result["interfaces"] = intCount
	writeJSON(ctx.w, result)
}

// DELETE
func (c *ObjectController) DELETE(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong object ID", true)
		return
	}

	mo, ok := handler.Objects.Load(id)
	if !ok {
		notFound(ctx.w)
		return
	}

	dbo := mo.(*handler.ManagedObject).DbObject

	logger.Rest("Deleting object %d (%s)", id, dbo.Name)

	if err := db.DB.Delete(&dbo); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	mo.(*handler.ManagedObject).BoxTimer.Stop()
	handler.Objects.Delete(id)

	returnOk(ctx.w)
}

// PUT: update
func (c *ObjectController) PUT(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong object ID", true)
		return
	}

	moInt, ok := handler.Objects.Load(id)
	if !ok {
		notFound(ctx.w)
		return
	}
	mo := moInt.(*handler.ManagedObject)

	o := mo.DbObject
	params, err := c.checkFields(ctx)
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}


	// Check uniq name, mgmt
	cnt, err := db.DB.Model(&models.Object{}).Where(`name = ?`, params.Name).Where(`id <> ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, "This name is already taken", true)
		return
	}

	cnt, err = db.DB.Model(&models.Object{}).Where(`mgmt = ?`, params.Mgmt).Where(`id <> ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, "This mgmt addr is already taken", true)
		return
	}

	o.Name = params.Name
	o.Mgmt = params.Mgmt
	o.AuthID = params.AuthID
	o.ProfileID = int32(params.OsID)
	o.DiscoveryID = params.DiscID

	if err = db.DB.Update(&o); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	// 1) Update DBO in memory
	// 2) re-schedule
	mo.MX.Lock()
	mo.DbObject = o
	mo.MX.Unlock()

	tasks.ScheduleBox(mo, false)

	returnOk(ctx.w)
}

// ADD
func (c *ObjectController) POST(ctx *HTTPContext) {
	params, err := c.checkFields(ctx)
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	// Check uniq name, mgmt
	cnt, err := db.DB.Model(&models.Object{}).Where(`name = ?`, params.Name).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, "This name is already taken", true)
		return
	}

	cnt, err = db.DB.Model(&models.Object{}).Where(`mgmt = ?`, params.Mgmt).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, "This mgmt addr is already taken", true)
		return
	}


	o := models.Object{
		Name:params.Name,
		Mgmt:params.Mgmt,
		AuthID:params.AuthID,
		DiscoveryID:params.DiscID,
		ProfileID:int32(params.OsID),
	}

	if err := db.DB.Insert(&o); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	mo := handler.ManagedObject{DbObject:o}
	handler.Objects.Store(o.ID, &mo)

	tasks.ScheduleBox(&mo, true)

	returnOk(ctx.w)
}

func (c *ObjectController) checkFields(ctx *HTTPContext) (objParams, error) {
	params := objParams{}

	required := []string{"name", "mgmt", "profile_id", "auth_id", "discovery_id"}
	if missing := c.CheckParams(ctx, required); len(missing) > 0 {
		return params, fmt.Errorf("Missing required parameters: %s", strings.Join(missing, ", "))
	}

	name := strings.Trim(ctx.Params["name"], " ")
	if len(name) < 2 {
		return params, fmt.Errorf("Name length should be > 2 symbols")
	}

	ip := net.ParseIP(ctx.Params["mgmt"])
	if ip == nil {
		return params, fmt.Errorf("Wrong ipv4 for mgmt addr")
	}

	profileID, err := c.IntParam(ctx, "profile_id")
	if err != nil {
		return params, fmt.Errorf( "Wrong Device profile ID")
	}
	if _, ok := dproto.ProfileType_name[int32(profileID)]; !ok {
		return params, fmt.Errorf("Wrong Device profile ID")
	}

	authID, err := c.IntParam(ctx, "auth_id")
	if err != nil {
		return params, fmt.Errorf("Wrong Auth profile ID")
	}
	if _, ok := handler.AuthProfiles.Load(authID); !ok {
		return params, fmt.Errorf( "Wrong Auth profile ID")
	}

	discID, err := c.IntParam(ctx, "discovery_id")
	if err != nil {
		return params, fmt.Errorf("Wrong discovery profile ID")
	}
	if _, ok := handler.DiscoveryProfiles.Load(discID); !ok {
		return params, fmt.Errorf("Wrong discovery profile ID")
	}

	params.Name = name
	params.Mgmt = ip.String()
	params.OsID = profileID
	params.DiscID = discID
	params.AuthID = authID

	return params, nil
}

// EDIT
//func (c *ObjectController) Save(id int64, ctx *HTTPContext) {
//}
