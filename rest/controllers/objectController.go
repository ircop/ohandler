package controllers

import (
	"bitbucket.org/zombiezen/cardcpx/natsort"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/tasks"
	"net"
	"sort"
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

type vlanInts struct {
	Vid		int64		`json:"vid"`
	Trunk	[]string	`json:"trunk"`
	Access	[]string	`json:"access"`
}

func (c *ObjectController) GetVlans(ctx *HTTPContext, obj models.Object) {
	result := make(map[string]interface{})

	// fetch all interfaces and put them into map[id]ifname
	var ints []models.Interface
	if err := db.DB.Model(&ints).Where(`object_id = ?`, obj.ID).
		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q.Where(`type = ?`, dproto.InterfaceType_AGGREGATED.String()).
				WhereOr(`type = ?`, dproto.InterfaceType_PHISYCAL.String()).
				WhereOr(`type = ?`, dproto.InterfaceType_SVI.String())
			return q, nil
		}).
		Select(); err != nil {
			returnError(ctx.w, err.Error(), true)
			return
		}
	intmap := make(map[int64]string)
	for i := range ints {
		intmap[ints[i].ID] = ints[i].Shortname
	}

	// select object vlans
	var ovlans []models.ObjectVlan
	if err := db.DB.Model(&ovlans).Where(`object_id = ?`, obj.ID).Order(`vid`).Select(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	// vlans: []vlanItem
	// item: map[string]interface

	vlanmap := make(map[int64]vlanInts)

	for i := range ovlans {
		vlan, ok := vlanmap[ovlans[i].VID]
		if !ok {
			vlan = vlanInts{
				Vid:ovlans[i].VID,
				Trunk:make([]string,0),
				Access:make([]string,0),
			}
		}

		ifname, ok := intmap[ovlans[i].InterfaceID]
		if !ok {
			returnError(ctx.w, fmt.Sprintf("Cannot find interface by id (%d)", ovlans[i].InterfaceID), true)
			return
		}

		if ovlans[i].Mode == models.VlanType_ACCESS.String() {
			vlan.Access = append(vlan.Access, ifname)
		} else if ovlans[i].Mode == models.VlanType_TRUNK.String() {
			vlan.Trunk = append(vlan.Trunk, ifname)
		}

		vlanmap[ovlans[i].VID] = vlan
	}

	// sort vlans by vlan id
	//vlans := make([]vlanInts, 0)
	vlans := make([]interface{}, 0)
	for i, _ := range vlanmap {
		v := vlanmap[i]
		// sort ifnames
		sort.Slice(v.Trunk, func(i, j int) bool { return natsort.Less(v.Trunk[i], v.Trunk[j]) })
		sort.Slice(v.Access, func(i, j int) bool { return natsort.Less(v.Access[i], v.Access[j]) })
		item := make(map[string]interface{})
		item["vid"] = v.Vid
		item["trunk"] = strings.Join(v.Trunk, ", ")
		item["access"] = strings.Join(v.Access, ", ")
		//vlans = append(vlans, v)
		vlans = append(vlans, item)
	}

	//sort.Slice(aps, func(i, j int) bool { return aps[i].Title < aps[j].Title })
	sort.Slice(vlans, func(i, j int) bool { return vlans[i].(map[string]interface{})["vid"].(int64) < vlans[j].(map[string]interface{})["vid"].(int64) })

	result["vlans"] = vlans

	writeJSON(ctx.w, result)
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
	case "vlans":
		c.GetVlans(ctx, obj)
		return
	}

	// Count interfaces
	intCount, err := obj.GetInterfacesCount("")
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	// Count vlans
	var vlanCnt int64
	if _, err := db.DB.Query(&vlanCnt, `select count(distinct(vid)) from object_vlans where object_id=?`, obj.ID); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["object"] = obj
	result["interfaces"] = intCount
	result["vlans"] = vlanCnt
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
