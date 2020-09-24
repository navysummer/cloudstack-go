package zabbix

import (
	"VCmonitor/golog"
	"VCmonitor/gozabbix"
	"time"
)

type Zabbix struct {
	url string
	username string
	password string
	
}

var (
	auth *gozabbix.API
)

func (zbx *Zabbix) Login(url string,username string,password string) (*Zabbix,error) {
	zbx.url = url
	zbx.username = username
	zbx.password = password
	api := &Zabbix{}
	napi, err := gozabbix.NewAPI(zbx.url+"/api_jsonrpc.php",zbx.username,zbx.password)
	auth = napi
	if err!=nil{
		return api,err
	}
	_, err = napi.Login()
	return api,err
}

func (zbx *Zabbix) GetHostGroup(name string) (hg map[string]string) {
	hostgroup := map[string]string{"name":name, "groupid":""}
	params := make(map[string]interface{}, 0)
	filter := make(map[string]string, 0)
	filter["name"] = name
	params["output"] = "extend"
	params["filter"] = filter
	method := "hostgroup.get"
	response, err := auth.ZabbixRequest(method,params)
	if err != nil {
		golog.Error.Printf("get zabbix hostgroup(name=%v) fail,message:%v", name,err.Error())
	}else {
		error := response.Error
		if error.Message != ""{
			golog.Error.Printf("get zabbix hostgroup(name=%v) fail,message:%v", name,error.Error())
		}else{
			result := response.Result.([] interface{})
			if len(result) != 0{
				hginfo := result[0].(map[string] interface{})
				groupid := hginfo["groupid"].(string)
				hostgroup["groupid"] = groupid
			}
		}
	}
	return hostgroup
}

func (zbx *Zabbix) CheckOrCreateHostGroup(name string) (hg map[string]string){
	hostgroup := map[string]string{"name":name, "groupid":""}
	ghostgroup := zbx.GetHostGroup(name)
	if ghostgroup["groupid"] == ""{
		cparams := map[string]string{"name":name}
		cmethod := "hostgroup.create"
		response, err := auth.ZabbixRequest(cmethod,cparams)
		if err != nil {
			golog.Error.Printf("create hostgroup(name=%v) fail,message:%v",name,err.Error())
		}else{
			error := response.Error
			if error.Message != ""{
				golog.Error.Printf("get zabbix hostgroup(name=%v) fail,message:%v", name,error.Error())
			}else{
				result := response.Result.(map[string]interface{})
				groupids := result["groupids"].([] interface{})
				if len(groupids) !=0{
					hostgroup["groupid"] = groupids[0].(string)
				}
			}

		}

	}else {
		hostgroup["groupid"] = ghostgroup["groupid"]
	}
	return hostgroup
}

func (zbx *Zabbix) GetHost(host string,name string,groupid string)(gozabbix.JsonRPCResponse, error){
	params := make(map[string]interface{},0)
	params["output"] = []string{"hostid", "host", "name"}
	params["selectGroups"] = "extend"
	params["selectParentTemplates"] = []string{"templateid", "name"}
	filter := make(map[string]interface{},0)
	if groupid != ""{
		params["groupids"] = groupid
	}
	if host != ""{
		filter["host"] = host
	}
	if name != ""{
		filter["name"] = name
	}
	if len(filter) != 0{
		params["filter"] = filter
	}
	method := "host.get"
	response, err := auth.ZabbixRequest(method,params)
	if err != nil {
		golog.Error.Printf("get host(parmas=%v) fail,message:%v",params,err.Error())
	}
	error := response.Error
	if error.Message != ""{
		golog.Error.Printf("get host(parmas=%v) fail,message:%v", params,error.Error())
	}
	return response, err
}

func (zbx *Zabbix) CheckOrCreateHost(host string,name string,groupid string,agent_interface []map[string]string,templateid string) (map[string]string){
	hostinfo := map[string]string{"name":name, "groupid":groupid,"hostid":"","host":host,"templateid":templateid}
	ckhost,err:= zbx.GetHost(host,name,"")
	if err != nil {
		golog.Error.Printf("get host(name=%v,host=%v),message:%v",name,host,err.Error())
	}else {
		error := ckhost.Error
		if error.Message != ""{
			golog.Error.Printf("get host(name=%v,host=%v),message:%v",name,host,err.Error())
		}else {
			result := ckhost.Result.([] interface{})
			if len(result) == 0{
				cparams := make(map[string]interface{},0)
				cmethod := "host.create"
				cparams["host"] = host
				cparams["interfaces"] = agent_interface
				if name != ""{
					cparams["name"] = name
				}
				cparams["groups"] = []map[string]string{{"groupid":groupid}}
				if templateid != ""{
					cparams["templates"]= []map[string]string{{"templateid":templateid}}
				}
				chost,err:= auth.ZabbixRequest(cmethod,cparams)
				if err != nil {
					golog.Error.Printf("create host(%v) fail,message:%v",cparams,err.Error())
				}else {
					error := chost.Error
					if error.Message != ""{
						golog.Error.Printf("get host(parmas=%v) fail,message:%v", cparams,error.Error())
					}else {
						cresult := chost.Result.(map[string]interface{})
						hostids := cresult["hostids"].([] interface{})
						if len(hostids) != 0{
							hostinfo["hostid"] = hostids[0].(string)
						}
					}
				}
			}else {
				hinfo := result[0].(map[string] interface{})
				hostinfo["hostid"] = hinfo["hostid"].(string)
			}
		}
	}
	return hostinfo
}

func (zbx *Zabbix) GetTemplate(host string) (map[string]string){
	templateinfo := map[string]string{"host":host,"name":"","templateid":""}
	params := make(map[string]interface{},0)
	params["output"] = []string{"templateid","host","name"}
	params["filter"] = map[string]string{"host":host}
	method := "template.get"
	response, err := auth.ZabbixRequest(method,params)
	if err != nil {
		golog.Error.Printf("get template(host=%v),message:%v",host,err.Error())
	}
	error := response.Error
	if error.Message != ""{
		golog.Error.Printf("get template(host=%v),message:%v", host,error.Error())
	}else {
		templates := response.Result.([] interface{})
		if len(templates) != 0{
			template := templates[0].(map[string]interface{})
			templateinfo["templateid"] = template["templateid"].(string)
			templateinfo["name"] = template["name"].(string)
		}
	}
	return templateinfo
}

func (zbx *Zabbix) LinkTemplate(hostid string,templateid string)  {
	params := make(map[string]interface{},0)
	params["templates"] = map[string]string{"templateid":templateid}
	params["hosts"]  = map[string]string{"hostid":hostid}
	method := "template.massadd"
	res, err := auth.ZabbixRequest(method,params)
	if err != nil {
		golog.Error.Printf("link host(hostid=%v) to template(templateid=%v),message:%v",hostid,templateid,err.Error())
	}
	error := res.Error
	if error.Message != ""{
		golog.Error.Printf("link host(hostid=%v) to template(templateid=%v),message:%v",hostid,templateid,error.Error())
	}
}

func (zbx *Zabbix) ChangeHostgroup(hosts []map[string]string, groups []map[string]string)  {
	params := make(map[string]interface{},0)
	params["groups"] = groups
	params["hosts"] = hosts
	method := "hostgroup.massadd"
	res,err := auth.ZabbixRequest(method,params)
	if err != nil {
		golog.Error.Printf("change host(hosts=%v) to hostgroup(groups=%v),message:%v",hosts,groups,err.Error())
	}
	error := res.Error
	if error.Message != ""{
		golog.Error.Printf("change host(hosts=%v) to hostgroup(groups=%v),message:%v",hosts,groups,error.Error())
	}
}

func (zbx *Zabbix)SenderItemData(host string,perf map[string]string,serverip string,port int)  {
	//fmt.Println(host,perf)
	var metrics []*gozabbix.Metric
	ctime := time.Now().Unix()
	for key,value := range perf{
		metrics = append(metrics,gozabbix.NewMetric(host,key,value, ctime))
	}
	packet := gozabbix.NewPacket(metrics)
	z := gozabbix.NewSender(serverip, port)
	data,err := z.Send(packet)
	if err != nil {
		golog.Error.Printf("send host(host=%v) item data(perf=%v),message:%v",host,perf,err.Error())
	}
	if len(data) == 0{
		golog.Error.Printf("send host(host=%v) item data(perf=%v)",host,perf)
	}
}





