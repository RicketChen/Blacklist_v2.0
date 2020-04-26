package esPackage

import (
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

var esLogger *logrus.Logger

func EsInit(logger *logrus.Logger, url string) {
	esLogger = logger

	elastic.NewClient()

}
