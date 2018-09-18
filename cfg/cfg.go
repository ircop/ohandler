package cfg

import (
	"github.com/spf13/viper"
)

// Cfg is struct for handling config parameters
type Cfg struct {
	NatsURL			string
	NatsTasks		string
	NatsReplies		string

	LogDir			string
	LogDebug		bool

	DBHost				string
	DBPort				int
	DBName				string
	DBUser				string
	DBPassword			string
}

// NewCfg reads config with given path
func NewCfg(path string) (*Cfg, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	c := new(Cfg)

	c.NatsURL = viper.GetString("nats.url")
	c.NatsTasks = viper.GetString("nats.tasks-chan")
	c.NatsReplies = viper.GetString("nats.replies-chan")

	c.DBHost = viper.GetString("db.host")
	c.DBPort = viper.GetInt("db.port")
	c.DBName = viper.GetString("db.name")
	c.DBPassword = viper.GetString("db.password")
	c.DBUser = viper.GetString("db.user")

	c.LogDir = viper.GetString("log.dir")
	c.LogDebug = viper.GetBool("log.debug")

	return c, nil
}
