package rest

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/rest/controllers"
	"net/http"
)

func getRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(false)

	router.HandleFunc("/login", obs(&controllers.AuthController{}))
	router.HandleFunc("/objects", obs(&controllers.ObjectsController{}))
	router.HandleFunc("/object", obs(&controllers.ObjectController{}))

	router.Use(middleware)
	return router
}

func obs(handler controllers.Controller) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		httpContext := controllers.NewContext(ctx, *req, w)

		httpContext.UnauthRoutes = append(httpContext.UnauthRoutes, "/login")
		httpContext.UnauthRoutes = append(httpContext.UnauthRoutes, "/ping")

		err := handler.Init(httpContext)
		if err != nil {
			logger.RestErr("[web]: error initializing controller: %s", err.Error())
			http.Error(w, "Error initializing controller", http.StatusInternalServerError)
			return
		}

		switch(req.Method) {
		case "GET":
			handler.GET(httpContext)
		case "POST":
			handler.POST(httpContext)
		default:
			logger.RestErr("[web]: unsupported method (%s) call from %s: ", req.Method, req.RemoteAddr, req.URL)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}