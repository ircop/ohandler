package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/ircop/ohandler/logger"
	"io"
	"net/http"
)

func internalError(w http.ResponseWriter, message string) {
	logger.RestErr("[web]: Internal error: %s", message)
	http.Error(w, fmt.Sprintf("Internal error: %s", message), http.StatusInternalServerError)
}

func returnError(w io.Writer, message string, authorized bool) {
	logger.RestErr("REST error: %s", message)
	fmt.Fprintf(w, fmt.Sprintf(`{"error":true,"message":"%s", "authorized":%v}`, message, authorized))
}

func returnOk(w io.Writer) {
	fmt.Fprintf(w, `{"ok":true}`)
}

func notFound(w http.ResponseWriter) {
	http.Error(w, `{"error":true,"message":"not found"}`, http.StatusNotFound)
}

func writeJSON(w http.ResponseWriter, value interface{}) {
	bytes, e := json.Marshal(value)
	if nil != e {
		internalError(w, e.Error())
		return
	}

	fmt.Fprintf(w, "%s", string(bytes))
}

