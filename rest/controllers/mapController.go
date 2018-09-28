package controllers

import (
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
)

type MapController struct {
	HTTPController
}

func (c *MapController) GET(ctx *HTTPContext) {
	var objs []models.Object
	var links []models.Link
	if err := db.DB.Model(&objs).Select(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if err := db.DB.Model(&links).Select(); err != nil {
		returnError(ctx.w, err.Error(), true)
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

	writeJSON(ctx.w, result)
}
