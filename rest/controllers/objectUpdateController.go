package controllers

import (
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"strconv"
)

type ObjectUpdateController struct {
	HTTPController
}

func (c *ObjectUpdateController) POST(ctx *HTTPContext) {
	what, ok := ctx.Params["what"]
	if !ok {
		ReturnError(ctx.W, "Missing 'what' parameter", true)
		return
	}
	value, ok := ctx.Params["value"]
	if !ok {
		ReturnError(ctx.W, "Missing value", true)
		return
	}

	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, "Wrong ID", true)
		return
	}

	var o models.Object
	if err := db.DB.Model(&o).Where(`id = ?`, id).First(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	switch what {
	case "foreign_id":
		fid, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			ReturnError(ctx.W, "Wrong foreign ID", true)
			return
		}
		o.ForeignID.Valid = true
		o.ForeignID.Int64 = fid
		if err = db.DB.Update(&o); err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
	}

	returnOk(ctx.W)
}