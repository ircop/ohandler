package controllers

import (
	"database/sql"
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/models"
	"strings"
)

type SegmentsController struct {
	HTTPController
}

func (c *SegmentsController) GET(ctx *HTTPContext) {
	result := make(map[string]interface{})
	id, err := c.IntParam(ctx, "id")
	if err == nil {
		c.getSegment(id, ctx)
		return
	}

	// get all
	segs := make([]models.Segment, 0)
	if err = db.DB.Model(&segs).Order(`title`).Select(); err != nil  {
		if err == pg.ErrNoRows {
			result["segments"] = segs
			WriteJSON(ctx.W, result)
			return
		}
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result["segments"] = segs

	WriteJSON(ctx.W, result)
}

func (c *SegmentsController) getSegment(id int64, ctx *HTTPContext) {
	var seg models.Segment
	if err := db.DB.Model(&seg).Where(`id = ?`, id).First(); err != nil {
		if err == pg.ErrNoRows {
			NotFound(ctx.W)
			return
		}
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["segment"] = seg
	WriteJSON(ctx.W, result)
}

// add new segment
func (c *SegmentsController) POST(ctx *HTTPContext) {
	title := strings.Trim(ctx.Params["title"], " ")
	foreign := sql.NullInt64{Int64:0,Valid:false}
	if title == "" {
		ReturnError(ctx.W, "Wrong segment name", true)
		return
	}
//	fmt.Printf("fid: '%s'\n", ctx.Params["foreign_id"])
	if fid, err := c.IntParam(ctx, "foreign_id"); err == nil && fid > 0 {
		foreign.Int64 = fid
		foreign.Valid = true
	}

	cnt, err := db.DB.Model(&models.Segment{}).Where(`title = ?`, title).Count()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if cnt > 0 {
		ReturnError(ctx.W, "There is already segment with this name", true)
		return
	}

	// check if there is already segment with same foreign ID
	if foreign.Valid && foreign.Int64 != 0 {
		cnt, err := db.DB.Model(&models.Segment{}).Where(`foreign_id = ?`, foreign.Int64).Count()
		if err != nil {
			ReturnError(ctx.W, err.Error(), true)
			return
		}
		if cnt > 0 {
			ReturnError(ctx.W, "There is already segment with this foreign_id", true)
			return
		}
	}

	seg := models.Segment{Title:title, ForeignID:foreign, Trash:false}
	if ctx.Params["trash"] == "true" {
		seg.Trash = true
	}
	if err = db.DB.Insert(&seg); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	returnOk(ctx.W)
}

// rename old segment
func (c *SegmentsController) PATCH(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, "Wrong ID", true)
		return
	}
	title := strings.Trim(ctx.Params["title"], " ")
	if title == "" {
		ReturnError(ctx.W, "Wrong segment name", true)
		return
	}

	foreign := sql.NullInt64{Int64:0,Valid:false}
	if fid, err := c.IntParam(ctx, "foreign_id"); err == nil && fid > 0 {
		foreign.Int64 = fid
		foreign.Valid = true
	}

	// check is title free
	cnt, err := db.DB.Model(&models.Segment{}).Where(`title = ?`, title).Where(`id <> ?`, id).Count()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if cnt > 0 {
		ReturnError(ctx.W, "There is already segment with this name", true)
		return
	}

	var seg models.Segment
	if err = db.DB.Model(&seg).Where(`id = ?`, id).First(); err != nil {
		if err == pg.ErrNoRows {
			NotFound( ctx.W)
			return
		}
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	seg.Title = title
	seg.ForeignID = foreign
	if ctx.Params["trash"] == "true" {
		seg.Trash = true
	} else {
		seg.Trash = false
	}
	if err = db.DB.Update(&seg); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}

	returnOk(ctx.W)
}

func (c *SegmentsController) DELETE(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		ReturnError(ctx.W, "Wrong ID", true)
		return
	}

	cnt, err := db.DB.Model(&models.ObjectSegment{}).Where(`segment_id = ?`, id).Count()
	if err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if cnt > 0 {
		ReturnError(ctx.W, "Cannot delete segment: there are objects bounded.", true)
		return
	}

	if _, err = db.DB.Model(&models.Segment{}).Where(`id = ?`, id).Delete(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	returnOk(ctx.W)
}
