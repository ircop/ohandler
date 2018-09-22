package controllers

import (
	"github.com/ircop/ohandler/models"
	"github.com/ircop/ohandler/db"
	"github.com/go-pg/pg"
	"fmt"
	"strings"
)

type AuthProfileController struct {
	HTTPController
}

func (c *AuthProfileController) GET(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err == nil {
		c.GetProfile(id, ctx)
		return
	}

	// get all profiles
	result := make(map[string]interface{})
	var aps []models.AuthProfile
	if err = db.DB.Model(&aps).Order(`title`).Select(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	result["profiles"] = aps
	writeJSON(ctx.w, result)
}

func (c *AuthProfileController) GetProfile(id int64, ctx *HTTPContext) {
	var ap models.AuthProfile
	if err := db.DB.Model(&ap).Where(`id = ?`, id).Select(); err != nil {
		if err == pg.ErrNoRows {
			notFound(ctx.w)
			return
		}
		returnError(ctx.w, err.Error(), true)
		return
	}

	writeJSON(ctx.w, ap)
}

func (c *AuthProfileController) POST(ctx *HTTPContext) {
	switch ctx.Params["what"] {
	case "save":
		c.Save(ctx)
		return
	case "add":
		c.Add(ctx)
		return
	case "delete":
		c.Delete(ctx)
		return 
	default:
		returnError(ctx.w, "Unknown action", true)
		return
	}
}

func (c *AuthProfileController) Delete(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong profile ID", true)
		return
	}

	cnt, err := db.DB.Model(&models.Object{}).Where(`auth_id = ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, fmt.Sprintf("Cannot delete: there is %d objects with this auth profile set.", cnt), true)
		return
	}

	if _, err = db.DB.Model(&models.AuthProfile{}).Where(`id = ?`, id).Delete(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	returnOk(ctx.w)
}

func (c *AuthProfileController) Add(ctx *HTTPContext) {
	required := []string{"title", "cli_type", "login", "password"}
	if errors := c.CheckParams(ctx, required); len(errors) > 0 {
		returnError(ctx.w, fmt.Sprintf("Missing required parameters: %s", strings.Join(errors, ", ")), true)
		return
	}

	// check for same title
	cnt, err := db.DB.Model(&models.AuthProfile{}).Where(`title = ?`, ctx.Params["title"]).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, fmt.Sprintf("There is already auth profile named '%s'", ctx.Params["name"]), true)
		return
	}

	// create and save profile
	ap := models.AuthProfile{
		Title:ctx.Params["title"],
		Login:ctx.Params["login"],
		Password:ctx.Params["password"],
		Enable:ctx.Params["enable"],
		RoCommunity:ctx.Params["ro_community"],
		RwCommunity:ctx.Params["rw_community"],
	}
	switch ctx.Params["cli_type"] {
	case "ssh":
		ap.CliType = models.CliTypeSSH
		break
	case "telnet":
		ap.CliType = models.CliTypeTelnet
		break
	default:
		ap.CliType = models.CliTypeNone
		break
	}

	if err = db.DB.Insert(&ap); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	returnOk(ctx.w)
}

func (c *AuthProfileController) Save(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong profile ID", true)
		return
	}

	required := []string{"id", "title", "cli_type", "login", "password"}
	if errors := c.CheckParams(ctx, required); len(errors) > 0 {
		returnError(ctx.w, fmt.Sprintf("Missing required parameters: %s", strings.Join(errors, ", ")), true)
		return
	}

	// check for same title
	cnt, err := db.DB.Model(&models.AuthProfile{}).Where(`title = ?`, ctx.Params["title"]).Where(`id != ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, fmt.Sprintf("There is already auth profile named '%s'", ctx.Params["name"]), true)
		return
	}

	var ap models.AuthProfile
	if err := db.DB.Model(&ap).Where(`id = ?`, id).Select(); err != nil {
		if err == pg.ErrNoRows {
			notFound(ctx.w)
			return
		}
		returnError(ctx.w, err.Error(), true)
		return
	}

	switch ctx.Params["cli_type"] {
	case "ssh":
		ap.CliType = models.CliTypeSSH
		break
	case "telnet":
		ap.CliType = models.CliTypeTelnet
		break
	default:
		ap.CliType = models.CliTypeNone
		break
	}
	ap.Title = ctx.Params["title"]
	ap.Login = ctx.Params["login"]
	ap.Password = ctx.Params["password"]
	ap.Enable = ctx.Params["enable"]
	ap.RoCommunity = ctx.Params["ro_community"]
	ap.RwCommunity = ctx.Params["rw_community"]
	if err = db.DB.Update(&ap); err != nil {
		returnError(ctx.w, err.Error(), true)
	} else {
		returnOk(ctx.w)
	}
}