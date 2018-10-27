package controllers

import (
	"crypto/rand"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
)

type ApiKeysController struct {
	HTTPController
}

func (c *ApiKeysController) GET(ctx *HTTPContext) {
	tokens := make([]models.RestToken, 0)
	err := db.DB.Model(&tokens).Where(`api = ?`, true).Select()
	if err != nil && err != pg.ErrNoRows {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["tokens"] = tokens
	WriteJSON(ctx.W, result)
}

// POST creates new API token
func (c *ApiKeysController) POST(ctx *HTTPContext) {
	b := make([]byte, 40)
	_, err := rand.Read(b)
	if nil != err {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	key := fmt.Sprintf("%x", b)

	t := models.RestToken{
		Api:true,
		Key:key,
	}
	if err := db.DB.Insert(&t); err != nil {
		ReturnError(ctx.W, err.Error(),true)
		return
	}

	returnOk(ctx.W)
}

func (c *ApiKeysController) DELETE(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, "Wrong ID", true)
		return
	}

	var t models.RestToken
	if err := db.DB.Model(&t).Where(`id = ?`, id).First(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	if err := db.DB.Delete(&t); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	returnOk(ctx.W)
}