package controllers

import (
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
			writeJSON(ctx.w, result)
			return
		}
		returnError(ctx.w, err.Error(), true)
		return
	}

	result["segments"] = segs

	writeJSON(ctx.w, result)
}

func (c *SegmentsController) getSegment(id int64, ctx *HTTPContext) {
	var seg models.Segment
	if err := db.DB.Model(&seg).Where(`id = ?`, id).First(); err != nil {
		if err == pg.ErrNoRows {
			notFound(ctx.w)
			return
		}
		returnError(ctx.w, err.Error(), true)
		return
	}

	result := make(map[string]interface{})
	result["segment"] = seg
	writeJSON(ctx.w, result)
}

// add new segment
func (c *SegmentsController) POST(ctx *HTTPContext) {
	title := strings.Trim(ctx.Params["title"], " ")
	if title == "" {
		returnError(ctx.w, "Wrong segment name", true)
		return
	}

	cnt, err := db.DB.Model(&models.Segment{}).Where(`title = ?`, title).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	if cnt > 0 {
		returnError(ctx.w, "There is already segment with this name", true)
		return
	}

	seg := models.Segment{Title:title}
	if err = db.DB.Insert(&seg); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	returnOk(ctx.w)
}

// rename old segment
func (c *SegmentsController) PATCH(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong ID", true)
		return
	}
	title := strings.Trim(ctx.Params["title"], " ")
	if title == "" {
		returnError(ctx.w, "Wrong segment name", true)
		return
	}

	// check is title free
	cnt, err := db.DB.Model(&models.Segment{}).Where(`title = ?`, title).Where(`id <> ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, "There is already segment with this name", true)
		return
	}

	var seg models.Segment
	if err = db.DB.Model(&seg).Where(`id = ?`, id).First(); err != nil {
		if err == pg.ErrNoRows {
			notFound( ctx.w)
			return
		}
		returnError(ctx.w, err.Error(), true)
		return
	}

	seg.Title = title
	if err = db.DB.Update(&seg); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}

	returnOk(ctx.w)
}

func (c *SegmentsController) DELETE(ctx *HTTPContext) {
	id, err := c.IntParam(ctx, "id")
	if err != nil {
		returnError(ctx.w, "Wrong ID", true)
		return
	}

	cnt, err := db.DB.Model(&models.ObjectSegment{}).Where(`segment_id = ?`, id).Count()
	if err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	if cnt > 0 {
		returnError(ctx.w, "Cannot delete segment: there are objects bounded.", true)
		return
	}

	if _, err = db.DB.Model(&models.Segment{}).Where(`id = ?`, id).Delete(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	returnOk(ctx.w)
}
