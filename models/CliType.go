package models

import (
	"database/sql/driver"
	"github.com/pkg/errors"
)

type CliType uint16

const (
	CliTypeNone		= CliType(0)
	CliTypeTelnet	= CliType(1)
	CliTypeSSH		= CliType(2)
)

func (ct CliType) String() string {
	switch ct {
	case CliTypeSSH:
		return "ssh"
	case CliTypeTelnet:
		return "telnet"
	case CliTypeNone:
		return "none"
	default:
		return "none"
	}
}

func (ct CliType) MarshalText() ([]byte, error) {
	return []byte(ct.String()), nil
}

func (ct *CliType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "ssh":
		*ct = CliTypeSSH
	case "telnet":
		*ct = CliTypeTelnet
	case "none":
		*ct = CliTypeNone
	default:
		return errors.New("Invalid cli_type")
	}

	return nil
}

func (ct CliType) Value() (driver.Value, error) {
	return ct.String(), nil
}

func (ct *CliType) Scan(src interface{}) error {
	buf, ok := src.([]byte)
	if !ok {
		return errors.New("Invalid cli_type")
	}

	return ct.UnmarshalText(buf)
}
