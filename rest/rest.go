package rest

import (
	"fmt"
	"github.com/ircop/ohandler/cfg"
	"github.com/ircop/ohandler/logger"
	"log"
	"net"
	"net/http"
)

type Rest struct {
	listenIP	string
	listenPort	int
	ssl			bool
	cert		string
	key			string
}

func New(cfg *cfg.Cfg) *Rest {
	r := Rest{
		ssl:cfg.Ssl,
		cert:cfg.SslCert,
		key:cfg.SslKey,
		listenIP:cfg.RestIP,
		listenPort:cfg.RestPort,
	}

	return &r
}

func (r *Rest) Listen() {
	listener, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", r.listenIP, r.listenPort))
	if err != nil {
		logger.Err(err.Error())
		log.Fatal(err.Error())
	}

	router := getRouter()
	if r.ssl {
		log.Fatal(http.ServeTLS(listener, router, r.cert, r.key))
	} else {
		log.Fatal(http.Serve(listener, router))
	}
}