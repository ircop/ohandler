package taskparser

import (
	"github.com/go-pg/pg"
	"github.com/ircop/ohandler/db"
	"github.com/ircop/ohandler/handler"
	"github.com/ircop/ohandler/logger"
	"github.com/ircop/ohandler/models"
	"github.com/pmezard/go-difflib/difflib"
	"strings"
)

func processConfig(newConfig string, mo *handler.ManagedObject, dbo models.Object) {
	// nothing to compare, whoops.
	if newConfig == "" {
		return
	}

	// select and compare with last config
	var prevCfg models.Config
	err := db.DB.Model(&prevCfg).Where(`object_id = ?`, dbo.ID).Last()
	if err != nil && err != pg.ErrNoRows {
		logger.Err("%s: Failed to select prev.config: %s", dbo.Name, err.Error())
		return
	}
	//logger.Debug("GOT ID = %d", prevCfg.ID)
	if err == pg.ErrNoRows {
		// just insert new config
		cfg := models.Config{
			Config:newConfig,
			ObjectID:dbo.ID,
			PrevDiff:"",
		}
		if err = db.DB.Insert(&cfg); err != nil {
			logger.Err("%s: Failed to save new config: %s", dbo.Name, err.Error())
		}
		return
	}

	diff := difflib.ContextDiff{
		A: prepareConfig(prevCfg.Config),
		B: prepareConfig(newConfig),
		Context:3,
		Eol: "\n",
	}
	result, err := difflib.GetContextDiffString(diff)
	if err != nil {
		logger.Err("%s: Failed to diff old+new configs: %s", dbo.Name, err.Error())
		return
	}

	if result != "" {
		logger.Update("%s: Config was changed: \n%s", dbo.Name, result)
		newone := models.Config{
			Config:newConfig,
			PrevDiff:result,
			ObjectID:dbo.ID,
		}
		if err = db.DB.Insert(&newone); err != nil {
			logger.Update("%s: Failed to insert new config: %s", dbo.Name, err.Error())
			logger.Err("%s: Failed to insert new config: %s", dbo.Name, err.Error())
			return
		}
	}
}

func prepareConfig(s string) []string {
	lines := strings.SplitAfter(s, "\n")
	var l2 []string
	for i := range lines {
		// stupid cisco changes configured value ntp clock-period =\
		str := strings.Trim(lines[i], " ")
		if str == "" {
			continue
		}
		if strings.Contains(lines[i], "ntp clock-period") {
			//lines = append(lines[:i], lines[i+1:]...)
			continue
		}
		//lines[i] = str
		l2 = append(l2, lines[i])
	}
	//lines[len(lines)-1] += "\n"
	l2[len(l2)-1] += "\n"
	return l2
}

