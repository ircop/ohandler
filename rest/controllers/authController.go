package controllers

import (
	"fmt"
	"github.com/ircop/ohandler/models"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"
)

// AuthController struct
type AuthController struct {
	HTTPController
}

// POST - try to authenticate user
func (c *AuthController) POST(ctx *HTTPContext) {
	missing := c.CheckParams(ctx, []string{"login", "password"})
	if len(missing) > 0 {
		ReturnError(ctx.W, fmt.Sprintf("Missing parameters: %s", strings.Join(missing, ",")), false)
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
		InternalError(ctx.W, err.Error())
		return
	}
	if user == nil {
		ReturnError(ctx.W,"Wrong login or password", false)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		ReturnError(ctx.W, "Wrong login or password", false)
		return
	}

	// create or find token
	t, err := models.TokenFindOrCreate(user.ID)
	if err != nil {
		ReturnError(ctx.W, err.Error(), false)
		return
	}

	// set cookie
	expire := time.Now().AddDate(0,0,1);
	ck := http.Cookie{
		Name:"ohandler",
		Value:t.Key,
		Expires:expire,
		HttpOnly:false,
	}
	http.SetCookie(ctx.W,&ck)

	fmt.Fprintf(ctx.W, `{"token":"%s"}`, t.Key)
}

