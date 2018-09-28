package controllers

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"strings"
)

type SearchController struct {
	HTTPController
}

func (c *SearchController) POST(ctx *HTTPContext) {
	what := ctx.Params["what"]
	switch what {
	case "object":
		c.searchObject(ctx)
		break
	default:
		returnOk(ctx.w)
		return
	}
}

func (c *SearchController) searchObject(ctx *HTTPContext) {
	objects := make([]interface{}, 0)
	result := make(map[string]interface{})

	query := strings.Trim(ctx.Params["query"],  " ")
	if query == "" {
		result["objects"] = objects
		writeJSON(ctx.w, result)
		return
	}

	var objs []models.Object
	if err := db.DB.Model(&objs).
		Where(`name like ?`, "%"+query+"%").
		WhereOr(`mgmt like ?`, "%"+query+"%").
		Order(`name`).
		Limit(20).
		Select(); err != nil {
			if err == pg.ErrNoRows {
				result["objects"] = objects
				writeJSON(ctx.w, result)
				return
			}

		returnError(ctx.w, err.Error(), true)
		return
	}

	for i := range objs {
		item := make(map[string]interface{})
		item["id"] = objs[i].ID
		item["name"] = objs[i].Name
		item["mgmt"] = objs[i].Mgmt
		item["model"] = objs[i].Model
		objects = append(objects, item)
	}

	result["objects"] = objects
	writeJSON(ctx.w, result)
}
