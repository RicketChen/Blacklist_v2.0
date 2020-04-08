module Blacklist_v2.0

go 1.14

require (
	MyPackage.com/ApiRequest v0.0.0-00010101000000-000000000000
	MyPackage.com/MyLOG v0.0.0-00010101000000-000000000000
	github.com/gin-gonic/gin v1.6.2
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f
	github.com/lestrrat/go-strftime v0.0.0-20180220042222-ba3bf9c1d042 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sirupsen/logrus v1.5.0
)

replace MyPackage.com/ApiRequest => ../ApiRequest

replace MyPackage.com/MyLOG => ../MyLOG