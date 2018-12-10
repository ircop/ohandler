package dash

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/rest/controllers"
	"html/template"
)

type PortController struct {
	controllers.HTTPController
}

func (c *PortController) GET(ctx *controllers.HTTPContext) {
	intID, err := c.IntParam(ctx, "interface_id")
	if err != nil {
		controllers.ReturnError(ctx.W, "Wrong interface ID", true)
		return
	}

	var iface models.Interface
	if err = db.DB.Model(&iface).Where(`id = ?`, intID).First(); err != nil {
		if err == pg.ErrNoRows {
			controllers.NotFound(ctx.W)
			return
		} else {
			controllers.ReturnError(ctx.W, err.Error(), true)
			return
		}
	}

	tpl, err := template.ParseFiles(fmt.Sprintf("%s/port-panel.json", ctx.Config.DashTemplates))
	if err != nil {
		controllers.ReturnError(ctx.W, err.Error(), true)
		return
	}

	if err = tpl.Execute(ctx.W, iface); err != nil {
		controllers.ReturnError(ctx.W, err.Error(), true)
		return
	}
	//controllers.WriteJSON(ctx.W, iface)
}
