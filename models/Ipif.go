package models

import (
	//"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/proto"
	"time"
)

type IpifType int32

const IpifType_DISCOVERED	IpifType = 10;
const IpifType_MANUAL		IpifType = 20;

type Ipif struct {
	TableName struct{} `sql:"ips"`

	ID 			int64		`json:"id"`
	ObjectID	int64		`json:"object_id" sql:"object_id"`
	InterfaceID	int64		`json:"interface_id" sql:"interface_id"`
	NetworkID	int64		`json:"network_id" sql:"network_id"`
	Addr		string		`json:"addr" sql:"addr"`
	Description	string		`json:"description"`
	Type		string		`json:"type" sql:"type"`
	CreatedAt	*time.Time	`json:"created_at" sql:"created_at"`
}

var IpifType_name = map[int32]string {
	10: "DISCOVERED",
	20: "MANUAL",
}
var IpifType_value = map[string]int32 {
	"DISCOVERED": 10,
	"MANUAL": 20,
}

func (x IpifType) String() string {
	return proto.EnumName(IpifType_name, int32(x))
}
