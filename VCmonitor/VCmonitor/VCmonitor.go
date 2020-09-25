package VCmonitor

import (
	"VCmonitor/golog"
	"VCmonitor/goredis"
	"github.com/navysummer/gozabbix"
	"VCmonitor/zabbix"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Vcenter struct {
	host     string
	port     string
	username string
	password string
}

type VConnect struct {
	connect *govmomi.Client
	context context.Context
}

var (
	counterInfo = make(map[string]int32, 0)
	metrics     performance.MetricList
	perfManager *performance.Manager
	connect     *VConnect
	api         *gozabbix.API
	rs          *goredis.Redis
	zbx         *zabbix.Zabbix
	vmKey       = ""
	diskKey     = ""
	wg          sync.WaitGroup
	data        map[string]interface{}
	port        int
)

func (vc *Vcenter) SetData(dt map[string]interface{}) {
	data = dt
	port = dt["zabbixSenderPort"].(int)
}

func (vc *Vcenter) Connect(host string, port string, username string, password string) bool {
	vc.host = host
	vc.port = port
	vc.username = username
	vc.password = password
	ctx, _ := context.WithCancel(context.Background())
	var vcurl string
	if vc.port != "443" {
		vcurl = "https://" + vc.host + ":" + vc.port + "/sdk"
	} else {
		vcurl = "https://" + vc.host + "/sdk"
	}
	u, err := soap.ParseURL(vcurl)
	if err != nil {
		golog.Error.Printf("soap URL parse fail,message:%v", err.Error())
		os.Exit(500)
	}
	u.User = url.UserPassword(vc.username, vc.password)
	client, err := govmomi.NewClient(ctx, u, true)
	connect = &VConnect{
		client, ctx,
	}
	flag := false
	if err == nil {
		flag = true
	} else {
		golog.Error.Printf("vcenter login fail,message:%v", err.Error())
	}
	return flag
}

func (vc *Vcenter) VcZabbix(zapi *zabbix.Zabbix) {
	zapi = zapi
}

func (vc *Vcenter) VcRedis(rsapi *goredis.Redis, vm_key string, disk_key string) {
	vmKey = vm_key
	diskKey = disk_key
	rs = rsapi
}

func (vc *Vcenter) GetVms() {
	c := connect.connect.Client
	ctx := connect.context
	kind := []string{"VirtualMachine"}
	m := view.NewManager(c)
	v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, kind, true)
	if err != nil {
		golog.Error.Printf("create container view fail,message:%v", err.Error())
	}
	perfManager = performance.NewManager(c)
	counters, err := perfManager.CounterInfoByName(ctx)
	if err != nil {
		golog.Error.Printf("get perfManager CounterInfo fail,message:%v", err.Error())
	}
	for key, info := range counters {
		if key == "cpu.usage.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = ""
			metrics = append(metrics, perfMetricId)
		} else if key == "mem.usage.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = ""
			metrics = append(metrics, perfMetricId)
		} else if key == "net.received.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = "*"
			metrics = append(metrics, perfMetricId)
		} else if key == "net.transmitted.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = "*"
			metrics = append(metrics, perfMetricId)
		} else if key == "net.packetsTx.summation" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = ""
			metrics = append(metrics, perfMetricId)
		} else if key == "net.packetsRx.summation" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = ""
			metrics = append(metrics, perfMetricId)
		} else if key == "virtualDisk.write.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = "*"
			metrics = append(metrics, perfMetricId)
		} else if key == "virtualDisk.read.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = "*"
			metrics = append(metrics, perfMetricId)
		} else if key == "virtualDisk.numberReadAveraged.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = "*"
			metrics = append(metrics, perfMetricId)
		} else if key == "virtualDisk.numberWriteAveraged.average" {
			counterInfo[key] = info.Key
			perfMetricId := types.PerfMetricId{}
			perfMetricId.CounterId = info.Key
			perfMetricId.Instance = "*"
			metrics = append(metrics, perfMetricId)
		}
	}
	var vms []mo.VirtualMachine
	err = v.Retrieve(ctx, kind, []string{"name", "runtime", "summary", "config", "network"}, &vms)
	if err != nil {
		golog.Error.Printf("RetrieveWithFilter fail,message:%v", err.Error())
	}
	vc.SyncGet(vms)
	//fmt.Println(vmnum)
	//for _, vm := range vms {
	//	vc.GetVm(vm)
	//}
}

func (vc *Vcenter) SyncGet(vms []mo.VirtualMachine) {
	vmnum := data["zbxVmnum"].(int64)
	num := len(vms)
	i := 0
	var cvm []mo.VirtualMachine
	var xvms []mo.VirtualMachine
	var gvms [][]mo.VirtualMachine
	for i < num {
		if int64(len(xvms)) < vmnum {
			xvms = append(xvms, vms[i])
		} else {
			gvms = append(gvms, xvms)
			xvms = cvm
			xvms = append(xvms, vms[i])
		}
		if i == num-1 && len(xvms) != 0 {
			gvms = append(gvms, xvms)
		}
		i++
	}
	var glen = len(gvms)
	wg.Add(glen)
	j := 0
	for j < glen {
		go vc.GetVmsData(gvms[j])
		j++
	}
	wg.Wait()
}

func (vc *Vcenter) GetVmsData(vms []mo.VirtualMachine) {
	for _, vm := range vms {
		vc.GetVm(vm)
	}
	wg.Done()
}

func (vc *Vcenter) GetVm(vm mo.VirtualMachine) {
	PowerState := string(vm.Runtime.PowerState)
	if PowerState == "poweredOn" {
		disk_uncommitted := float64(vm.Summary.Storage.Uncommitted)
		disk_committed := float64(vm.Summary.Storage.Committed)
		disk_capacity := disk_uncommitted + disk_committed
		var disk_util_inband float64
		if disk_capacity == 0 {
			disk_util_inband = 0.0
		}
		disk_util_inband = disk_committed / disk_capacity
		disk_util_inband_str := strconv.FormatFloat(disk_util_inband, 'f', 2, 64)
		vm_name := vm.Name
		rs.CheckOrCreateVm(vmKey, vm_name)
		zbx.CheckOrCreateHost(vm_name, vm_name,
			data["hostGroup"].(map[string]string)["groupid"],
			data["agentInterface"].([]map[string]string),
			data["hostTemplate"].(map[string]string)["templateid"])
		var (
			device_dict           = make(map[string]string, 0)
			disk_backing_filename = make(map[string]string, 0)
			disk_filename_uuid    = make(map[string]string, 0)
		)
		//cpu_util := strconv.FormatFloat(float64(vm.Summary.QuickStats.OverallCpuUsage)/100,'f',2,64)
		//mem_util := strconv.FormatFloat(float64(vm.Summary.QuickStats.HostMemoryUsage)/100,'f',2,64)
		hardwaredevice := vm.Config.Hardware.Device
		for _, device := range hardwaredevice {
			typeStr := reflect.TypeOf(device).String()
			if strings.Contains(typeStr, "IDE") {
				deviceinfob, _ := json.Marshal(device)
				deviceinfos := string(deviceinfob)
				device_label := gjson.Get(deviceinfos, "DeviceInfo.Label").String()
				lower_device_label := strings.ToLower(device_label)
				device_label_splice := strings.Split(lower_device_label, " ")
				controllername := device_label_splice[0] + device_label_splice[1] + ":"
				device_list := gjson.Get(deviceinfos, "Device").Array()
				if len(device_list) != 0 {
					for idx, value := range device_list {
						k := value.String()
						v := fmt.Sprintf("%s%d", controllername, idx)
						device_dict[k] = v
					}
				}
			}
			if strings.Contains(typeStr, "LsiLogic") {
				deviceinfob, _ := json.Marshal(device)
				deviceinfos := string(deviceinfob)
				device_label := gjson.Get(deviceinfos, "DeviceInfo.Label").String()
				lower_device_label := strings.ToLower(device_label)
				var device_label_splice []string
				if strings.Contains(lower_device_label, " controller ") {
					device_label_splice = strings.Split(lower_device_label, " controller ")
				} else {
					device_label_splice = strings.Split(lower_device_label, " \\xe6\\x8e\\xa7\\xe5\\x88\\xb6\\xe5\\x99\\xa8 ")
				}
				controllername := device_label_splice[0] + device_label_splice[1] + ":"
				device_list := gjson.Get(deviceinfos, "Device").Array()
				if len(device_list) != 0 {
					for idx, value := range device_list {
						k := value.String()
						v := fmt.Sprintf("%s%d", controllername, idx)
						device_dict[k] = v
					}
				}
			}
			if strings.Contains(typeStr, "VirtualDisk") {
				deviceinfob, _ := json.Marshal(device)
				deviceinfos := string(deviceinfob)
				devicefiname := gjson.Get(deviceinfos, "Backing.FileName").String()
				deviceuuid := gjson.Get(deviceinfos, "Backing.Uuid").String()
				devicekey := gjson.Get(deviceinfos, "Key").String()
				devicefiname_arr := strings.Split(devicefiname, " ")
				devicefiname_str := devicefiname_arr[0]
				path_str := devicefiname_arr[1]
				path_arr := strings.Split(path_str, ".vmdk")
				path := path_arr[0]
				devicefiname_str_len := len(devicefiname_str)
				storageid := strings.Replace(devicefiname_str[1:devicefiname_str_len-1], "-", "", -1)
				if strings.Contains(devicefiname, "/") {
					path_arr1 := strings.Split(path, "/")
					path = path_arr1[1]
				}
				devicelastname := storageid + "_" + vm_name + "_" + path
				disk_backing_filename[devicekey] = devicelastname
				disk_filename_uuid[devicelastname] = vm_name + "_" + deviceuuid
				rs.CheckOrCreateDisk(diskKey, disk_filename_uuid[devicelastname], devicelastname)
			}
		}
		device_con_dict := make(map[string]string, 0)
		for name, host := range disk_filename_uuid {
			rs.CheckOrCreateDisk(diskKey, name, host)
			zbx.CheckOrCreateHost(host, name,
				data["diskGroup"].(map[string]string)["groupid"],
				data["agentInterface"].([]map[string]string),
				data["diskTemplate"].(map[string]string)["templateid"])
		}
		for devicekey, devicelastname := range disk_backing_filename {
			for key, value := range device_dict {
				if devicekey == key {
					device_con_dict[disk_filename_uuid[devicelastname]] = value
				}
			}
		}

		perfs, disk_name_perfs := vc.GetItemData(vm, device_con_dict)
		perfs["disk_util_inband"] = disk_util_inband_str
		zbx.SenderItemData(vm_name, perfs, data["zbxHost"].(string), port)
		for disk_name, disk_perfs := range disk_name_perfs {
			zbx.SenderItemData(disk_name, disk_perfs, data["zbxHost"].(string), port)
		}
	}
}

func (vc *Vcenter) GetVcData(vm mo.VirtualMachine) ([]types.BasePerfEntityMetricBase, error) {
	var perf_result []types.BasePerfEntityMetricBase
	var err error
	currentTime, err := methods.GetCurrentTime(connect.context, connect.connect)
	if err != nil {
		golog.Error.Printf("Get Current Time Fail,message:%v", err.Error())
	}
	if currentTime != nil {
		p1t, _ := time.ParseDuration("-1m")
		n1t, _ := time.ParseDuration("1m")
		startTime := currentTime.Add(p1t)
		endTime := currentTime.Add(n1t)
		spec := types.PerfQuerySpec{
			Entity:    vm.Self,
			MetricId:  metrics,
			StartTime: &startTime,
			EndTime:   &endTime,
		}
		pqss := []types.PerfQuerySpec{spec}
		perf_result, err = perfManager.Query(connect.context, pqss)
		if err != nil {
			golog.Error.Printf("Query Item Data Fail,message:%v", err.Error())
		}
	}
	return perf_result, err
}

func (vc *Vcenter) GetItemData(vm mo.VirtualMachine, device_con_dict map[string]string) (map[string]string, map[string]map[string]string) {
	var (
		network_incoming_bytes_rate_inband float64 = 0
		network_outing_bytes_rate_inband   float64 = 0
		disk_write_bytes_rate              float64 = 0
		disk_read_bytes_rate               float64 = 0
		disk_write_requests_rate           float64 = 0
		disk_read_requests_rate            float64 = 0
		cpu_util                           float64 = 0
		mem_util                           float64 = 0
		perfs                                      = make(map[string]string, 0)
		disk_name_perfs                            = make(map[string]map[string]string, 0)
		tdisk_name_perfs                           = make(map[string]map[string]float64, 0)
	)
	perf_result, err := vc.GetVcData(vm)
	if err == nil {
		for k, _ := range device_con_dict {
			disk_item := map[string]float64{"disk_write_bytes_rate": 0,
				"disk_read_bytes_rate":     0,
				"disk_read_requests_rate":  0,
				"disk_write_requests_rate": 0}
			tdisk_name_perfs[k] = disk_item
		}
		if len(perf_result) > 0 {
			perf_resultb, err := json.Marshal(perf_result[0])
			perf_resultstr := string(perf_resultb)
			if err != nil {
				golog.Error.Printf("get item data fail,message:%v", err.Error())
			} else {
				perf_values := gjson.Get(perf_resultstr, "Value").Array()
				for _, perf_value := range perf_values {
					counter_id := int32(perf_value.Get("Id.CounterId").Int())
					instance := perf_value.Get("Id.Instance").Str
					value := perf_value.Get("Value").Array()[0].Num
					if counter_id == counterInfo["net.received.average"] && instance == "" {
						network_incoming_bytes_rate_inband += value
					}
					if counter_id == counterInfo["net.transmitted.average"] && instance == "" {
						network_outing_bytes_rate_inband += value
					}
					if counter_id == counterInfo["cpu.usage.average"] && instance == "" {
						cpu_util += value / 100.0
					}
					if counter_id == counterInfo["mem.usage.average"] && instance == "" {
						mem_util += value / 100.0
					}
					if counter_id == counterInfo["virtualDisk.write.average"] && instance == "" {
						disk_write_bytes_rate += value
					}

					if counter_id == counterInfo["virtualDisk.read.average"] && instance == "" {
						disk_read_bytes_rate += value
					}
					if counter_id == counterInfo["virtualDisk.numberReadAveraged.average"] {
						disk_read_requests_rate += value
					}

					if counter_id == counterInfo["virtualDisk.numberWriteAveraged.average"] {
						disk_write_requests_rate += value
					}

					if counter_id == counterInfo["virtualDisk.read.average"] {
						for k, v := range device_con_dict {
							if instance == v {
								tdisk_name_perfs[k]["disk_read_bytes_rate"] += value
							}

						}
					}
					if counter_id == counterInfo["virtualDisk.write.average"] {
						for k, v := range device_con_dict {
							if instance == v {
								tdisk_name_perfs[k]["disk_write_bytes_rate"] += value

							}

						}
					}
					if counter_id == counterInfo["virtualDisk.numberReadAveraged.average"] {
						for k, v := range device_con_dict {
							if instance == v {
								tdisk_name_perfs[k]["disk_read_requests_rate"] += value
							}

						}
					}

					if counter_id == counterInfo["virtualDisk.numberWriteAveraged.average"] {
						for k, v := range device_con_dict {
							if instance == v {
								tdisk_name_perfs[k]["disk_write_requests_rate"] += value
							}

						}
					}
				}
				for name, disk_perfs := range tdisk_name_perfs {
					item := make(map[string]string, 0)
					for key, value := range disk_perfs {
						item[key] = strconv.FormatFloat(value, 'f', 2, 64)
					}
					disk_name_perfs[name] = item
				}
				perfs["network_incoming_bytes_rate_inband"] = strconv.FormatFloat(network_incoming_bytes_rate_inband*8, 'f', 2, 64)
				perfs["network_outing_bytes_rate_inband"] = strconv.FormatFloat(network_outing_bytes_rate_inband*8, 'f', 2, 64)
				perfs["disk_write_bytes_rate"] = strconv.FormatFloat(disk_write_bytes_rate, 'f', 2, 64)
				perfs["disk_read_bytes_rate"] = strconv.FormatFloat(disk_read_bytes_rate, 'f', 2, 64)
				perfs["cpu_util"] = strconv.FormatFloat(cpu_util, 'f', 2, 64)
				perfs["mem_util"] = strconv.FormatFloat(mem_util, 'f', 2, 64)
				perfs["disk_read_requests_rate"] = strconv.FormatFloat(disk_read_requests_rate, 'f', 2, 64)
				perfs["disk_write_requests_rate"] = strconv.FormatFloat(disk_write_requests_rate, 'f', 2, 64)

			}
		}
	}
	return perfs, disk_name_perfs
}
