package controllers

import "github.com/ircop/dproto"

type DiscoveryProblemsController struct {
	HTTPController
}

func (c *DiscoveryProblemsController) GET(ctx *HTTPContext) {
	probs := dproto.DiscoveryProblem_name

	list := make([]interface{}, 0)
	for id, name := range probs {
		item := make(map[string]interface{})
		item["id"] = id
		item["title"] = name
		list = append(list, item)
	}

	result := make(map[string]interface{})
	result["problems"] = list

	WriteJSON(ctx.W, result)
}