package rest

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/rest/controllers"
	"github.com/ircop/ohandler/rest/controllers/dash"
	"net/http"
)

func (r *Rest) getRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(false)

	router.HandleFunc("/login", r.obs(&controllers.AuthController{}))
	router.HandleFunc("/objects", r.obs(&controllers.ObjectsController{}))
	router.HandleFunc("/raw-objects", r.obs(&controllers.RawObjectsController{}))
	router.HandleFunc("/object", r.obs(&controllers.ObjectController{}))
	router.HandleFunc("/update-object", r.obs(&controllers.ObjectUpdateController{}))
	router.HandleFunc("/os-profiles", r.obs(&controllers.OsProfilesController{}))
	router.HandleFunc("/auth-profiles", r.obs(&controllers.AuthProfileController{}))
	router.HandleFunc("/discovery-profiles", r.obs(&controllers.DiscoveryProfileController{}))
	router.HandleFunc("/account", r.obs(&controllers.AccountController{}))
	router.HandleFunc("/users", r.obs(&controllers.UsersController{}))
	router.HandleFunc("/vlans", r.obs(&controllers.VlansController{}))
	router.HandleFunc("/networks", r.obs(&controllers.NetworksController{}))
	router.HandleFunc("/map", r.obs(&controllers.MapController{}))
	router.HandleFunc("/search", r.obs(&controllers.SearchController{}))
	router.HandleFunc("/links", r.obs(&controllers.LinksController{}))
	router.HandleFunc("/segments", r.obs(&controllers.SegmentsController{}))
	router.HandleFunc("/configs", r.obs(&controllers.ConfigsController{}))
	router.HandleFunc("/keys", r.obs(&controllers.ApiKeysController{}))
	router.HandleFunc("/models", r.obs(&controllers.ModelsController{}))

	router.HandleFunc("/dash/port", r.obs(&dash.PortController{}))
	router.HandleFunc("/dash/object", r.obs(&dash.ObjectController{}))

	router.Use(middleware)
	return router
}

func (r *Rest) obs(handler controllers.Controller) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		httpContext := controllers.NewContext(ctx, *req, w, r.dashTemplates)

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
		case "PUT":
			handler.PUT(httpContext)
		case "PATCH":
			handler.PATCH(httpContext)
		case "DELETE":
			handler.DELETE(httpContext)
		case "OPTIONS":
			handler.OPTIONS(httpContext)
		default:
			logger.RestErr("[web]: unsupported method (%s) call from %s: ", req.Method, req.RemoteAddr, req.URL)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}