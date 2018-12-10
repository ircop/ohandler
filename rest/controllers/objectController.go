package controllers

import (
	"bitbucket.org/zombiezen/cardcpx/natsort"
	"database/sql"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/streamer"
	"github.com/ircop/ohandler/tasks"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type linkInfo struct {
	Iface			models.Interface
	Ifname			string
	LinkID			int64
	RemoteObjID		int64
	RemotePortID	int64
	RemoteName		string
	RemotePortName	string
}

type ObjectController struct {
	HTTPController
}

type objParams struct {
	Name	string
	Mgmt	string
	OsID	int64
	AuthID	int64
	DiscID	int64

	Trash	bool
}

type vlanInts struct {
	Vid		int64		`json:"vid"`
	Trunk	[]string	`json:"trunk"`
	Access	[]string	`json:"access"`
}

func (c *ObjectController) GetLinks(ctx *HTTPContext, obj models.Object) {
	result := make(map[string]interface{})

	// 1: select interfaces (we should show interfaces + theyr links)
	var ints []models.Interface
	_, err := db.DB.Query(&ints, `select * from interfaces where object_id = ? AND (type = ? OR type = ?) order by natsort(name)`, obj.ID, dproto.InterfaceType_PHISYCAL.String(), dproto.InterfaceType_AGGREGATED.String())
	if err != nil {
		ReturnError(ctx.W, err.Error(),true)
		return
	}

	var links []models.Link
	if err := db.DB.Model(&links).Where(`object1_id = ?`, obj.ID).WhereOr(`object2_id = ?`, obj.ID).Select(); err != nil {
		ReturnError(ctx.W, err.Error(),true)
		return
	}

	objIDs := make([]int64,0)
	intIDs := make([]int64,0)
	objIdName := make(map[int64]string)
	intIdName := make(map[int64]string)
	for i := range links {
		if links[i].Object1ID == obj.ID {
			objIDs = append(objIDs, links[i].Object2ID)
			intIDs = append(intIDs, links[i].Int2ID)
		} else if links[i].Object2ID == obj.ID {
			objIDs = append(objIDs, links[i].Object1ID)
			intIDs = append(intIDs, links[i].Int1ID)
		}
	}

	// select objects and interfaces
	var objects []models.Object
	var interfaces []models.Interface
	if len(links) > 0 {
		if err = db.DB.Model(&objects).Where(`id in (?)`, pg.In(objIDs)).Select(); err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
		if err = db.DB.Model(&interfaces).Where(`id in (?)`, pg.In(intIDs)).Select(); err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
		for i := range objects {
			objIdName[objects[i].ID] = objects[i].Name
		}
		for i := range interfaces {
			intIdName[interfaces[i].ID] = interfaces[i].Shortname
		}
	}

	// interface -> remote port -> remote object
	// []{ intID(local), intName(local), linkID, nei.ID, nei.int.ID, nei.Name, nei.int.Name

	ifaces := make([]interface{}, 0)
	for i := range ints {
		item := make(map[string]interface{})
		item["local_port"] = ints[i].Shortname
		item["local_port_id"] = ints[i].ID
		item["local_port_descr"] = ints[i].Description
		item["link_id"] = 0
		item["remote_port"] = ""
		item["remote_port_id"] = 0
		item["remote_object"] = ""
		item["remote_object_id"] = ""

		// loop over links; fill current link if matched
		for n := range links {
			if links[n].Int2ID == ints[i].ID {
				item["link_id"] = links[n].ID
				item["remote_port"] = intIdName[links[n].Int1ID]
				item["remote_port_id"] = links[n].Int1ID
				item["remote_object"] = objIdName[links[n].Object1ID]
				item["remote_object_id"] = links[n].Object1ID
				break
			}
			if links[n].Int1ID == ints[i].ID {
				item["link_id"] = links[n].ID
				item["remote_port"] = intIdName[links[n].Int2ID]
				item["remote_port_id"] = links[n].Int2ID
				item["remote_object"] = objIdName[links[n].Object2ID]
				item["remote_object_id"] = links[n].Object2ID
				break
			}
		}
		ifaces = append(ifaces, item)
	}

	result["ifaces"] = ifaces

	WriteJSON(ctx.W, result)
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
			ReturnError(ctx.W, err.Error(), true)
			return
		}
	intmap := make(map[int64]string)
	for i := range ints {
		intmap[ints[i].ID] = ints[i].Shortname
	}

	// select object vlans
	var ovlans []models.ObjectVlan
	if err := db.DB.Model(&ovlans).Where(`object_id = ?`, obj.ID).Order(`vid`).Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
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
			ReturnError(ctx.W, fmt.Sprintf("Cannot find interface by id (%d)", ovlans[i].InterfaceID), true)
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

	WriteJSON(ctx.W, result)
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
	_, err := db.DB.Query(&ints, `select * from interfaces where object_id = ? order by natsort(name)`, obj.ID)
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
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

	WriteJSON(ctx.W, result)
}


func (c *ObjectController) GET(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	var obj models.Object
	err = db.DB.Model(&obj).Where(`id = ?`, id).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			NotFound(ctx.W)
			return
		}
		ReturnError(ctx.W, fmt.Sprintf("Cannot select object: %s", err.Error()), true)
		return
	}

	switch ctx.Params["what"] {
	case "links":
		c.GetLinks(ctx, obj)
		return
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
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	// Count vlans
	var vlanCnt int64
	if _, err := db.DB.Query(&vlanCnt, `select count(distinct(vid)) from object_vlans where object_id=?`, obj.ID); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	var linkCnt int
	if linkCnt, err = db.DB.Model(&models.Link{}).Where(`object1_id = ?`, obj.ID).WhereOr(`object2_id = ?`, obj.ID).Count(); err != nil {
		ReturnError(ctx.W, err.Error(),true)
		return
	}

	var segs []models.ObjectSegment
	if err = db.DB.Model(&segs).Where(`object_id = ?`, obj.ID).Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	segments := make([]int64, 0)
	for i := range segs {
		segments = append(segments, segs[i].SegmentID)
	}

	result := make(map[string]interface{})
	result["segments"] = segments
	result["object"] = obj
	result["interfaces"] = intCount
	result["vlans"] = vlanCnt
	result["links"] = linkCnt
	WriteJSON(ctx.W, result)
}

// DELETE
func (c *ObjectController) DELETE(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, "Wrong object ID", true)
		return
	}

	mo, ok := handler.Objects.Load(id)
	if !ok {
		NotFound(ctx.W)
		return
	}

	dbo := mo.(*handler.ManagedObject).DbObject

	logger.Rest("Deleting object %d (%s)", id, dbo.Name)

	if err := db.DB.Delete(&dbo); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	mo.(*handler.ManagedObject).BoxTimer.Stop()
	handler.Objects.Delete(id)

	streamer.UpdateObject(dbo, true)

	returnOk(ctx.W)
}

// PUT: update
func (c *ObjectController) PUT(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, "Wrong object ID", true)
		return
	}

	foreign := sql.NullInt64{Int64:0,Valid:false}
	if fid, err := c.IntParam(ctx, "foreign_id"); err == nil && fid > 0 {
		foreign.Int64 = fid
		foreign.Valid = true
	}

	sendUpdate := false

	moInt, ok := handler.Objects.Load(id)
	if !ok {
		NotFound(ctx.W)
		return
	}
	mo := moInt.(*handler.ManagedObject)

	o := mo.DbObject
	params, err := c.checkFields(ctx)
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}


	// Check uniq name, mgmt
	cnt, err := db.DB.Model(&models.Object{}).Where(`name = ?`, params.Name).Where(`id <> ?`, id).Count()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if cnt > 0 {
		ReturnError(ctx.W, "This name is already taken", true)
		return
	}

	if params.Mgmt != "" && !params.Trash {
		cnt, err = db.DB.Model(&models.Object{}).Where(`mgmt = ?`, params.Mgmt).Where(`id <> ?`, id).Count()
		if err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
		if cnt > 0 {
			ReturnError(ctx.W, "This mgmt addr is already taken", true)
			return
		}
	}

	if o.Mgmt != params.Mgmt {
		sendUpdate = true
	}
	if o.DiscoveryID != params.DiscID {
		sendUpdate = true
	}

	o.Name = params.Name
	o.Mgmt = params.Mgmt
	o.AuthID = params.AuthID
	o.ProfileID = int32(params.OsID)
	o.DiscoveryID = params.DiscID
	o.ForeignID = foreign

	if err = db.DB.Update(&o); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	// 1) Update DBO in memory
	// 2) re-schedule
	mo.MX.Lock()
	mo.DbObject = o
	mo.MX.Unlock()

	tasks.SheduleBox(mo, false)

	if sendUpdate {
		logger.Debug("Sending update about %s", o.Name)
		go streamer.UpdateObject(o, false)
	}

	if err = c.updateSegments(ctx, o); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	returnOk(ctx.W)
}

func (c *ObjectController) parseSegmentIDs(ctx *HTTPContext) ([]int64) {
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindAllStringSubmatch(ctx.Params["segments"], -1)
	newSegs := make([]int64, 0)
	for i := range matches{
		sInt := matches[i][0]
		segID, err := strconv.ParseInt(sInt, 10, 64)
		if err == nil {
			newSegs = append(newSegs, segID)
		}
	}

	return newSegs
}

func (c *ObjectController) updateSegments(ctx *HTTPContext, dbo models.Object) error {
	// compare segments
	var oldSegs []models.ObjectSegment
	if err := db.DB.Model(&oldSegs).Where(`object_id = ?`, dbo.ID).Select(); err != nil {
		return err
	}
	/*newSegs := make([]int64, 0)
	re, err := regexp.Compile(`(\d+)`)
	if err != nil {
		return err
	}
	// segments is a string like '[123 543 567 765]', because we have parsed all params as string :(
	matches := re.FindAllStringSubmatch(ctx.Params["segments"], -1)
	for i := range matches{
		sInt := matches[i][0]
		segID, err := strconv.ParseInt(sInt, 10, 64)
		if err == nil {
			newSegs = append(newSegs, segID)
		}
	}*/
	newSegs := c.parseSegmentIDs(ctx)

	//logger.Debug("NEW SEGS: %v", newSegs)
	//logger.Debug("OLD SEGS: %+v", oldSegs)

	// remove old and add new segments
	for i := range newSegs {
		//logger.Debug(" -- new seg: %d", newSegs[i])
		found := false
		for j := range oldSegs {
			if oldSegs[j].SegmentID == newSegs[i] {
				found = true
				break
			}
		}

		if !found {
			// add new segment
			s := models.ObjectSegment{SegmentID:newSegs[i], ObjectID:dbo.ID}
			if err := db.DB.Insert(&s); err != nil {
				return err
			}
		}
	}

	for i := range oldSegs {
		found := false
		for j := range newSegs {
			if newSegs[j] == oldSegs[i].SegmentID {
				found = true
				break
			}
		}
		// remove old segment
		if !found {
			if _, err := db.DB.Model(&models.ObjectSegment{}).Where(`id = ?`, oldSegs[i].ID).Delete(); err != nil {
				return err
			}
		}
	}

	return nil
}

// PATCH: re-discover
func (c *ObjectController) PATCH(ctx *HTTPContext) {
	// re-discover
	id, err := c.IntParam(ctx, "id")
	if err != nil || id == 0 {
		ReturnError(ctx.W, "Wrong object ID", true)
		return
	}

	moInt, ok := handler.Objects.Load(id)
	if !ok {
		NotFound(ctx.W)
		return
	}

	mo := moInt.(*handler.ManagedObject)
	tasks.SheduleBox(mo, true)

	returnOk(ctx.W)
}

// ADD
func (c *ObjectController) POST(ctx *HTTPContext) {
	params, err := c.checkFields(ctx)
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	// Check uniq name, mgmt
	cnt, err := db.DB.Model(&models.Object{}).Where(`name = ?`, params.Name).Count()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if cnt > 0 {
		ReturnError(ctx.W, "This name is already taken", true)
		return
	}

	if !params.Trash {
		cnt, err = db.DB.Model(&models.Object{}).Where(`mgmt = ?`, params.Mgmt).Count()
		if err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
		if cnt > 0 {
			ReturnError(ctx.W, "This mgmt addr is already taken", true)
			return
		}
	}

	foreign := sql.NullInt64{Int64:0,Valid:false}
	if fid, err := c.IntParam(ctx, "foreign_id"); err == nil && fid > 0 {
		foreign.Int64 = fid
		foreign.Valid = true
	}

	o := models.Object{
		Name:params.Name,
		Mgmt:params.Mgmt,
		AuthID:params.AuthID,
		DiscoveryID:params.DiscID,
		ProfileID:int32(params.OsID),
		ForeignID:foreign,
	}

	if err := db.DB.Insert(&o); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	mo := handler.ManagedObject{DbObject:o}
	handler.Objects.Store(o.ID, &mo)

	tasks.SheduleBox(&mo, true)
	c.updateSegments(ctx, o)


	go streamer.UpdateObject(o, false)

	returnOk(ctx.W)
}

func (c *ObjectController) checkFields(ctx *HTTPContext) (objParams, error) {
	params := objParams{Trash:false}

	///////////
	// Check if object is in TRASH segment
	///////////
	segIDs := c.parseSegmentIDs(ctx)
	if len(segIDs) > 0 {
		// check if there is trash segment...
		segments := make([]models.Segment,0)
		//if err = db.DB.Model(&objects).Where(`id in (?)`, pg.In(objIDs)).Select(); err != nil {
		err := db.DB.Model(&segments).Where(`id in (?)`, pg.In(segIDs)).Select()
		if err != nil {
			return params, fmt.Errorf("Cannot select segments: %s", err.Error())
		}

		// is there trash segment?
		for i := range segments {
			if segments[i].Trash {
				// trash!
				params.Trash = true
				params.Mgmt = ""
				if len(segments) > 1 {
					return params, fmt.Errorf("If object is in trash segment, only one segment allowed.")
				}
			}
		}
	}


	var required []string
	if params.Trash {
		required = []string{"name", "profile_id", "auth_id", "discovery_id"}
	} else {
		required = []string{"name", "mgmt", "profile_id", "auth_id", "discovery_id"}
	}
	if missing := c.CheckParams(ctx, required); len(missing) > 0 {
		return params, fmt.Errorf("Missing required parameters: %s", strings.Join(missing, ", "))
	}

	name := strings.Trim(ctx.Params["name"], " ")
	if len(name) < 2 {
		return params, fmt.Errorf("Name length should be > 2 symbols")
	}

	if ctx.Params["mgmt"] != "" && !params.Trash {
		ip := net.ParseIP(ctx.Params["mgmt"])
		if ip == nil {
			return params, fmt.Errorf("Wrong ipv4 for mgmt addr")
		}
		params.Mgmt = ip.String()
	} else {
		params.Mgmt = ""
	}

	profileID, err := c.IntParam(ctx, "profile_id")
	if err != nil {
		return params, fmt.Errorf( "Wrong Device profile ID")
	}
	if _, ok := dproto.ProfileType_name[int32(profileID)]; !ok {
		return params, fmt.Errorf("Wrong Device profile ID (%d)", int32(profileID))
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
	//params.Mgmt = ip.String()
	params.OsID = profileID
	params.DiscID = discID
	params.AuthID = authID

	return params, nil
}

// EDIT
//func (c *ObjectController) Save(id int64, ctx *HTTPContext) {
//}
