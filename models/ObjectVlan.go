package models

import "github.com/golang/protobuf/proto"

//import "github.com/gogo/protobuf/proto"

type VlanType int32

const (
	VlanType_TRUNK		VlanType = 10
	VlanType_ACCESS		VlanType = 20
	VlanType_NATIVE		VlanType = 30
	VlanType_FORBIDDEN	VlanType = 40
)

type ObjectVlan struct {
	TableName struct{} `sql:"object_vlans"`

	ID				int64		`json:"id"`
	ObjectID		int64		`json:"object_id" sql:"object_id"`
	InterfaceID		int64		`json:"interface_id" sql:"interface_id"`
	VlanID			int64		`json:"vlan_id" sql:"vlan_id"`
	Mode			string		`json:"type" sql:"mode"`
	VID				int64		`json:"vid" sql:"vid"`
}


var VlanType_name = map[int32]string {
	10:		"TRUNK",
	20:		"ACCESS",
	30:		"NATIVE",
	40:		"FORBIDDEN",
}
var VlanType_value = map[string]int32 {
	"TRUNK":		10,
	"ACCESS":		20,
	"NATIVE":		30,
	"FORBIDDEN":	40,
}
func (x VlanType) String() string {
	return proto.EnumName(VlanType_name, int32(x))
}