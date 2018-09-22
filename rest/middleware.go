package rest

import (
	"github.com/ircop/ohandler/logger"
	"net/http"
)

// middleware is used mostly for auth purposes
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			if r := recover(); r != nil {
				logger.Panic("Recovered in http middleware: %+v", r)
				logger.RestErr("Recovered in http middleware: %+v", r)


				http.Error(w, "Internal error (panic in middleware)", 500)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
