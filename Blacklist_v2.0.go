package main

import (
	"Blacklist_v2.0/esPackage"
	"context"
	"flag"
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"
)

func GetLocalIpAddress(IP *string, InterfaceName string) {

	var Iface *net.Interface
	var err error
	InterfaceNameArray := []string{
		"eth0",
		"WLAN",
	}
	if InterfaceName != "" {
		Iface, err = net.InterfaceByName(InterfaceName)
	}
	if err != nil || InterfaceName == "" {
		if err != nil {
			log.Println("Specify Interface name not found!Error:", err)
		}
		for _, tempInterface := range InterfaceNameArray {
			Iface, err = net.InterfaceByName(tempInterface)
			if err == nil {
				break
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}

	IPAddressArray, _ := Iface.Addrs()
	vaild, _ := regexp.Compile("^([0-9]+\\.)*[0-9]*")
	var IpAddress []string
	for _, AddrStr := range IPAddressArray {
		IpAddress = vaild.FindAllString(AddrStr.String(), -1)
		if IpAddress[0] != "" {
			*IP = IpAddress[0]
			break
		}
	}
}

func FlagArgs() (*int, *bool, *int) {
	port := flag.Int("port", 8080, "specify a http port for use,default port:8080")
	debug := flag.Bool("debug", false, "using debug mode(default no)")
	loglevel := flag.Int("level", 2, "specify a log level,0 - 5 for trace,debug,info,warning,error,fatal,panic")
	flag.Parse()
	return port, debug, loglevel
}

func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		t := time.Now()

		//before request
		context.Next()

		//after request
		latency := time.Since(t)

		//get status
		status := context.Writer.Status()

		//获取请求体接口
		interfaceRequestBody, _ := context.Get("requestBody")

		//请求体接口转json格式
		jsRequestBody := interfaceRequestBody.(*simplejson.Json)

		callId, _ := jsRequestBody.GetPath("RewriteE164Req", "callId").Int()
		calleeE164, _ := jsRequestBody.GetPath("RewriteE164Req", "calleeE164").String()
		callerE164, _ := jsRequestBody.GetPath("RewriteE164Req", "callerE164").String()

		logger.WithFields(logrus.Fields{
			"server": logrus.Fields{
				"statusCode": status,
				"latency":    latency.String(),
				"clientID":   context.Request.RemoteAddr,
				"Method":     context.Request.Method,
				"Path":       context.Request.URL.Path,
				"requestBody": logrus.Fields{
					"callId":     callId,
					"calleeE164": calleeE164,
					"callerE164": callerE164,
				}}}).Debug()

		strResponse, msg := responseHandle(context)
		logger.WithFields(logrus.Fields{
			"client": logrus.Fields{
				"response": strResponse,
			}}).Debug(msg)
	}
}
func responseHandle(context *gin.Context) (logrus.Fields, string) {

	//获取响应的错误信息接口
	interfaceErr, err := context.Get("responseErr")
	//判断是否有错误信息产生
	if err == true {
		return logrus.Fields{
			"responseErr": interfaceErr.(error).Error(),
		}, "response error"
	}

	//获取响应的消息接口
	interfaceResponse, _ := context.Get("response")
	//接口格式转换
	response, _ := interfaceResponse.(*fasthttp.Response)
	//转json格式
	jsResponse, jsErr := simplejson.NewJson(response.Body())
	if jsErr != nil {
		//响应信息转json失败，内容非json数据
		return logrus.Fields{
			"statusCode": response.StatusCode(),
			"err":        jsErr.Error(),
		}, "response msg error"
	}

	callId, _ := jsResponse.GetPath("RewriteE164Rsp", "callId").Int()
	calleeE164, _ := jsResponse.GetPath("RewriteE164Rsp", "calleeE164").String()
	callerE164, _ := jsResponse.GetPath("RewriteE164Rsp", "callerE164").String()

	return logrus.Fields{
		"statusCode": response.StatusCode(),
		"responseBody": logrus.Fields{
			"callId":     callId,
			"calleeE164": calleeE164,
			"callerE164": callerE164,
		},
	}, "normal"
}
func setRotate(logger *logrus.Logger) logrus.Hook {

	//设置日志特殊功能
	logWrite, _ := rotatelogs.New(
		//日志存放路径和命名
		"%Y-%m-%d/%H/"+"%Y%m%d%H.log",
		//设置软连接（快捷方式）
		rotatelogs.WithLinkName("./fileLog.log"),
		//WithMaxAge和RotationCount二选一
		//设置日志保存的最长时间
		rotatelogs.WithMaxAge(time.Hour),
		//设置日志保存的个数
		//rotatelogs.WithRotationCount(10),
		//设置日志分割的时间
		rotatelogs.WithRotationTime(time.Hour),
	)

	//设置日志输出方式
	logger.SetOutput(io.MultiWriter(logWrite, os.Stdout))

	return nil
}

type DefaultFieldHook struct {
}

func (hook *DefaultFieldHook) Fire(entry *logrus.Entry) error {
	filePath := time.Now().Format("2006-01-02/15")
	os.MkdirAll(filePath, 0755)
	return nil
}

func (hook *DefaultFieldHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

var ServerLog *logrus.Logger

var esMapping = `{
    "mappings": {
        "properties": {
            "timestamp": {
                "type": "date",
                "format":"epoch_millis"
            },
            "phoneInfo": {
                "type": "object",
                "properties": {
                    "nums": {
                        "type": "text"
                    },
                    "numstype": {
                        "type": "text"
                    }
                }
            }
        }
    }
}`

func main() {

	var IPAddress string
	GetLocalIpAddress(&IPAddress, "WLAN")
	port, debug, logLevel := FlagArgs()

	if *debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	ServerLog = logrus.New()

	if gin.Mode() == gin.DebugMode {
		textFormatter := new(logrus.TextFormatter)
		textFormatter.TimestampFormat = "2006/01/02-15:04:05"
		textFormatter.ForceColors = false
		ServerLog.SetFormatter(textFormatter)
	} else {
		jsonFormatter := new(logrus.JSONFormatter)
		jsonFormatter.TimestampFormat = "2006/01/02-15:04:05"
		jsonFormatter.PrettyPrint = false
		ServerLog.SetFormatter(jsonFormatter)
	}

	ServerLog.SetLevel(logrus.Level(6 - *logLevel))

	ServerLog.AddHook(&DefaultFieldHook{})

	setRotate(ServerLog)

	router := gin.New()

	router.Use(Logger(ServerLog))

	esPackage.EsInit()

	esCtx := context.Background()
	esHost := "http://127.0.0.1:9211"
	client, err := elastic.NewClient(elastic.SetURL(esHost), elastic.SetSniff(false))
	if err != nil {
		ServerLog.Fatal("client error :", err)
	}
	info, code, err := client.Ping(esHost).Do(esCtx)
	if err != nil {
		ServerLog.Fatal("ping error :", err)
	}
	ServerLog.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	indexName := []string{"blacklist", "whitelist", "ytblacklist"}
	for _, temp := range indexName {
		ret, _ := client.IndexExists(temp).Do(esCtx)
		if ret != false {
			ServerLog.WithField("index", temp).Debug("index exists")
			continue
		} else {
			createIndex, err := client.CreateIndex(temp).BodyString(esMapping).Do(esCtx)
			if err != nil {
				ServerLog.WithField("index", temp).Fatal("Create index failed!")
			}
			if createIndex.Acknowledged {
				ServerLog.WithFields(logrus.Fields{
					"index":        temp,
					"Acknowledged": createIndex.Acknowledged,
				})
			}
		}
	}

	/*	for i:=0;i<10;i++ {
			jsEsBody := simplejson.New()
			nowUnixTime := time.Now().UnixNano()
			jsEsBody.Set("timestamp", nowUnixTime)
			phoneNums := 13800138000+i
			jsEsBody.SetPath([]string{"phoneInfo", "nums"}, strconv.Itoa(phoneNums))
			jsEsBody.SetPath([]string{"phoneInfo", "numstype"}, "blacklist")

			client.Index().Index("blacklist").Id(strconv.Itoa(int(nowUnixTime))).BodyJson(jsEsBody).Do(esCtx)
		//	time.Sleep(time.Second)
		}*/

	searchRet, searchErr := client.Search("blacklist").
		Query(elastic.NewTermQuery("phoneInfo.nums", "13800138000")).
		Do(esCtx)
	if searchErr != nil {
		ServerLog.Fatal(searchErr)
	}
	for _, temp := range searchRet.Hits.Hits {
		byteSource, _ := temp.Source.MarshalJSON()
		jsSource, _ := simplejson.NewJson(byteSource)
		nums, _ := jsSource.GetPath("phoneInfo", "nums").String()
		numstype, _ := jsSource.GetPath("phoneInfo", "numstype").String()
		ServerLog.WithFields(logrus.Fields{
			"index":    temp.Index,
			"id":       temp.Id,
			"nums":     nums,
			"numstype": numstype,
		}).Info(*temp.Score)
	}

	return

	relativePath := "/vos3000/blacklist"
	router.POST(relativePath, blacklistHandler)

	ServerLog.WithFields(logrus.Fields{
		"Server":   IPAddress + ":" + strconv.Itoa(*port),
		"Path":     relativePath,
		"Method":   "POST",
		"logLevel": ServerLog.GetLevel().String(),
	}).Print("Server initialized")
	router.Run(IPAddress + ":" + strconv.Itoa(*port))
}

func blacklistHandler(ctx *gin.Context) {

	defer ctx.Request.Body.Close()

	var byteResult []byte

	//	requestUrl := "http://47.112.31.183:9993/vos_yt/blacklist/vos30002160"
	requestUrl := "http://192.168.2.230:9993/vos_yt/blacklist/vos30002160"

	BodyGet := ctx.Request.Body

	jsBody, _ := simplejson.NewFromReader(BodyGet)

	//	bufBody, _ := jsBody.MarshalJSON()

	ctx.Set("requestBody", jsBody)

	strCallee, _ := jsBody.GetPath("RewriteE164Req", "calleeE164").String()

	lenCallee := len(strCallee)
	//fasthttp response params initialize
	response := fasthttp.AcquireResponse()

	defer response.ConnectionClose()
	var responseErr error
	if lenCallee >= 11 {

		//	strPrefix := strCallee[:lenCallee-11]

		strSuffix := strCallee[lenCallee-11:]

		bufBody, _ := jsBody.MarshalJSON()

		newJsBody, _ := simplejson.NewJson(bufBody)

		newJsBody.SetPath([]string{"RewriteE164Req", "calleeE164"}, strSuffix)

		byteNewBody, _ := newJsBody.MarshalJSON()
		//fasthttp request params initialize
		request := fasthttp.AcquireRequest()

		request.Header.SetMethod("POST")

		request.Header.SetContentType("application/json")

		request.SetRequestURI(requestUrl)

		request.SetBody(byteNewBody)

		responseErr = fasthttp.Do(request, response)
		//		responseErr = fasthttp.DoTimeout(request,response,2)

		if responseErr != nil {
			if response.StatusCode() == 200 {
				ctx.Set("responseErr", responseErr)
				response.SetStatusCode(500)
			}

			jsBody.Set("msg", "remote server error")
			byteResult, _ = jsBody.MarshalJSON()

		} else {

			byteResp := response.Body()

			jsResp, _ := simplejson.NewJson(byteResp)

			strBackCallee, _ := jsResp.GetPath("RewriteE164Req", "calleeE1644").String()

			if len(strBackCallee) != 11 {

				byteResult, _ = jsResp.MarshalJSON()

			} else {

				byteResult, _ = jsBody.MarshalJSON()

			}
		}
	} else {

		newJsBody := simplejson.New()

		requestCallId, _ := jsBody.GetPath("RewriteE164Req", "callId").Int()
		requestCallee, _ := jsBody.GetPath("RewriteE164Req", "calleeE164").String()
		requestCaller, _ := jsBody.GetPath("RewriteE164Req", "callerE164").String()

		newJsBody.SetPath([]string{"RewriteE164Rsp", "calleeE164"}, "Wrong"+requestCallee)
		newJsBody.SetPath([]string{"RewriteE164Rsp", "callerE164"}, requestCaller)
		newJsBody.SetPath([]string{"RewriteE164Rsp", "callId"}, requestCallId)

		byteResult, _ = newJsBody.MarshalJSON()

		response.SetBody(byteResult)

	}

	ctx.Set("response", response)

	ctx.Header("Content-Type", "application/json")

	ctx.Writer.Write(byteResult)
}
