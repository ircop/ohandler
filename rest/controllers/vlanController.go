package controllers

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"strconv"
	"strings"
)

type VlanController struct {
	HTTPController
}

type Intvlan struct {
	TableName struct{} `sql:"interfaces" json:"-"`

	ID 			int64	`json:"id"`
	ObjectID	int64	`json:"object_id"`
	Type		string	`json:"type"`
	Name		string	`json:"name"`
	Shortname	string	`json:"shortname"`
	Description	string	`json:"description"`
	LldpID		string	`json:"lldp_id"`
	Mode		string	`json:"mode"`
}

func (c *VlanController) GET(ctx *HTTPContext) {
	idStr := strings.Trim(ctx.Params["id"], " ")
	if idStr == "" {
		NotFound(ctx.W)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		NotFound(ctx.W)
		return
	}

	// page
	var page int64 = 1
	pgStr := strings.Trim(ctx.Params["page"], " ")
	p, err := strconv.ParseInt(pgStr, 10, 64)
	if err == nil {
		page = p
	}
	// inpage
	var inpage int64 = 30
	ipStr := strings.Trim(ctx.Params["inpage"], " ")
	p, err = strconv.ParseInt(ipStr, 10, 64)
	if err == nil {
		inpage = p
	}

	vlan := models.Vlan{}
	if err = db.DB.Model(&vlan).Where(`vid = ?`, id).First(); err != nil {
		if err == pg.ErrNoRows {
			NotFound(ctx.W)
			return
		}

		ReturnError(ctx.W, err.Error(), true)
		return
	}

	// objects:
	var offset int64 = 0
	if page > 1 {
		offset = (page - 1) * inpage
	}
	objects := make([]models.Object, 0)
	err = db.DB.Model(&objects).
		Join(`inner join object_vlans ov on ov.object_id = object.id`).
		Where(`ov.vid = ?`, id).
		Group(`object.id`).
		OrderExpr(`natsort(object.name)`).
		Limit(int(inpage)).Offset(int(offset)).
		Select()
	if err != nil {
		if err == pg.ErrNoRows {
			NotFound(ctx.W)
			return
		}
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	items := make([]interface{}, 0)
	for i := range objects {
		o := objects[i]
		item := make(map[string]interface{})
		item["object"] = o

		// select interfaces?
		intvlans := make([]Intvlan, 0)
		err = db.DB.Model(&intvlans).
			Join(`inner join object_vlans ov on intvlan.id = ov.interface_id`).
			Where(`ov.vid = ?`, id).Where(`ov.object_id = ?`, o.ID).
			Column(`intvlan.*`, `ov.mode`).
			OrderExpr(`natsort(intvlan.name)`).
			Select()
		if err != nil && err != pg.ErrNoRows {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
		item["interfaces"] = intvlans

		items = append(items, item)
	}

	result := make(map[string]interface{})
	result["items"] = items

	WriteJSON(ctx.W, result)
}