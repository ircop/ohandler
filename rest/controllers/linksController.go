package controllers

import (
	"fmt"
	"github.com/go-pg/pg/orm"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"strings"
)

type LinksController struct {
	HTTPController
}

// todo: log

func (c *LinksController) DELETE(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong link ID", true)
		return
	}

	if _, err = db.DB.Model(&models.Link{}).Where(`id = ?`, id).Delete(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	returnOk(ctx.w)
}

func (c *LinksController) POST(ctx *HTTPContext) {
	// add new link
	required := []string{"object_id", "port_id", "remote_object_id", "remote_port_id"}
	if missing := c.CheckParams(ctx, required); len(missing) > 0 {
		returnError(ctx.w, fmt.Sprintf("Missing parameters: %s", strings.Join(missing, ",")), true)
		return
	}

	oid, err := c.IntParam(ctx, "object_id")
	pid, err2 := c.IntParam(ctx, "port_id")
	roid, err3 := c.IntParam(ctx, "remote_object_id")
	rpid, err4 := c.IntParam(ctx, "remote_port_id")
	if err != nil { returnError(ctx.w, "Wrong object_id given", true); return }
	if err2 != nil { returnError(ctx.w, "Wrong port_id given", true); return }
	if err3 != nil { returnError(ctx.w, "Wrong remote_object_id given", true); return }
	if err4 != nil { returnError(ctx.w, "Wrong remote_port_id given", true); return }

	// check this objects/interfaces
	var o1 models.Object
	var o2 models.Object
	if err = db.DB.Model(&o1).Where(`id = ?`, oid).First(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if err = db.DB.Model(&o2).Where(`id = ?`, roid).First(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	var p1 models.Interface
	var p2 models.Interface
	if err = db.DB.Model(&p1).Where(`id = ?`, pid).Where(`object_id = ?`, o1.ID).First(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if err = db.DB.Model(&p2).Where(`id = ?`, rpid).Where(`object_id = ?`, o2.ID).First(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	// check that there is no such link already in DB
	cnt, err := db.DB.Model(&models.Link{}).
		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q.Where(`int1_id = ?`, p1.ID).Where(`int2_id= ? `, p2.ID)
			return q, nil
		}).
		WhereOrGroup(func(q *orm.Query) (*orm.Query, error) {
			q.Where(`int2_id = ?`, p1.ID).Where(`int1_id= ? `, p2.ID)
			return q, nil
		}).
		Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w,"There is already link on one of this ports", true)
		return
	}

	// Finally, insert link
	l := models.Link{
		Object1ID:oid,
		Int1ID:pid,
		Object2ID:roid,
		Int2ID:rpid,
		LinkType:"MANUAL",
	}
	if err = db.DB.Insert(&l); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	returnOk(ctx.w)
}