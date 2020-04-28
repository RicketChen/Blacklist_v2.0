package esPackage

import (
	"context"
	"github.com/bitly/go-simplejson"
	"github.com/olivere/elastic/v7"
	errors2 "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

var esCtx *context.Context

type EsInfo struct {
	EsHost      string
	esLogger    *logrus.Logger
	EsCtx       context.Context
	IndicesInfo []IndexInfo
	EsClient    *elastic.Client
}
type IndexInfo struct {
	IndexName string
	EsType    string
	Mapping   string
}

var once sync.Once
var esInstance *EsInfo

func GetEsInstance() *EsInfo {
	once.Do(func() {
		esInstance = &EsInfo{}
	})
	return esInstance
}

func (esInfoTemp *EsInfo) EsSetInfo(logger *logrus.Logger, url string) {

	esInfoTemp.EsSetLogger(logger)
	esInfoTemp.EsCtx = context.Background()
	esInfoTemp.EsSetUrl(url)
}
func (esInfoTemp *EsInfo) EsInit() (*elastic.PingResult, int, error) {

	esHost := esInfoTemp.EsHost
	esLogger := esInfoTemp.esLogger
	//	client := esInfoTemp.EsClient
	var err error
	esInfoTemp.EsClient, err = elastic.NewClient(elastic.SetURL(esHost), elastic.SetSniff(false))
	if err != nil {
		esLogger.Fatal("client error :", err)
		return nil, 0, err
	}
	info, code, err := esInfoTemp.EsClient.Ping(esHost).Do(esInfoTemp.EsCtx)
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
	esInfoTemp.EsHost = url
}
func (esInfoTemp *EsInfo) EsCreateIndex(indexName string, mapping string) error {
	client := esInfoTemp.EsClient
	ctx := esInfoTemp.EsCtx
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
		return errors2.Errorf("index [%s] already exists !", indexName)
	}

	_, err = client.CreateIndex(indexName).BodyString(mapping).Do(ctx)

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
func (esInfoTemp *EsInfo) EsSetIndex(indexName string, mapping string) (*IndexInfo, error) {

	esInfo := GetEsInstance()

	index := IndexInfo{
		IndexName: indexName,
	}

	esInfoTemp.IndicesInfo = append(esInfoTemp.IndicesInfo, IndexInfo{
		IndexName: indexName,
	})

	err := esInfo.EsCreateIndex(indexName, mapping)

	return &index, err
}

func (Index *IndexInfo) SetIndexName(indexName string) {
	//	fmt.Println(Index.IndexName)
	//	append()
	Index.IndexName = indexName
}
func (Index *IndexInfo) InsertDoc(phoneNums string, numsType string) error {
	esInfo := GetEsInstance()
	client := esInfo.EsClient
	ctx := esInfo.EsCtx
	logger := esInfo.esLogger

	jsEsBody := simplejson.New()
	nowUnixTime := time.Now().UnixNano() / 1e6
	jsEsBody.Set("timestamp", nowUnixTime)
	jsEsBody.SetPath([]string{"phoneInfo", "nums"}, phoneNums)
	jsEsBody.SetPath([]string{"phoneInfo", "numsType"}, numsType)

	indexResponse, err := client.Index().Index(Index.IndexName).Id(strconv.Itoa(int(nowUnixTime))).BodyJson(jsEsBody).Do(ctx)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"errMsg": err,
			"id":     nowUnixTime,
			"phoneInfo": logrus.Fields{
				"nums":     phoneNums,
				"numsType": numsType,
			},
		}).Error("Index a document failed!")
		return err
	}
	logger.WithFields(logrus.Fields{
		"id":       nowUnixTime,
		"response": indexResponse,
		"phoneInfo": logrus.Fields{
			"nums":     phoneNums,
			"numsType": numsType,
		},
	}).Info("Index a document success!")
	return nil
}
func (Index *IndexInfo) SearchDoc(nums string) {

}
