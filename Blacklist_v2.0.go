package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
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

func FlagArgs(level *int, logEnable *bool, logFilename *string, port *string) {
	*level = *flag.Int("level", 3, "specify log level,0 for no log,5 for TRACE to ERROR,4 for DEBUG to ERROR,3 for INFO to ERROR(default),2 for WARNING to ERROR,1 for ERROR")
	*logFilename = *flag.String("log", "", "specific a filename as log filename")
	*logEnable = *flag.Bool("l", true, "enable/disable log file by with or without -l")
	*port = *flag.String("port", "8080", "specify a http port for use,default port:8080")
	flag.Parse()
}
func Logger1() gin.HandlerFunc {
	logClient := logrus.New()

	//禁止logrus的输出
	apiLogPath := "./api.log"
	src, err := os.OpenFile(apiLogPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("err", err)
	}
	logClient.Out = src
	logClient.SetLevel(logrus.DebugLevel)
	logWriter, err := rotatelogs.New(
		apiLogPath+".%Y-%m-%d-%H-%M.log",
		rotatelogs.WithLinkName(apiLogPath),       // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(7*24*time.Hour),     // 文件最大保存时间
		rotatelogs.WithRotationTime(24*time.Hour), // 日志切割时间间隔
	)
	writeMap := lfshook.WriterMap{
		logrus.InfoLevel:  logWriter,
		logrus.FatalLevel: logWriter,
	}
	lfHook := lfshook.NewHook(writeMap, &logrus.JSONFormatter{})
	logClient.AddHook(lfHook)

	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		// 处理请求
		c.Next()
		// 结束时间
		end := time.Now()
		//执行时间
		latency := end.Sub(start)

		path := c.Request.URL.Path

		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		logClient.Infof("| %3d | %13v | %15s | %s  %s |",
			statusCode,
			latency,
			clientIP,
			method, path,
		)
	}
}

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		statusCodeColor := green
		methodColor := cyan
		t := time.Now()

		context.Set("example", "test")

		//before request
		context.Next()

		//after request
		latency := time.Since(t)
		//log.Printf("%13v",latency)

		//get status
		status := context.Writer.Status()
		if status == 404 {
			statusCodeColor = red
		} else {
			statusCodeColor = yellow
		}
		//get request method
		method := context.Request.Method
		if method != "POST" {
			methodColor = blue
		}

		if logFile == nil {
			os.MkdirAll(logFilePath, 0755)
			logFile, _ = os.OpenFile(logFilePath+logFilename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
			logger.SetOutput(io.MultiWriter(logFile, os.Stdout))
		}

		logger.Info(fmt.Sprintf("%v |%s %3d %s| %10v | %15s |%s %-5s %s %#v\n",
			t.Format("2006/01/01 - 15:04:05"),
			statusCodeColor, status, reset,
			latency,
			context.Request.RemoteAddr,
			methodColor, context.Request.Method, reset,
			context.Request.URL.Path,
		))
	}
}

var logFile *os.File
var logFilePath string
var logFilename string

func main() {

	newlog := logrus.New()

	gin.ForceConsoleColor()
	newlog.Formatter = new(logrus.TextFormatter)
	//	newlog.Formatter = new(logrus.JSONFormatter)

	newlog.SetLevel(logrus.InfoLevel)

	newlog.Formatter.(*logrus.TextFormatter).DisableTimestamp = true
	newlog.Formatter.(*logrus.TextFormatter).ForceColors = true

	var firstMinute int = 0
	go func() {
		for {
			if firstMinute != time.Now().Minute() {
				timeHour := time.Now().Format("2006-01-02")
				timeMin := time.Now().Format("2006-01-02-15")
				filePath := "./" + timeHour + "/"
				fileName := strings.Replace(timeMin, ":", "", -1)
				if logFile != nil {
					logFile.Close()
					logFile = nil
				}
				firstMinute = time.Now().Minute()
				logFilePath = filePath
				logFilename = fileName + ".log"
			}
			time.Sleep(time.Second)
		}
	}()
	/*	go func() {
		for {
			timeHour := time.Now().Format("2006-01-02")
			timeMin := time.Now().Format("2006-01-02-15")
			nowTime := time.Now().Hour()
			if nowTime >= firstTime.Hour() {
				filePath := "./" + timeHour + "/"
				fileName := strings.Replace(timeMin, ":", "", -1)
				os.MkdirAll(filePath, 755)
				if logFile != nil {
					if err := logFile.Close(); err != nil {
						log.Fatalln(err)
					}
				}
				logFile, _ = os.OpenFile(filePath+fileName+".log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 755)

				newlog.SetOutput(io.MultiWriter(logFile, os.Stdout))
				firstTime = time.Now()
			}
			time.Sleep(time.Second)
		}
	}()*/

	var level int
	var logEnable bool
	var logFilename string
	var port string

	var IPAddress string
	GetLocalIpAddress(&IPAddress, "WLAN")

	FlagArgs(&level, &logEnable, &logFilename, &port)

	//	MyLOG.LogInit(level, logEnable, logFilename)

	//	MyLOG.Info.Println("TEST")

	router := gin.New()
	//	router := gin.Default()
	router.Use(Logger(newlog))

	router.POST("/vos3000/blacklist", mainHandler)
	router.Run(IPAddress + ":" + port)
	for {

	}
}

func mainHandler(ctx *gin.Context) {

	ctx.JSON(200, gin.H{"a": "data"})
}
