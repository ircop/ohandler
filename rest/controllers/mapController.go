package controllers

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
)

type MapController struct {
	HTTPController
}

func (c *MapController) GET(ctx *HTTPContext) {
	sid, err := c.IntParam(ctx, "segment")
	if err != nil {
		ReturnError(ctx.W, "Wrong segment id", true)
		return
	}

	var objs []models.Object
	oids := make([]int64, 0)
	var links []models.Link
	if err := db.DB.Model(&objs).
			Join(`JOIN object_segments AS s ON s.object_id = object.id`).
			Where(`s.segment_id = ?`, sid).
			Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	for i := range objs {
		oids = append(oids, objs[i].ID)
	}

	if err := db.DB.Model(&links).
			Where(`object1_id in (?)`, pg.In(oids)).
			WhereOr(`object2_id in (?)`, pg.In(oids)).
			Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	oarr := make([]interface{}, 0)
	larr := make([]interface{}, 0)
	for i := range objs {
		o := make(map[string]interface{})
		o["id"] = objs[i].ID
		o["label"] = objs[i].Name
		oarr = append(oarr, o)
	}
	for i := range links {
		l := make(map[string]interface{})
		l["from"] = links[i].Object1ID
		l["to"] = links[i].Object2ID
		larr = append(larr, l)
	}

	result := make(map[string]interface{})
	result["objects"] = oarr
	result["links"] = larr

	WriteJSON(ctx.W, result)
}
