package db

import (
	"github.com/go-pg/pg"
	"fmt"
)

var DB *pg.DB

func InitDB(host string, port int, dbname string, user string, password string) error {
	DB = pg.Connect(&pg.Options{
		Addr:		fmt.Sprintf("%s:%d", host, port),
		User:		user,
		Password:   password,
		Database:	dbname,
	})

	var n int
	_, err := DB.QueryOne(pg.Scan(&n), "SELECT 1")
	if nil != err {
		return err
	}

	return nil
}
