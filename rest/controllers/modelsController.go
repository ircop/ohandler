package controllers

import (
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/logger"
)

/*select model from (
select distinct model from objects
where model is not null
) t
order by natsort(model)*/

type ModelsController struct {
	HTTPController
}

func (c *ModelsController) GET(ctx *HTTPContext) {
	models := make([]string, 0)
	_, err := db.DB.Query(&models, `select model from (select distinct model from objects where model is not null) t order by natsort(model)`)
	if err != nil {
		logger.RestErr("Error selecting models: %s", err.Error())
		ReturnError(ctx.W, err.Error(), true)
	}

	result := make(map[string]interface{})
	result["models"] = models

	WriteJSON(ctx.W, result)
}