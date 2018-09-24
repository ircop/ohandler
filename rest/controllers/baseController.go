package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// HTTPContext definition
// context is used mostly for parameter passing over requests
type HTTPContext struct {
	Ctx		*context.Context
	r		http.Request
	w		http.ResponseWriter
	Params	map[string]string
	UnauthRoutes	[]string
}

// NewContext returns new HTTPContext instance
func NewContext(ctx context.Context, r http.Request, w http.ResponseWriter) *HTTPContext {
	c := new(HTTPContext)
	c.Ctx = &ctx
	c.r = r
	c.w = w

	c.Params = make(map[string]string)
	c.UnauthRoutes = make([]string,0)

	return c
}

type Controller interface {
	GET(ctx *HTTPContext)
	POST(ctx *HTTPContext)
	PUT(ctx *HTTPContext)
	DELETE(ctx *HTTPContext)
	OPTIONS(ctx *HTTPContext)
	Init(ctx *HTTPContext) error
}

// HTTPController is Controller instance for furtore inheritance
type HTTPController struct {
	Controller
}

// Init controller instance
func (c *HTTPController) Init(ctx *HTTPContext) error {
	ctx.w.Header().Set("Access-Control-Allow-Origin", "*")
	ctx.w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PATCH, PUT, DELETE")
	ctx.w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	ctx.w.Header().Set("Content-Type", "application/json")

	// parse json and query parameters
	JSONParams := make(map[string]interface{})
	body, err := ioutil.ReadAll(ctx.r.Body)
	if nil != err {
		return err
	}
	defer ctx.r.Body.Close() // nolint

	// parse json body if exist
	err = json.Unmarshal(body, &JSONParams)
	if nil == err {
		for k, v := range JSONParams {
			ctx.Params[k] = fmt.Sprintf("%v", v)
		}
	}

	GetParams := ctx.r.URL.Query()
	for param, val := range GetParams {
		param = strings.ToLower(param)
		if _, ok := ctx.Params[param]; !ok {
			ctx.Params[param] = strings.Join(val, "")
		}
	}

	// todo; auth stuff. Here or in middleware?
	if !c.checkAuth(ctx) {
		// not authorized
		ctx.w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(ctx.w, "Not authorized", http.StatusUnauthorized)
		return fmt.Errorf("Not authorized")
	}
	return nil
}

func (c *HTTPController) checkAuth(ctx *HTTPContext) bool {
	uri := ctx.r.RequestURI

	if ctx.r.Method == "OPTIONS" {
		return true
	}

	// if this is one of unauth-urls, auth is not needed
	for _, url := range ctx.UnauthRoutes {
		if url == uri {
			return true
		}
	}

	// We need a token
	tokenString, ok := ctx.Params["token"]
	if !ok {
		return false
	}

	// todo: CACHE USER TOKENS to 1..5 minutes
	// todo: roles, rights, etc...
	var t models.RestToken
	err := db.DB.Model(&t).Where(`key = ?`, tokenString).First()
	if err != nil && err == pg.ErrNoRows {
		return false
	}
	if err != nil && err != pg.ErrNoRows {
		logger.RestErr("Cannot select token: %s", err.Error())
		return false
	}

	if t.UserID != 0 {
		return true
	}

	return true
}


// CheckParams return true of false after checking of all passed param names in params map
func (c *HTTPController) CheckParams(ctx *HTTPContext, names []string) []string {
	ret := make([]string, 0)
	for i := range names {
		p, ok := ctx.Params[names[i]]
		if !ok || p == "" {
			ret = append(ret, names[i])
		}
	}
	return ret
}

// IntParam returns in64-converted parameter value or error
func (c *HTTPController) IntParam(ctx *HTTPContext, name string) (int64, error) {
	param, ok := ctx.Params[name]
	if !ok {
		return 0, fmt.Errorf("Parameter '%s' doesn't exist", name)
	}

	i, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Parameter '%s' is not integer (%s)", name, param)
	}

	return i, nil
}

// OPTIONS handler
func (c *HTTPController) OPTIONS(ctx *HTTPContext) {
	returnOk(ctx.w)
}

// DELETE handler
func (c *HTTPController) DELETE(ctx *HTTPContext) {
	http.Error(ctx.w, "Method not allowed", http.StatusMethodNotAllowed)
}

// POST handler
func (c *HTTPController) POST(ctx *HTTPContext) {
	http.Error(ctx.w, "Method not allowed", http.StatusMethodNotAllowed)
}

// POST handler
func (c *HTTPController) PUT(ctx *HTTPContext) {
	http.Error(ctx.w, "Method not allowed", http.StatusMethodNotAllowed)
}

// GET handler
func (c *HTTPController) GET(ctx *HTTPContext) {
	http.Error(ctx.w, "Method not allowed", http.StatusMethodNotAllowed)
}
