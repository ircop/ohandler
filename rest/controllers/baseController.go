package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/cfg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"math"
)

// HTTPContext definition
// context is used mostly for parameter passing over requests
type HTTPContext struct {
	Ctx          	*context.Context
	R            	http.Request
	W            	http.ResponseWriter
	Params       	map[string]string
	UnauthRoutes 	[]string

	//DashTemplates	string
	Config *cfg.Cfg
}

// NewContext returns new HTTPContext instance
func NewContext(ctx context.Context, r http.Request, w http.ResponseWriter, config *cfg.Cfg) *HTTPContext {
	c := new(HTTPContext)
	c.Ctx = &ctx
	c.R = r
	c.W = w
	//c.DashTemplates = dashTemplates
	c.Config = config

	c.Params = make(map[string]string)
	c.UnauthRoutes = make([]string,0)

	return c
}

type Controller interface {
	GET(ctx *HTTPContext)
	POST(ctx *HTTPContext)
	PUT(ctx *HTTPContext)
	PATCH(ctx *HTTPContext)
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
	if origin := ctx.R.Header.Get("Origin"); origin != "" {
		ctx.W.Header().Set("Access-Control-Allow-Origin", origin)
		ctx.W.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		ctx.W.Header().Set("Access-Control-Allow-Origin", "*")
	}

	ctx.W.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PATCH, PUT, DELETE")
	ctx.W.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	ctx.W.Header().Set("Content-Type", "application/json")

	// parse json and query parameters
	JSONParams := make(map[string]interface{})
	body, err := ioutil.ReadAll(ctx.R.Body)
	if nil != err {
		return err
	}
	defer ctx.R.Body.Close() // nolint

//    logger.Debug("json body: '%s'", body)
	// parse json body if exist
	err = json.Unmarshal(body, &JSONParams)
	//logger.Debug("-- jsonparams -- '%+v'", JSONParams)
	if nil == err {
		for k, v := range JSONParams {
			ctx.Params[k] = fmt.Sprintf("%v", v)
			switch v.(type) {
			    case int64:
				    ctx.Params[k] = fmt.Sprintf("%d", v)
			    case float64:
//				    logger.Debug("F64")
				    if v.(float64) == math.Trunc(v.(float64)) {
					    // this is int as float
					    ctx.Params[k] = fmt.Sprintf("%d", int64(v.(float64)))
//					    logger.Debug("WRITE '%d'", int64(v.(float64)))
				    } else {
					    ctx.Params[k] = fmt.Sprintf("%f", v.(float64))
				    }
			}
		}
	}

	GetParams := ctx.R.URL.Query()
	for param, val := range GetParams {
		param = strings.ToLower(param)
		if _, ok := ctx.Params[param]; !ok {
			ctx.Params[param] = strings.Join(val, "")
		}
	}

	// todo; auth stuff. Here or in middleware?
	if !c.checkAuth(ctx) {
		// not authorized
		ctx.W.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(ctx.W, "Not authorized", http.StatusUnauthorized)
		return fmt.Errorf("Not authorized")
	}
	return nil
}

func (c *HTTPController) checkAuth(ctx *HTTPContext) bool {
	uri := ctx.R.RequestURI

	if ctx.R.Method == "OPTIONS" {
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
		ck, err := ctx.R.Cookie("ohandler")
		if err == nil {
			tokenString = ck.Value
		}
		if tokenString == "" {
			return false
		}
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

	if t.Api {
		return true
	}

	if t.UserID != 0 {
		// set or update cookie
		expire := time.Now().AddDate(0,0,1);
		ck := http.Cookie{
			Name:"ohandler",
			Value:t.Key,
			Expires:expire,
			HttpOnly:false,
		}
		http.SetCookie(ctx.W,&ck)

		return true
	}

	return false
}


// CheckParams return true of false after checking of all passed param names in params map
func (c *HTTPController) CheckParams(ctx *HTTPContext, names []string) []string {
	ret := make([]string, 0)
	for i := range names {
		p, ok := ctx.Params[names[i]]
		if !ok || strings.Trim(p, " ") == "" {
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
	returnOk(ctx.W)
}

// DELETE handler
func (c *HTTPController) DELETE(ctx *HTTPContext) {
	http.Error(ctx.W, "Method not allowed", http.StatusMethodNotAllowed)
}

// POST handler
func (c *HTTPController) POST(ctx *HTTPContext) {
	http.Error(ctx.W, "Method not allowed", http.StatusMethodNotAllowed)
}

// POST handler
func (c *HTTPController) PUT(ctx *HTTPContext) {
	http.Error(ctx.W, "Method not allowed", http.StatusMethodNotAllowed)
}

// PATCH handler
func (c *HTTPController) PATCH(ctx *HTTPContext) {
	http.Error(ctx.W, "Method not allowed", http.StatusMethodNotAllowed)
}

// GET handler
func (c *HTTPController) GET(ctx *HTTPContext) {
	http.Error(ctx.W, "Method not allowed", http.StatusMethodNotAllowed)
}
