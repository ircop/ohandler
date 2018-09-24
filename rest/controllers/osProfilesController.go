package controllers

import (
	"github.com/ircop/discoverer/dproto"
	"sort"
)

type OsProfilesController struct {
	HTTPController
}

func (c *OsProfilesController) GET(ctx *HTTPContext) {
	// return all profiles...
	result := make(map[string]interface{}, 1)
	profiles := make([]map[string]interface{}, 0)
	for id, name := range dproto.ProfileType_name {
		prof := make(map[string]interface{})
		prof["id"] = id
		prof["title"] = name
		profiles = append(profiles, prof)
	}

	//sort.Slice(aps, func(i, j int) bool { return aps[i].Title < aps[j].Title })
	sort.Slice(profiles, func(i, j int) bool { return profiles[i]["title"].(string) < profiles[j]["title"].(string) })

	result["profiles"] = profiles

	writeJSON(ctx.w, result)
}
