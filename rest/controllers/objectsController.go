package controllers

import (
	"github.com/go-pg/pg"
	"github.com/ircop/discoverer/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"strconv"
)

type ObjectsController struct {
	HTTPController
}
/*
type webObj struct {
	ID			int64		`json:"id"`
	Mgmt		string		`json:"mgmt"`
	Name		string		`json:"name"`
	Alive		bool		`json:"alive"`
	Model		string		`json:"model"`
	Revision	string		`json:"revision"`
	Version		string		`json:"version"`
	Serial		string		`json:"serial"`
	Interfaces	int64		`json:"int_count"'`
	Links		int64		`json:"link_count"'`
}*/

type intCount struct {
	TableName struct{} 		`sql:"interfaces" json:"-"`
	ObjectID		int64	`sql:"object_id"`
	Count			int64	`sql:"cnt"`
}

type objvlans struct {
	TableName struct{} 		`sql:"object_vlans" json:"-"`
	ObjectID		int64	`sql:"object_id"`
	Count			int64	`sql:"cnt"`
}

type linkCount struct {
	TableName struct{} 		`sql:"links" json:"-"`
	ObjectID		int64	`sql:"oid"`
	Count			int 	`sql:"cnt"`
}

func (c *ObjectsController) GET(ctx *HTTPContext) {
	// todo: pagination, sorting
	order := "id ASC"
	var limit int64 = 10
	var page int64
	var offset int64

	if psize, ok := ctx.Params["pagesize"]; ok {
		pageSize, err := strconv.ParseInt(psize, 10, 64)
		if err == nil {
			limit = pageSize
		}
	}

	if pageStr, ok := ctx.Params["page"]; ok {
		pageNum, err := strconv.ParseInt(pageStr, 10, 64)
		if err == nil {
			page = pageNum
			if page > 1 {
				offset = (page -1) * limit
			}
		}
	}

	ofield := ctx.Params["sortfield"]
	oorder := ctx.Params["sortorder"]
	if ofield != "" && oorder != "" {
		o1 := ""
		o2 := "ASC"
		if oorder == "descend" {
			o2 = "DESC"
		}

		switch ofield {
		case "id":
			o1 = "id"
			break
		case "name":
			o1 = "natsort(name)"
			break
		case "mgmt":
			o1 = "mgmt"
			break
		case "alive":
			o1 = "alive"
			break
		case "model":
			o1 = "model"
			break
		}
		if o1 != "" {
			order = o1 + " " + o2
		}
	}

	// total count:
	cnt, err := db.DB.Model(&models.Object{}).Count()
	if err != nil {
		logger.RestErr("Cannot select objects count: %s", err.Error())
		internalError(ctx.w, err.Error())
		return
	}

	var objects []models.Object
	//err = db.DB.Model(&objects).Order(order).Limit(int(limit)).Offset(int(offset)).Select()
	_, err = db.DB.Query(&objects, `select * from objects order by ` + order + ` limit ? offset ?`, limit, offset)
	if err != nil {
		logger.RestErr("Error selecting objects: %s", err.Error())
		internalError(ctx.w, err.Error())
		return
	}

	rows := make([]map[string]interface{}, 0)
	var ids []int64
	for i, _ := range objects {
		item := make(map[string]interface{})
		item["id"] = objects[i].ID
		item["name"] = objects[i].Name
		item["mgmt"] = objects[i].Mgmt
		item["alive"] = objects[i].Alive
		item["model"] = objects[i].Model
		item["revision"] = objects[i].Revision
		item["version"] = objects[i].Version
		item["serial"] = objects[i].Serial
		item["interfaces"] = 0
		item["links"] = 0

		ids = append(ids, objects[i].ID)
		rows = append(rows, item)
	}

	// Count vlans
	var vcounts []objvlans
	if err := db.DB.Model(&vcounts).Column(`object_id`).ColumnExpr(`count(distinct(vid)) as cnt`).
		Where(`object_id in (?)`, pg.In(ids)).
		Group(`object_id`).
		Select(); err != nil {
			returnError(ctx.w, err.Error(), true)
			return
		}

	for i := range vcounts {
		for j := range rows {
			if rows[j]["id"].(int64) == vcounts[i].ObjectID {
				rows[j]["vlans"] = vcounts[i].Count
			}
		}
	}

	if err = c.GetInterfaceCounts(rows, ids); err != nil {
		logger.RestErr("Error selecting int count: %s", err.Error())
		internalError(ctx.w, err.Error())
		return
	}

	// Count links: 2 times, because of 'object1 or object2'
	var lcount1 []linkCount
	var lcount2 []linkCount
	if err := db.DB.Model(&lcount1).ColumnExpr(`object1_id as oid`).ColumnExpr(`count(*) as cnt`).
		Where(`object1_id in (?)`, pg.In(ids)).Group(`object1_id`).Select(); err != nil {
			returnError(ctx.w, err.Error(), true)
		return
	}
	if err := db.DB.Model(&lcount2).ColumnExpr(`object2_id as oid`).ColumnExpr(`count(*) as cnt`).
		Where(`object2_id in (?)`, pg.In(ids)).Group(`object2_id`).Select(); err != nil {
		returnError(ctx.w, err.Error(), true)
		return
	}
	for i := range lcount1 {
		for j := range rows {
			if rows[j]["id"].(int64) == lcount1[i].ObjectID {
				links := rows[j]["links"].(int) + lcount1[i].Count
				rows[j]["links"] = links
			}
		}
	}
	for i := range lcount2 {
		for j := range rows {
			if rows[j]["id"].(int64) == lcount2[i].ObjectID {
				links := rows[j]["links"].(int) + lcount2[i].Count
				rows[j]["links"] = links
			}
		}
	}

	// return result
	results := make(map[string]interface{})
	results["total"] = cnt
	results["rows"] = rows
	writeJSON(ctx.w, results)
	//returnOk(ctx.w)
}

func (c *ObjectsController) GetInterfaceCounts(rows []map[string]interface{}, ids []int64) error {
	/* phisycal interfaces */
	var counts []intCount
	if err := db.DB.Model(&counts).
		Column(`object_id`).
		ColumnExpr(`count(*) as cnt`).
		Where(`object_id in (?)`, pg.In(ids)).
		Where(`type = ?`, dproto.InterfaceType_PHISYCAL.String()).
		Group(`object_id`).
		Select(); err != nil {
		return err
	}
	for i := range counts {
		for j := range rows {
			if rows[j]["id"].(int64) == counts[i].ObjectID {
				rows[j]["ports"] = counts[i].Count
				break
			}
		}
	}

	// todo: svis?
	// todo: laggs?

	return nil
}
