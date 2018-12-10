package dash

import (
	"bytes"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/rest/controllers"
	"text/template"

	//"html/template"
	"strings"
)

type ObjectController struct {
	controllers.HTTPController
}

type ObjectData struct {
	Name		string
	Mgmt		string
	Ports		string
}

func (c *ObjectController) GET(ctx *controllers.HTTPContext) {
	ID, err := c.IntParam(ctx, "id")
	if err != nil {
		controllers.ReturnError(ctx.W, "Wrong object ID", true)
		return
	}

	var dbo models.Object
	if err = db.DB.Model(&dbo).Where(`id = ?`, ID).Select(); err != nil {
		if err == pg.ErrNoRows {
			controllers.NotFound(ctx.W)
			return
		}
		controllers.ReturnError(ctx.W, err.Error(), true)
		return
	}

	data := ObjectData{
		Name:dbo.Name,
		Mgmt:dbo.Mgmt,
		Ports:"",
	}

	portParts := []string{}
	// select ports and put them into templates
	// todo: port settings like collect or not ports data
	ifaces := make([]models.Interface,0)
	if _, err = db.DB.Query(&ifaces, `select * from interfaces where object_id = ? AND (type = ? or type = ?) order by natsort(name)`, dbo.ID, dproto.InterfaceType_PHISYCAL.String(), dproto.InterfaceType_AGGREGATED.String()); err != nil {
		if err != pg.ErrNoRows {
			controllers.ReturnError(ctx.W, err.Error(), true)
			return
		}
	}
	tplPort, err := template.ParseFiles(fmt.Sprintf("%s/port-row.json", ctx.Config.DashTemplates))
	if err != nil {
		controllers.ReturnError(ctx.W, err.Error(), true)
		return
	}
	for i := range ifaces {
		var buf bytes.Buffer
		ifaces[i].Description = strings.Replace(ifaces[i].Description, `"`, `\"`, -1)
		if err = tplPort.Execute(&buf, ifaces[i]); err != nil {
			controllers.ReturnError(ctx.W, err.Error(), true)
			return
		}
		portParts = append(portParts, buf.String())
	}

	data.Ports = strings.Join(portParts, ",")

	tpl, err := template.ParseFiles(fmt.Sprintf("%s/object-panel.json", ctx.Config.DashTemplates))
	if err != nil {
		controllers.ReturnError(ctx.W, err.Error(), true)
		return
	}
	if err = tpl.Execute(ctx.W, data); err != nil {
		controllers.ReturnError(ctx.W, err.Error(), true)
		return
	}
}
