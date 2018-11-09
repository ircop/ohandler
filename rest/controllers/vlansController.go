package controllers

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"strconv"
	"strings"
)

type VlansController struct {
	HTTPController
}

type vlanInstance struct {
	ID		int64		`json:"id"`
	VID		int64		`json:"vid"`
	Ints	int64		`json:"interfaces"`
}

type vlanObjects struct {
	TableName struct{} 		`sql:"object_vlans" json:"-"`
	VID		int64		`sql:"vid"`
	Objects	int64		`sql:"cnt"`
}

func (c *VlansController) searchVlan(str string, ctx *HTTPContext) {
	var vlans []models.Vlan

	q := db.DB.Model(&vlans)
	q.Where(`name like ?`, "%"+str+"%")
	q.WhereOr(`description like ?`, "%"+str+"%")
	if num, err := strconv.ParseInt(str, 10, 64); err == nil && num > 0 && num < 4096 {
		q.WhereOr(`vid = ?`, num)
	}
	q.Order(`vid`)
	if err := q.Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})

	ids := make([]int64,0)
	for i := range vlans {
		ids = append(ids, vlans[i].ID)
	}
	var vObjs []vlanObjects
	if err := db.DB.Model(&vObjs).Column(`vid`).ColumnExpr(`count(distinct(object_id)) as cnt`).
		Where(`vlan_id in (?)`, pg.In(ids)).
		Group(`vid`).
		Select(); err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}

	rows := make([]map[string]interface{}, 0)
	for i := range vlans {
		item := make(map[string]interface{})
		item["id"] = vlans[i].ID
		item["vid"] = vlans[i].Vid
		item["name"] = vlans[i].Name
		item["descr"] = vlans[i].Description
		item["objects"] = 0

		for n := range vObjs {
			if vObjs[n].VID == vlans[i].Vid {
				item["objects"] = vObjs[n].Objects
			}
		}
		rows = append(rows, item)
	}
	result["rows"] = rows
	result["total"] = len(vlans)

	WriteJSON(ctx.W, result)
}

func (c *VlansController) GET(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err == nil {
		c.getVlan(id, ctx)
		return
	}

	if searchstring := strings.Trim(ctx.Params["searchstring"],  " "); searchstring != "" {
		c.searchVlan(searchstring, ctx)
		return
	}

	// return all vlans
	var limit int64 = 10
	var page int64
	var offset int64
	var vlans []models.Vlan

	if psize, ok := ctx.Params["pagesize"]; ok {
		pageSize, err := strconv.ParseInt(psize, 10, 64)
		if err == nil {
			limit = pageSize
		}
	}

	if pageStr, ok := ctx.Params["page"]; ok {
		pageNum, err := strconv.ParseInt(pageStr, 10, 64)
		if err == nil {
			page = pageNum
			if page > 1 {
				offset = (page -1) * limit
			}
		}
	}

	// total count:
	cnt, err := db.DB.Model(&models.Vlan{}).Count()
	if err != nil {
		logger.RestErr("Cannot select vlans count: %s", err.Error())
		InternalError(ctx.W, err.Error())
		return
	}
	_, err = db.DB.Query(&vlans, `select * from vlans order by vid limit ? offset ?`, limit, offset)
	if err != nil {
		logger.RestErr("Error selecting vlans: %s", err.Error())
		InternalError(ctx.W, err.Error())
		return
	}

	ids := make([]int64,0)
	for i := range vlans {
		ids = append(ids, vlans[i].ID)
	}

	var vObjs []vlanObjects
	if err := db.DB.Model(&vObjs).Column(`vid`).ColumnExpr(`count(distinct(object_id)) as cnt`).
		Where(`vlan_id in (?)`, pg.In(ids)).
		Group(`vid`).
		Select(); err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
	}

	rows := make([]map[string]interface{}, 0)
	for i := range vlans {
		item := make(map[string]interface{})
		item["id"] = vlans[i].ID
		item["vid"] = vlans[i].Vid
		item["name"] = vlans[i].Name
		item["descr"] = vlans[i].Description
		item["objects"] = 0

		for j := range vObjs {
			if vObjs[j].VID == vlans[i].Vid {
				item["objects"] = vObjs[j].Objects
				break
			}
		}
		rows = append(rows, item)
	}

	result := make(map[string]interface{})
	result["total"] = cnt
	result["rows"] = rows

	WriteJSON(ctx.W, result)
}

func (c *VlansController) getVlan(id int64, ctx *HTTPContext) {
	//
}