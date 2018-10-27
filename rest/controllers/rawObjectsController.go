package controllers

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
)

type RawObjectsController struct {
	HTTPController
}

func (c *RawObjectsController) GET(ctx *HTTPContext) {
	what := ctx.Params["what"];
	switch what {
	case "ipid":
		c.getIPIDs(ctx)
	}
	return
}

func (c *RawObjectsController) getIPIDs(ctx *HTTPContext) {
	objs := make([]models.Object, 0)
	if err := db.DB.Model(&objs).Column(`id`, `mgmt`, `foreign_id`).Select(); err != nil && err != pg.ErrNoRows {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	objects := make([]map[string]interface{}, 0)
	for i := range objs {
		item := make(map[string]interface{})
		item["id"] = objs[i].ID
		item["ip"] = objs[i].Mgmt
		item["foreign_id"] = objs[i].ForeignID
		objects = append(objects, item)
	}

	result["objects"] = objects
	WriteJSON(ctx.W, result)
}
