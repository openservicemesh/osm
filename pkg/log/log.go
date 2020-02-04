package log

import "github.com/golang/glog"

const (
	LvlFatal glog.Level = 0
	LvlError glog.Level = 5
	LvlWarn  glog.Level = 10
	LvlInfo  glog.Level = 15
	LvlDebug glog.Level = 20
	LvlTrace glog.Level = 25
)
