package logging

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

var (
	logger = logrus.New()
	isInit = false
)

func initLogger() {
	var err error
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	logger.Level, err = logrus.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		logger.Level = logrus.InfoLevel
	}
}

func GetLogger() *logrus.Logger {
	if !isInit {
		initLogger()
		isInit = true
	}
	return logger
}
