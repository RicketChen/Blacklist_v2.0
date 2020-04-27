package esPackage

import (
	"context"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

var esCtx *context.Context

type EsInfo struct {
	esHost      string
	esLogger    *logrus.Logger
	esCtx       context.Context
	indicesInfo []IndexInfo
	esClient    *elastic.Client
}
type IndexInfo struct {
	IndexName string
	EsType    string
	Mapping   string
}

func (esInfoTemp *EsInfo) EsSetInfo(logger *logrus.Logger, url string) {

	esInfoTemp.EsSetLogger(logger)
	esInfoTemp.esCtx = context.Background()
	esInfoTemp.EsSetUrl(url)
}
func (esInfoTemp *EsInfo) EsInit() (*elastic.PingResult, int, error) {

	//	esInfoTemp.indicesInfo = make([]IndexInfo,10)

	//	esInfoTemp.indicesInfo = new([10]IndexInfo)

	esHost := esInfoTemp.esHost
	esLogger := esInfoTemp.esLogger
	//	client := esInfoTemp.esClient
	var err error
	esInfoTemp.esClient, err = elastic.NewClient(elastic.SetURL(esHost), elastic.SetSniff(false))
	if err != nil {
		esLogger.Fatal("client error :", err)
		return nil, 0, err
	}
	info, code, err := esInfoTemp.esClient.Ping(esHost).Do(esInfoTemp.esCtx)
	if err != nil {
		esLogger.Fatal("ping error :", err)
		return nil, 0, err
	}
	esInfoTemp.esLogger.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)
	return info, code, nil

}

func (esInfoTemp *EsInfo) EsSetLogger(logger *logrus.Logger) {
	esInfoTemp.esLogger = logger
}
func (esInfoTemp *EsInfo) EsSetUrl(url string) {
	esInfoTemp.esHost = url
}
func (esInfoTemp *EsInfo) EsCreateIndex(indexName string) error {
	client := esInfoTemp.esClient
	ctx := esInfoTemp.esCtx
	ret, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		esInfoTemp.esLogger.WithFields(logrus.Fields{
			"err":       err,
			"indexName": indexName,
		}).Warning("Error occur while creating index!")
		return err
	}
	if ret != false {
		esInfoTemp.esLogger.WithFields(logrus.Fields{
			"exists":    ret,
			"indexName": indexName,
		}).Warning("Index already exists,create failed")
		return err
	}

	_, err = client.CreateIndex(indexName).Do(ctx)

	if err != nil {
		esInfoTemp.esLogger.WithFields(logrus.Fields{
			"indexName": indexName,
			"errMsg":    err,
		}).Warning("Create index failed!")
	}
	esInfoTemp.esLogger.WithFields(logrus.Fields{
		"indexName": indexName,
	}).Info("Create index success")
	return nil
}
func (esInfoTemp *EsInfo) EsSetIndex(indexName string) (*IndexInfo, bool, error) {

	temp := &append(esInfoTemp.indicesInfo, IndexInfo{
		IndexName: indexName,
	})[0]
	fmt.Println(esInfoTemp.esClient)
	ret, err := esInfoTemp.esClient.IndexExists(indexName).Do(esInfoTemp.esCtx)
	if err != nil {
		return nil, ret, err
	}
	return temp, ret, nil
}

func (Index *IndexInfo) SetIndexName(indexName string) {
	//	fmt.Println(Index.IndexName)
	//	append()
	Index.IndexName = indexName
}
func (Index *IndexInfo) SetMapping(mapping string) {
	Index.Mapping = mapping
}
