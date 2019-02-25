package main

import (
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/wuzuoliang/micro-backend/MyLogger"
	"github.com/wuzuoliang/micro-backend/Tools/Define"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
)

const (
	ServiceName = "autoftp"
)

var (
	cfgPath    string
	logPath    string
	logLevel   string
	instanceID int
	cfgJson    Cfg
)

func init() {
	var projectConf string
	if runtime.GOOS == "windows" {
		projectConf = "c:\\etc\\autoftp\\auto_ftp.json"
	} else {
		projectConf = "/Users/wuzuoliang/Documents/workspace/go/src/github.com/wuzuoliang/autoftp/auto_ftp.json"
	}
	flag.StringVar(&cfgPath, "c", projectConf, "config_file path,examples linux: /etc/autoftp/auto_ftp.json,"+
		"windows:  c:\\etc\\autoftp\\auto_ftp.json")
	flag.StringVar(&logLevel, "l", "DEBUG", "log level: DEBUG, INFO, ERROR")
	flag.IntVar(&instanceID, "i", 2, " instanceID -i 2")
}

func prepare() {
	data, err := ioutil.ReadFile(cfgPath)
	checkErr(err, "[prepare] ReadFile ", true)
	err = json.Unmarshal([]byte(data), &cfgJson)
	if err != nil {
		checkErr(err, "[prepare] Unmarshal ", true)
	}
	if runtime.GOOS == "windows" {
		logPath = "c:\\log\\autoftp\\autoftp.log"
	} else {
		logPath = "/Users/wuzuoliang/Documents/workspace/go/src/github.com/wuzuoliang/autoftp/autoFtp.log"
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Critical("[main]", "server crash: ", err)
			log.Critical("[main]", "stack: ", string(debug.Stack()))
		}
	}()
	flag.Parse()
	prepare()
	level := log.SetLogPath(ServiceName, logPath, "")
	switch logLevel {
	case Define.LOG_DEBUG:
		level.SetLevel(log.DEBUG, "")
	case Define.LOG_INFO:
		level.SetLevel(log.INFO, "")
	case Define.LOG_NOTICE:
		level.SetLevel(log.NOTICE, "")
	case Define.LOG_WARNING:
		level.SetLevel(log.WARNING, "")
	case Define.LOG_ERROR:
		level.SetLevel(log.ERROR, "")
	case Define.LOG_CRITICAL:
		level.SetLevel(log.CRITICAL, "")
	}
	log.SetBackend(level)
	hasFoundInstance := false
	var ins Instance
	for i := 0; i < len(cfgJson.Instances); i++ {
		if cfgJson.Instances[i].InstanceID == instanceID {
			log.Info("[main]", "load conf", fmt.Sprintf("%+v", cfgJson.Instances[i]))
			ins = cfgJson.Instances[i]
			hasFoundInstance = true
		}
	}
	if !hasFoundInstance {
		log.Error("[main]", "not found instanceID conf", instanceID)
	}
	go finish()
	initInstance(&ins)
	fin := make(chan bool)
	<-fin
	log.Debug("[main]", "instanceID", instanceID, "finish")
}
func finish() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	for {
		s := <-c
		//
		//   TODO   -- add codes which you want to handle before process exit
		//
		log.Info("[finish]", "get signal:", s)
		os.Exit(1)
	}
}
func checkErr(err error, errMessage string, isQuit bool) {
	if err != nil {
		log.Error(errMessage, "error", err)
		if isQuit {
			os.Exit(1)
		}
	}
}
