package models

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"ircop/lightnms/models"
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

func UserByToken(token string) (*User, error) {
	var t models.Token
	if err := db.DB.Model(&t).Where(`key = ?`, token).First(); err != nil {
		return nil, err
	}

	var u User
	if err := db.DB.Model(&u).Where(`id = ?`, t.UserID).First(); err != nil {
		return nil, err
	}

	return &u, nil
}