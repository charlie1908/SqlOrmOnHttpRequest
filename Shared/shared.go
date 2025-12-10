package shared

import (
	"fmt"
	"github.com/spf13/viper"
	"strconv"
	"strings"
	"time"
)

type Byte []byte

func (b Byte) byteAppendComma(text string) []byte {
	for _, num := range strings.Split(text, ",") {
		intNum, _ := strconv.Atoi(strings.TrimSpace(num))
		b = append(b, byte(intNum))
	}
	return b
}

type configuration struct {
	SQLURL                 string
	SQLTIMEOUT             time.Duration
	SECRETKEY              string
	REDISPASSWORD          string
	TOKENEXPIRETIME        time.Duration
	REFRESHTOKENEXPIRETIME time.Duration
	REDISURL               string
	BYTES                  []byte
	ELASTICUSER            string
	ELASTICPASSWORD        string
	ELASTICURL             string
	ELASTICLOGININDEX      string
	ELASTICAUDITINDEX      string
	ELASTICERRORINDEX      string
	RABBITMQHOST           string
	RABBITMQUSER           string
	RABBITMQPASSWORD       string
	RABBITMQPORT           int
	LANG                   string
}

var byteStrings = getConfigPrm("encryption", "bytes").(string)
var bytes = make(Byte, 0).byteAppendComma(byteStrings)
var Config = configuration{
	SQLURL:                 getConfigPrm("database", "sqlurl").(string),
	SQLTIMEOUT:             time.Second * time.Duration(getConfigPrm("database", "sqltimeout").(int)),
	SECRETKEY:              "2909012565820034",
	REDISPASSWORD:          getConfigPrm("redis", "password").(string),
	REDISURL:               getConfigPrm("redis", "url").(string),
	TOKENEXPIRETIME:        time.Minute * time.Duration(getConfigPrm("redis", "tokenExpireTime").(int)),
	REFRESHTOKENEXPIRETIME: time.Minute * time.Duration(getConfigPrm("redis", "refreshTokenExpireTime").(int)),
	BYTES:                  bytes,
	ELASTICUSER:            getConfigPrm("elastic", "username").(string),
	ELASTICPASSWORD:        getConfigPrm("elastic", "password").(string),
	ELASTICURL:             getConfigPrm("elastic", "url").(string),
	ELASTICLOGININDEX:      getConfigPrm("elastic", "elasticLoginIndex").(string),
	ELASTICAUDITINDEX:      getConfigPrm("elastic", "elasticAuditIndex").(string),
	ELASTICERRORINDEX:      getConfigPrm("elastic", "elasticErrorIndex").(string),
	RABBITMQHOST:           getConfigPrm("rabbitmq", "host").(string),
	RABBITMQUSER:           getConfigPrm("rabbitmq", "username").(string),
	RABBITMQPASSWORD:       getConfigPrm("rabbitmq", "password").(string),
	RABBITMQPORT:           getConfigPrm("rabbitmq", "port").(int),
	LANG:                   getConfigPrm("", "lang").(string),
}

/*func getConfigPrm(root string, prm string) interface{} {
	viper.SetConfigName("config")
	viper.AddConfigPath("./Shared")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}
	return viper.Get(root + "." + prm)
}*/

func getConfigPrm(root string, prm string) interface{} {
	viper.SetConfigName("config")
	viper.AddConfigPath("./Shared")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	// 🔧 Kök seviyedeki key'ler için özel durum
	if root == "" {
		return viper.Get(prm)
	}
	return viper.Get(root + "." + prm)
}
