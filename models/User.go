package models

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
)

type User struct {
	TableName struct{} `sql:"users"`

	ID			int64		`json:"id"`
	Login		string		`json:"login"`
	Password	string		`json:"password"`
}

// UserByLogin returns user, founded by login, or error
func UserByLogin(login string) (*User, error) {
	var user User
	err := db.DB.Model(&user).Where("lower(login) = lower(?)", login).First()
	if err != nil && err != pg.ErrNoRows {
		return nil, err
	}

	if err == pg.ErrNoRows {
		return nil, nil
	}

	return &user, nil
}
