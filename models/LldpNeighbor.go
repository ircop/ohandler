package models

import "time"

type LldpNeighbor struct {
	ID					int64		`json:"id"`
	ObjectID			int64		`json:"object_id"`
	LocalInterfaceID	int64		`json:"local_interface_id"`
	NeighborID			int64		`json:"neighbor_id"`
	NeighborInterfaceID	int64		`json:"neighbor_interface_id"`
	CreatedAt			*time.Time	`json:"created_at"`
}