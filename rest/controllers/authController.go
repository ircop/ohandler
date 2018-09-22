package controllers

import (
	"fmt"
	"github.com/ircop/ohandler/models"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

// AuthController struct
type AuthController struct {
	HTTPController
}

// POST - try to authenticate user
func (c *AuthController) POST(ctx *HTTPContext) {
	missing := c.CheckParams(ctx, []string{"login", "password"})
	if len(missing) > 0 {
		returnError(ctx.w, fmt.Sprintf("Missing parameters: %s", strings.Join(missing, ",")), false)
		return
	}

	login := ctx.Params["login"]
	password := ctx.Params["password"]
	//salt := "wptj5yh8-&(^R%R#@j2pa"
	//salted := fmt.Sprintf("%s%s", password, salt)

	// search for user
	//tokenString := ""
	user, err := models.UserByLogin(login)
	if err != nil {
		internalError(ctx.w, err.Error())
		return
	}
	if user == nil {
		returnError(ctx.w,"Wrong login or password", false)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		returnError(ctx.w, "Wrong login or password", false)
		return
	}

	// create or find token
	t, err := models.TokenFindOrCreate(user.ID)
	if err != nil {
		returnError(ctx.w, err.Error(), false)
		return
	}

	fmt.Fprintf(ctx.w, `{"token":"%s"}`, t.Key)
}

