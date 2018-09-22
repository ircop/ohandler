package controllers

import (
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/db"
	"github.com/go-pg/pg"
	"fmt"
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/logger"
)

type ObjectController struct {
	HTTPController
}

func (c *ObjectController) GetInterfaces(ctx *HTTPContext, obj models.Object) {
	result := make(map[string]interface{})

	// select:
	// Physical/Aggregated
	// SVI / Tunnels
	// Virtual (tunnel/svi)
	// Other (loopback, null, mgmt)
	phis := make([]interface{}, 0)
	virtual := make([]interface{}, 0)
	other := make([]interface{}, 0)

	ints := make([]models.Interface, 0)
	/*err := db.DB.Model(&ints).Where(`object_id = ?`, obj.ID).
		Order(`natsort(name)`).
		Select()*/
	_, err := db.DB.Query(&ints, `select * from interfaces where object_id = ? order by natsort(name)`, obj.ID)
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	for n := range ints {
		iface := ints[n]
		item := make(map[string]interface{})
		item["id"] = iface.ID
		item["name"] = iface.Name
		item["shortname"] = iface.Shortname
		item["description"] = iface.Description
		logger.Debug("if: %s", iface.Name)
		switch iface.Type {
		case dproto.InterfaceType_PHISYCAL.String(), dproto.InterfaceType_AGGREGATED.String(), dproto.InterfaceType_MANAGEMENT.String():
			phis = append(phis, item)
			break
		case dproto.InterfaceType_SVI.String(), dproto.InterfaceType_TUNNEL.String(), dproto.InterfaceType_LOOPBACK.String(), dproto.InterfaceType_NULL.String():
			virtual = append(virtual, item)
			break
		default:
			other = append(other, item)
			break
		}
	}

	result["phisycal"] = phis
	result["virtual"] = virtual
	result["other"] = other

	writeJSON(ctx.w, result)
}

/*func (c *ObjectController) selectInterfaces(intType string, objectID int64) ([]interface{}, error) {
	result := make([]interface{}, 0)

	var ints []models.Interface
	err := db.DB.Model(&ints).Where(`object_id = ?`, objectID).
		Where(`type = ?`, intType).
		Order(`name`).
		Select()
	if err != nil {
		return result, err
	}
	for i := range ints {
		item := make(map[string]interface{})
		item["id"] = ints[i].ID
		item["name"] = ints[i].Name
		item["shortname"] = ints[i].Shortname
		item["description"] = ints[i].Description

		result = append(result, item)
	}

	return result, nil
}*/

func (c *ObjectController) GET(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	var obj models.Object
	err = db.DB.Model(&obj).Where(`id = ?`, id).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			notFound(ctx.w)
			return
		}
		returnError(ctx.w, fmt.Sprintf("Cannot select object: %s", err.Error()), true)
		return
	}

	switch ctx.Params["what"] {
	case "interfaces":
		c.GetInterfaces(ctx, obj)
		return
	}

	intCount, err := obj.GetInterfacesCount("")
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["object"] = obj
	result["interfaces"] = intCount
	writeJSON(ctx.w, result)
}
