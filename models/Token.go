package models

import (
	"crypto/rand"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
)

type RestToken struct {
	TableName struct{} `sql:"tokens"`

	ID		int64	`json:"id"`
	Key		string	`json:"key"`
	UserID	int64	`json:"user_id"`
	Api		bool	`json:"api"`
}

func TokenFindOrCreate(uid int64) (*RestToken, error) {
	var token RestToken

	err := db.DB.Model(&token).Where("user_id = ?", uid).Last()
	if err != nil && err != pg.ErrNoRows {
		return nil, err
	}
	if err == nil {
		return &token, nil
	}

	// errnorows here
	b := make([]byte, 40)
	_, err = rand.Read(b)
	if nil != err {
		return nil, err
	}
	key := fmt.Sprintf("%x", b)

	t := RestToken{
		UserID:uid,
		Key: key,
		Api:false,
	}

	// write token to DB
	err = db.DB.Insert(&t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}
