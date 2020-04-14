package main

import (
	"bytes"
	"flag"
	"github.com/bitly/go-simplejson"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"io"
	"log"
	"net"
	"os"
	"regexp"
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

func FlagArgs(port *string) {
	*port = *flag.String("port", "8080", "specify a http port for use,default port:8080")
	flag.Parse()
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

		bufBody, _ := context.Get("bufBody")

		logger.WithFields(logrus.Fields{
			"role":       "server",
			"statusCode": status,
			"latency":    latency,
			"clientID":   context.Request.RemoteAddr,
			"Method":     context.Request.Method,
			"Path":       context.Request.URL.Path,
		}).Info(bytes.NewBuffer(bufBody.([]byte)).String())

		interfaceResp, _ := context.Get("response")
		Resp := interfaceResp.(*fasthttp.Response)
		logger.WithFields(logrus.Fields{
			"role":   "client",
			"status": Resp.StatusCode(),
		}).Info(bytes.NewBuffer(Resp.Body()).String())
	}
}

func setRotate(logger *logrus.Logger) logrus.Hook {

	logWrite, _ := rotatelogs.New(
		"%Y-%m-%d/%H/"+"%Y%m%d%H.log",
		rotatelogs.WithLinkName("./fileLog.log"),
		rotatelogs.WithMaxAge(time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)

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

var serverLog *logrus.Logger

func main() {

	var port string
	var IPAddress string
	GetLocalIpAddress(&IPAddress, "WLAN")
	FlagArgs(&port)
	serverLog = logrus.New()

	gin.SetMode(gin.ReleaseMode)
	if gin.Mode() == gin.DebugMode {
		textFormatter := new(logrus.TextFormatter)
		textFormatter.TimestampFormat = "2006/01/02-15:04:05"
		textFormatter.ForceColors = false
		serverLog.SetFormatter(textFormatter)
	} else {
		jsonFormatter := new(logrus.JSONFormatter)
		jsonFormatter.TimestampFormat = "2006/01/02-15:04:05"
		serverLog.SetFormatter(jsonFormatter)
	}

	serverLog.SetLevel(logrus.InfoLevel)

	serverLog.AddHook(&DefaultFieldHook{})

	setRotate(serverLog)

	router := gin.New()

	router.Use(Logger(serverLog))

	router.POST("/vos3000/blacklist", blacklistHandler)
	router.Run(IPAddress + ":" + port)

}

func blacklistHandler(ctx *gin.Context) {

	var byteResult []byte

	requestUrl := "http://47.112.31.183:9993/vos_yt/blacklist/vos30002160"

	BodyGet := ctx.Request.Body

	jsBody, _ := simplejson.NewFromReader(BodyGet)

	bufBody, _ := jsBody.MarshalJSON()

	ctx.Set("bufBody", bufBody)

	strCallee, _ := jsBody.GetPath("RewriteE164Req", "calleeE164").String()

	lenCallee := len(strCallee)
	response := fasthttp.AcquireResponse()
	if lenCallee >= 11 {

		//	strPrefix := strCallee[:lenCallee-11]

		strSuffix := strCallee[lenCallee-11:]

		bufBody, _ := jsBody.MarshalJSON()
		newJsBody, _ := simplejson.NewJson(bufBody)

		newJsBody.SetPath([]string{"RewriteE164Req", "calleeE164"}, strSuffix)

		byteNewBody, _ := newJsBody.MarshalJSON()

		request := fasthttp.AcquireRequest()
		request.Header.SetMethod("POST")
		request.Header.SetContentType("application/json")
		request.SetRequestURI(requestUrl)
		request.SetBody(byteNewBody)

		fasthttp.Do(request, response)

		byteResp := response.Body()

		jsResp, _ := simplejson.NewJson(byteResp)

		strBackCallee, _ := jsResp.GetPath("RewriteE164Req", "calleeE1644").String()

		//
		byteResult, _ = jsBody.MarshalJSON()
		if len(strBackCallee) != 11 {
			byteResult, _ = jsResp.MarshalJSON()
		}
	} else {
		jsBody.SetPath([]string{"RewriteE164Req", "calleeE1644"}, "Wrong"+strCallee)
		byteResult, _ = jsBody.MarshalJSON()
	}

	ctx.Set("response", response)

	ctx.Header("Content-Type", "application/json")
	ctx.Writer.Write(byteResult)
}
