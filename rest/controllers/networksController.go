package controllers

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"net"
	"strconv"
	"strings"
)

type NetworksController struct {
	HTTPController
}

type netChildren struct {
	TableName struct{} 		`sql:"networks" json:"-"`
	ParentID	int64		`sql:"parent_id"`
	Count		int64		`sql:"cnt"`
}

func (c *NetworksController) POST(ctx *HTTPContext) {
	cidr := strings.Trim(ctx.Params["cidr"], " ")
	descr := strings.Trim(ctx.Params["description"], " ")

	if _, _, err := net.ParseCIDR(cidr); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	// search this net
	cnt, err := db.DB.Model(&models.Network{}).Where(`network = ?`, cidr).Count()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if cnt > 0 {
		ReturnError(ctx.W, fmt.Sprintf("Network %s already exist.", cidr), true)
		return
	}

	// add network
	net := models.Network{
		Network:cidr,
		Description:descr,
		Type:"MANUAL",
	}
	if err := db.DB.Insert(&net); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	returnOk(ctx.W)
}

// get children of  given ID
func (c *NetworksController) GET(ctx *HTTPContext) {
	var pid int64
	id, err := strconv.ParseInt(ctx.Params["pid"], 10, 60)
	if err == nil && id > 0 {
		pid = id
	}

	//logger.Debug("pid: %d, ctx param: %s", pid, ctx.Params["pid"])

	// load children of this network
	var nets []models.Network
	if pid > 0 {
		err = db.DB.Model(&nets).
			Where(`parent_id = ?`, pid).
			Order(`network`).Select()
	} else {
		err = db.DB.Model(&nets).
			Where(`parent_id IS NULL`).
			Where(`type = ?`, "MANUAL").
			Order(`network`).Select()
	}
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	var ids []int64
	for i := range nets {
		ids = append(ids, nets[i].ID)
	}

	var children []netChildren
	err = db.DB.Model(&children).Column(`parent_id`).ColumnExpr(`count(*) as cnt`).
		Where(`parent_id in (?)`, pg.In(ids)).
		Group(`parent_id`).Select()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	for i := range nets {
		for j := range children {
			if children[j].ParentID == nets[i].ID {
				nets[i].Children = children[j].Count
				break
			}
		}
	}

	result := make(map[string]interface{})
	result["nets"] = nets
	WriteJSON(ctx.W, result)
}
