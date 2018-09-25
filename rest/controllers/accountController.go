package controllers

import (
	"fmt"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"strings"
)

type AccountController struct {
	HTTPController
}

func (c *AccountController) GET(ctx *HTTPContext) {
	// get user data
	token := ctx.Params["token"]
	if token == "" {
		returnError(ctx.w, "No token", false)
		return
	}

	user, err := models.UserByToken(token)
	if err != nil {
		returnError(ctx.w, err.Error(), false)
		return
	}
	if user == nil {
		returnError(ctx.w, "User not found", false)
		return
	}

	result := make(map[string]interface{})
	result["user_id"] = user.ID
	result["login"] = user.Login

	writeJSON(ctx.w, result)
}

func (c *AccountController) PUT(ctx *HTTPContext) {
	// trying to change password
	required := []string{"token", "old", "new1", "new2"}
	if missing := c.CheckParams(ctx, required); len(missing) > 0 {
		returnError(ctx.w, fmt.Sprintf("Missing required parameters: %s", strings.Join(missing, ", ")), true)
		return
	}

	user, err := models.UserByToken(ctx.Params["token"])
	if err != nil || user == nil {
		returnError(ctx.w, "User not found", false)
		return
	}

	old := ctx.Params["old"]
	new1 := ctx.Params["new1"]
	new2 := ctx.Params["new2"]

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(old)); err != nil {
		returnError(ctx.w, "Old password is wrong", true)
		return
	}

	if new1 != new2 {
		returnError(ctx.w, "Password and confirmation are not equal", true)
		return
	}

	if len(new1) < 6 {
		returnError(ctx.w, "Passwourd should contain at least 6 chars", true)
		return
	}

	re1, err1 := regexp.Compile(`[a-zA-Z]`)
	re2, err2 := regexp.Compile(`[0-9]`)
	if err1 != nil || err2 != nil {
		returnError(ctx.w, fmt.Sprintf("Cannot compile regex: %v %v", err1, err2), true)
		return
	}

	if !re1.Match([]byte(new1)) || !re2.Match([]byte(new2)) {
		returnError(ctx.w, fmt.Sprintf("Password should contain both letters and numbers"), true)
		return
	}

	// change password
	newHash, err := bcrypt.GenerateFromPassword([]byte(new1), bcrypt.DefaultCost)
	if err != nil {
		returnError(ctx.w, fmt.Sprintf("Cannot hash new password: %s", err.Error()), true)
		return
	}

	user.Password = string(newHash)
	if err = db.DB.Update(user); err != nil {
		returnError(ctx.w, err.Error(),true)
		return
	}

	returnOk(ctx.w)
}
