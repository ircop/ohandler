package controllers

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/ircop/discoverer/util/mac"
	"github.com/ircop/dproto"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"net"
	"regexp"
	"strconv"
	"strings"
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

func (c *ObjectsController) formatWhere(ctx *HTTPContext, query *orm.Query) {
	re, err := regexp.Compile(`(\d+)`)
	if err != nil {
		logger.RestErr("Cannot compile \\d+ regex: %s", err.Error())
		return
	}

	ipname := strings.Trim(ctx.Params["ipname"], " ")

	// find out segments, if any
	segString := ctx.Params["segments"]
	if segString != "" && segString != "[]" {

		matches := re.FindAllStringSubmatch(segString, -1)
		segmentIDs := make([]int64,0)
		for i := range matches {
			idStr := matches[i][0]
			segID, err := strconv.ParseInt(idStr, 10, 64)
			if err == nil {
				segmentIDs = append(segmentIDs, segID)
			}
		}

		if len(segmentIDs) > 0 {
			query.Join(`JOIN object_segments AS os ON os.object_id = object.id`).
				Where(`os.segment_id in (?)`, pg.In(segmentIDs))
		}
	}

	// models?
	modString := ctx.Params["models"]
	if modString != "" && modString != "[]" {
		modString = strings.Replace(modString, "[", "", -1)
		modString = strings.Replace(modString, "]", "", -1)
		models := strings.Split(modString, " ")
		query.Where(`model in (?)`, pg.In(models))
	}

	if ipname != "" {
		// search by CIDR, if any
		_,_, err := net.ParseCIDR(ipname)
		if err == nil {
			query.Join(`JOIN ips AS ips ON ips.object_id = object.id`).
				Where(`? >>= inet(host(addr))`, ipname).
				Group(`object.id`)
			return
		}

		// search by mac
		if Mac.IsMac(ipname) {
			omacQuery := db.DB.Model(&models.ObjectMac{}).Where(`mac = ?`, ipname).Column(`object_id`)
			query.Where(`id = (?)`, omacQuery)
			return
		}

		// todo: search by ip NOT ONLY MGMT
		// search by ip OR ip part OR name OR name part
		if err != nil {
			query.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
				q.Where(`mgmt like ?`, "%"+ipname+"%").
					WhereOr(`name like ?`, "%"+ipname+"%")
				return q, nil
			})
		}
	}

	dproblems := strings.Trim(ctx.Params["dproblems"], " ")
	if dproblems != "" && dproblems != "[]" {
		matches := re.FindAllStringSubmatch(dproblems, -1)
		for i := range matches {
			idStr := matches[i][0]
			code, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				continue
			}

			switch code {
			case int64(dproto.DiscoveryProblem_NO_NEIGHBORS):
				query.Join(`LEFT JOIN lldp_neighbors n1 on n1.neighbor_id = object.id`).
					Join(`LEFT JOIN lldp_neighbors n2 on n2.object_id = object.id`).
					Where(`n1.id IS NULL`).Where(`n2.id IS NULL`)
				break
			case int64(dproto.DiscoveryProblem_NO_VLANS):
				query.Join(`LEFT JOIN object_vlans ovc on ovc.object_id = object.id`).
					Where(`ovc.id IS NULL`)
				break
			case int64(dproto.DiscoveryProblem_NO_UPLINK):
				query.Where(`object.uplink_id IS NULL`)
				break
			case int64(dproto.DiscoveryProblem_NO_INTERFACES):
				query.Join(`LEFT JOIN interfaces ifs ON ifs.object_id = object.id`).
					Where(`ifs.id IS NULL`)
				break
			}
		}
	}

	alive := strings.Trim(ctx.Params["alive"], " ")
	if alive == "true" {
		query.Where(`alive = ?`, true)
	}
	if alive == "false" {
		query.Where(`alive = ?`, false)
	}
}

func (c *ObjectsController) POST(ctx *HTTPContext) {
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

	var objects []models.Object
	query := db.DB.Model(&objects)

	// first, form WHERE clauses
	c.formatWhere(ctx, query)

	// second, select count:
	cnt, err := query.Count()
	if err != nil {
		logger.RestErr(`Cannot select objects count: %s`, err.Error())
		InternalError(ctx.W, err.Error())
		return
	}
	if cnt == 0 {
		results := make(map[string]interface{})
		results["total"] = cnt
		results["rows"] = make([]string,0)
		WriteJSON(ctx.W, results)
		return
	}

	// third, form ORDER clause and select objects
	oField := ctx.Params["sortField"]
	oOrder := ctx.Params["sortOrder"]
	ord := "ASC"
	if oOrder == "descending" {
		ord = "DESC"
	}

	switch oField {
	case "id":
		query.Order(`id `+ord)
		break
	case "name":
		query.OrderExpr(`natsort("name") ` + ord)
		break
	case "mgmt":
		query.Order(`mgmt ` + ord)
		break
	case "alive":
		query.Order(`alive ` + ord)
		break
	case "model":
		query.Order(`model ` + ord)
		break
	default:
		query.Order(`id ASC`)
		break
	}

	err = query.Limit(int(limit)).Offset(int(offset)).Select()
	if err != nil {
		if err == pg.ErrNoRows {
			NotFound(ctx.W)
			return
		}
		logger.RestErr(`Cannot select objects: %s`, err.Error())
		InternalError(ctx.W, err.Error())
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
		ReturnError(ctx.W, err.Error(), true)
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
		InternalError(ctx.W, err.Error())
		return
	}

	// Count links: 2 times, because of 'object1 or object2'
	var lcount1 []linkCount
	var lcount2 []linkCount
	if err := db.DB.Model(&lcount1).ColumnExpr(`object1_id as oid`).ColumnExpr(`count(*) as cnt`).
		Where(`object1_id in (?)`, pg.In(ids)).Group(`object1_id`).Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
		return
	}
	if err := db.DB.Model(&lcount2).ColumnExpr(`object2_id as oid`).ColumnExpr(`count(*) as cnt`).
		Where(`object2_id in (?)`, pg.In(ids)).Group(`object2_id`).Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
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
	WriteJSON(ctx.W, results)
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

	ofield := ctx.Params["sortField"]
	oorder := ctx.Params["sortOrder"]
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
		InternalError(ctx.W, err.Error())
		return
	}

	var objects []models.Object
	_, err = db.DB.Query(&objects, `select * from objects order by ` + order + ` limit ? offset ?`, limit, offset)
	if err != nil {
		logger.RestErr("Error selecting objects: %s", err.Error())
		InternalError(ctx.W, err.Error())
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
			ReturnError(ctx.W, err.Error(), true)
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
		InternalError(ctx.W, err.Error())
		return
	}

	// Count links: 2 times, because of 'object1 or object2'
	var lcount1 []linkCount
	var lcount2 []linkCount
	if err := db.DB.Model(&lcount1).ColumnExpr(`object1_id as oid`).ColumnExpr(`count(*) as cnt`).
		Where(`object1_id in (?)`, pg.In(ids)).Group(`object1_id`).Select(); err != nil {
			ReturnError(ctx.W, err.Error(), true)
		return
	}
	if err := db.DB.Model(&lcount2).ColumnExpr(`object2_id as oid`).ColumnExpr(`count(*) as cnt`).
		Where(`object2_id in (?)`, pg.In(ids)).Group(`object2_id`).Select(); err != nil {
		ReturnError(ctx.W, err.Error(), true)
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
	WriteJSON(ctx.W, results)
	//returnOk(ctx.W)
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
