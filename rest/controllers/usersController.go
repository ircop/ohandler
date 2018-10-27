package controllers

import (
	"fmt"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"strings"
)

type UsersController struct {
	HTTPController
}

func (c *UsersController) GET(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err == nil {
		c.getUser(id, ctx)
		return
	}
	var users []models.User
	if err := db.DB.Model(&users).OrderExpr(`natsort(login)`).Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["users"] = users

	WriteJSON(ctx.W, result)
}

func (c *UsersController) getUser(id int64, ctx *HTTPContext) {
	var user models.User
	if err := db.DB.Model(&user).Where(`id = ?`, id).First(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["login"] = user.Login
	result["id"] = user.ID

	WriteJSON(ctx.W, result)
}

// Add user
func (c *UsersController) POST(ctx *HTTPContext) {
	required := []string{"login", "password"}
	if missing := c.CheckParams(ctx, required); len(missing) > 0 {
		ReturnError(ctx.W, fmt.Sprintf("Missing parameters: %s", strings.Join(missing, ", ")), true)
		return
	}

	reLogin, err := regexp.Compile(`^[a-zA-Z0-9]+$`)
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	login := strings.Trim(ctx.Params["login"], " ")
	if len(login) < 2 {
		ReturnError(ctx.W, "Login len should be 2+ chars", true)
		return
	}
	if !reLogin.Match([]byte(login)) {
		ReturnError(ctx.W, "Login should be alphanumeric string", true)
		return
	}

	cnt, err := db.DB.Model(&models.User{}).Where(`login = ?`, login).Count()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	if cnt > 0 {
		ReturnError(ctx.W, "This login is already taken", true)
		return
	}

	pw, err := bcrypt.GenerateFromPassword([]byte(strings.Trim(ctx.Params["password"], " ")), bcrypt.DefaultCost)
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	u := models.User{
		Login:ctx.Params["login"],
		Password:string(pw),
	}
	if err = db.DB.Insert(&u); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	returnOk(ctx.W)
}

// todo: drop user tokens/sessions

// change password and/or login
func (c *UsersController) PATCH(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, "Wrong user ID", true)
		return
	}

	reLogin, err := regexp.Compile(`^[a-zA-Z0-9]+$`)
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	var user models.User
	if err := db.DB.Model(&user).Where(`id = ?`, id).First(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	changed := false
	newPassword := strings.Trim(ctx.Params["password"], " ")
	if newPassword != "" {
		if err = c.checkPW(newPassword); err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}

		// change password
		newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			ReturnError(ctx.W, fmt.Sprintf("Cannot hash new password: %s", err.Error()), true)
			return
		}

		user.Password = string(newHash)
		changed = true
	}
	//---------
	newLogin := strings.Trim(ctx.Params["login"], " ")
	if newLogin != "" && newLogin != user.Login {
		if len(newLogin) < 2 {
			ReturnError(ctx.W, "Login length should be 2+ chars", true)
			return
		}
		if !reLogin.Match([]byte(newLogin)) {
			ReturnError(ctx.W, "Login should be alphanumeric string", true)
			return
		}

		user.Login = newLogin
		changed = true
	}

	if changed {
		if err = db.DB.Update(&user); err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
	}

	returnOk(ctx.W)
}

func (c *UsersController) checkPW(pw string) error {
	if len(pw) < 6 {
		return fmt.Errorf("Passwourd should contain at least 6 chars")
	}

	re1, err1 := regexp.Compile(`[a-zA-Z]`)
	re2, err2 := regexp.Compile(`[0-9]`)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("Cannot compile regex: %v %v", err1, err2)
	}

	if !re1.Match([]byte(pw)) || !re2.Match([]byte(pw)) {
		return fmt.Errorf("Password should contain both letters and numbers")
	}

	return nil
}
