package cfg

import (
	"github.com/spf13/viper"
)

// Cfg is struct for handling config parameters
type Cfg struct {
	NatsURL			string
	NatsTasks		string
	NatsReplies		string
	NatsDB			string

	LogDir			string

	DBHost			string
	DBPort			int
	DBName			string
	DBUser			string
	DBPassword		string

	RestIP			string
	RestPort		int
	SslCert    		string
	SslKey     		string
	Ssl				bool
	LogDebug		bool
	DashTemplates	string
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
	c.NatsDB = viper.GetString("nats.db-chan")

	c.DBHost = viper.GetString("db.host")
	c.DBPort = viper.GetInt("db.port")
	c.DBName = viper.GetString("db.name")
	c.DBPassword = viper.GetString("db.password")
	c.DBUser = viper.GetString("db.user")

	c.LogDir = viper.GetString("log.dir")
	c.LogDebug = viper.GetBool("log.debug")

	c.RestIP = viper.GetString("rest.ip")
	c.RestPort = viper.GetInt("rest.port")
	c.Ssl = viper.GetBool("rest.ssl")
	c.SslCert = viper.GetString("rest.cert")
	c.SslKey = viper.GetString("rest.key")
	c.DashTemplates = viper.GetString("rest.dash-templates")

	return c, nil
}
