package controllers

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"github.com/pmezard/go-difflib/difflib"
	"strings"
)

type ConfigsController struct {
	HTTPController
}

func (c *ConfigsController) GET(ctx *HTTPContext) {
	oid, err := c.IntParam(ctx, "object_id")
	if err != nil {
		ReturnError(ctx.W, "Wrong object ID", true)
		return
	}

	cid, err := c.IntParam(ctx, "config_id")
	if err == nil {
		c.getConfig(cid, ctx)
		return
	}

	configs := make([]models.Config,0)
	if err = db.DB.Model(&configs).Where(`object_id = ?`, oid).
		Order(`id DESC`).
		Column(`id`, `created_at`).
		Select(); err != nil {
		if err != pg.ErrNoRows {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
	}

	result := make(map[string]interface{})
	result["configs"] = configs
	WriteJSON(ctx.W, result)
}

func (c *ConfigsController) getConfig(id int64, ctx *HTTPContext) {
	var cfg models.Config
	if err := db.DB.Model(&cfg).Where(`id = ?`, id).First(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["Config"] = cfg

	WriteJSON(ctx.W, result)
}

func (c *ConfigsController) PATCH(ctx *HTTPContext) {
	firstID, err := c.IntParam(ctx, "first_id")
	secondID, err2 := c.IntParam(ctx, "second_id")
	if err != nil || err2 != nil {
		ReturnError(ctx.W, "Wrong Config ID", true)
		return
	}

	var first models.Config
	var second models.Config
	if err = db.DB.Model(&first).Where(`id = ?`, firstID).First(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if err = db.DB.Model(&second).Where(`id = ?`, secondID).First(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	diff := difflib.ContextDiff{
		B: c.prepareConfig(first.Config),
		A: c.prepareConfig(second.Config),
		Context: 3,
		Eol: "\n",
	}

	diffstr, err := difflib.GetContextDiffString(diff)
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["firstDate"] = first.CreatedAt
	result["secondDate"] = second.CreatedAt
	result["diff"] = diffstr

	WriteJSON(ctx.W, result)
}

func (с *ConfigsController) prepareConfig(s string) []string {
	lines := strings.SplitAfter(s, "\n")
	var l2 []string
	for i := range lines {
		// stupid cisco changes configured value ntp clock-period =\
		str := strings.Trim(lines[i], " ")
		if str == "" {
			continue
		}
		l2 = append(l2, lines[i])
	}
	//lines[len(lines)-1] += "\n"
	l2[len(l2)-1] += "\n"
	return l2
}

