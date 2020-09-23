package golog

import (
	"gopkg.in/ini.v1"
	"io"
	"log"
	"os"
)

var (
	Info *log.Logger
	Warning *log.Logger
	Error * log.Logger
)
func init(){

	cfg, err := ini.Load("./config/VCmonitor.ini")
	log_file := cfg.Section("loggers").Key("file").Value()
	log_level := cfg.Section("loggers").Key("level").Value()
	errFile,err:=os.OpenFile(log_file,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0666)
	if err!=nil{
		log.Fatalln("打开日志文件失败：",err)
		os.Exit(500)
	}
	log_levels := [...] string{"info","warning","error"}
	idx := -1
	for index,value:=range log_levels{
		if value == log_level{
			idx = index
		}
	}
	if idx == -1 {
		idx = 0
	}

	Info = log.New(os.Stdout,"Info:",log.Ldate | log.Ltime | log.Lshortfile)
	Warning = log.New(os.Stdout,"Warning:",log.Ldate | log.Ltime | log.Lshortfile)
	Error = log.New(os.Stdout,"Error:",log.Ldate | log.Ltime | log.Lshortfile)
	if idx == 0{
		Info = log.New(io.MultiWriter(os.Stderr,errFile),"Info:",log.Ldate | log.Ltime | log.Lshortfile)
		Warning = log.New(io.MultiWriter(os.Stderr,errFile),"Warning:",log.Ldate | log.Ltime | log.Lshortfile)
		Error = log.New(io.MultiWriter(os.Stderr,errFile),"Error:",log.Ldate | log.Ltime | log.Lshortfile)
	}else if idx == 1{
		Warning = log.New(io.MultiWriter(os.Stderr,errFile),"Warning:",log.Ldate | log.Ltime | log.Lshortfile)
		Error = log.New(io.MultiWriter(os.Stderr,errFile),"Error:",log.Ldate | log.Ltime | log.Lshortfile)
	}else{
		Error = log.New(io.MultiWriter(os.Stderr,errFile),"Error:",log.Ldate | log.Ltime | log.Lshortfile)
	}
}
