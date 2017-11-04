package logging

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

var (
	logger = logrus.New()
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
	initLogger()
	return logger
}
