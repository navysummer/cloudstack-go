package main

import (
	"./VCmonitor"
	"./config"
	"./golog"
	"./goredis"
	"./zabbix"
	"fmt"
	"os"
	"time"
)

var (
	vcApi VCmonitor.Vcenter
)

func initData()  {
	Conf, err := config.ConfInit()
	if err != nil {
		golog.Error.Println("open config file fail")
		os.Exit(500)
	}
	vcHost := Conf.Section("vcenter").Key("host").Value()
	vcPort := Conf.Section("vcenter").Key("port").Value()
	vcUsername := Conf.Section("vcenter").Key("username").Value()
	vcPassword := Conf.Section("vcenter").Key("password").Value()

	zbxHost := Conf.Section("zabbix").Key("address").Value()
	zbxPort := Conf.Section("zabbix").Key("port").Value()
	zbxUsername :=  Conf.Section("zabbix").Key("username").Value()
	zbxPassword := Conf.Section("zabbix").Key("password").Value()
	zbxVmnum:= Conf.Section("agent").Key("vmnum").MustInt64(1000)
	zbxHostgroup := Conf.Section("agent").Key("host").Value()
	zbxDiskgroup := Conf.Section("agent").Key("disk").Value()
	zbxHostTemplate := Conf.Section("agent").Key("host").Value()
	zbxDiskTemplate := Conf.Section("agent").Key("disk").Value()
	redisHost := Conf.Section("redis").Key("host").Value()
	redisPort := Conf.Section("redis").Key("port").Value()
	redisPassword := Conf.Section("redis").Key("password").Value()
	redisVmKey := Conf.Section("redis").Key("vm").Value()
	redisDiskKey := Conf.Section("redis").Key("diskhostuuid").Value()
	flag := vcApi.Connect(vcHost,vcPort,vcUsername,vcPassword)
	if flag == false {
		golog.Error.Printf("vcenter connect fail,message:%v",err.Error())
		os.Exit(500)
	}
	var zbxUrl string
	if zbxPort != "80"{
		zbxUrl = fmt.Sprintf("http://%v:%v/zabbix",zbxHost,zbxPort)
	}else {
		zbxUrl = fmt.Sprintf("http://%v/zabbix",zbxHost)
	}
	zbx := zabbix.Zabbix{}
	zbxApi, err := zbx.Login(zbxUrl,zbxUsername,zbxPassword)
	if err != nil {
		golog.Error.Printf("zabbix Login fail,message:%v",err.Error())
		os.Exit(500)
	}
	vcApi.VcZabbix(zbxApi)

	rs := &goredis.Redis{}
	err = rs.Connect(redisHost,redisPort,redisPassword)
	if err != nil {
		golog.Error.Printf("redis connect fail,message:%v",err.Error())
		os.Exit(500)
	}
	vcApi.VcRedis(rs,redisVmKey,redisDiskKey)

	hostGroup := zbx.CheckOrCreateHostGroup(zbxHostgroup)
	hostGroupid := hostGroup["groupid"]
	if hostGroupid == ""{
		golog.Error.Printf("get hostgroup(name=%v) fail",zbxHostgroup)
		os.Exit(500)
	}
	diskGroup := zbx.CheckOrCreateHostGroup(zbxDiskgroup)
	diskGroupid := diskGroup["groupid"]
	if diskGroupid == ""{
		golog.Error.Printf("get disk hostgroup(name=%v) fail",zbxDiskgroup)
		os.Exit(500)
	}
	hostTemplate := zbx.GetTemplate(zbxHostTemplate)
	hostTemplateid := hostTemplate["templateid"]
	if hostTemplateid == ""{
		golog.Error.Printf("get host template(host=%v) fail",zbxHostTemplate)
		os.Exit(500)
	}
	diskTemplate := zbx.GetTemplate(zbxDiskTemplate)
	diskTemplateid := diskTemplate["templateid"]
	if diskTemplateid == ""{
		golog.Error.Printf("get disk template(host=%v) fail",zbxDiskTemplate)
		os.Exit(500)
	}
	var data=make(map[string]interface{},0)
	data["vcHost"] = vcHost
	data["vcPort"] = vcPort
	data["vcUsername"] = vcUsername
	data["vcPassword"] = vcPassword
	agentInterface := make([]map[string]string,0)
	info := make(map[string]string,0)
	info["type"] = Conf.Section("agent").Key("type").Value()
	info["main"] = Conf.Section("agent").Key("main").Value()
	info["dns"] = Conf.Section("agent").Key("dns").Value()
	info["ip"] = Conf.Section("agent").Key("ip").Value()
	info["useip"] = Conf.Section("agent").Key("useip").Value()
	info["port"] = Conf.Section("agent").Key("port").Value()
	agentInterface = append(agentInterface,info)
	data["agentInterface"] = agentInterface
	data["hostTemplate"] = hostTemplate
	data["diskTemplate"] = diskTemplate
	data["hostGroup"] = hostGroup
	data["diskGroup"] = diskGroup
	data["zbxHost"] = zbxHost
	data["zbxVmnum"] = zbxVmnum
	data["zabbixSenderPort"] = Conf.Section("agent").Key("sender_port").MustInt(10051)
	vcApi.SetData(data)
}

func main()  {
	initData()
	for{
		vcApi.GetVms()
		time.Sleep(time.Second*30)
	}
}
