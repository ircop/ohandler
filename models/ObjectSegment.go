package models

type ObjectSegment struct {
	TableName struct{} `sql:"object_segments" json:"-"`
	ID			int64		`json:"id" sql:"id"`
	ObjectID	int64		`json:"object_id" sql:"object_id"`
	SegmentID	int64		`json:"segment_id" sql:"segment_id"`
}