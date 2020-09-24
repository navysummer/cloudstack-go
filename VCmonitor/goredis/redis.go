package goredis

import (
	"fmt"
	"github.com/go-redis/redis"
)

type Redis struct {
	host string
	port string
	db string
	password string
}


var (
	client *redis.Client
)


func (rs Redis) Connect(host string,port string,password string)  (error){
	redis_addr := fmt.Sprintf("%s:%s",host,port)
	client = redis.NewClient(&redis.Options{
		Addr:     redis_addr,
		Password: password,
		DB:       0,
	})
	_,err := client.Ping().Result()
	return err
}

func (rs Redis) GetAllVm(key string)([]string){
	vms := client.SMembers(key).Val()
	return vms
}

func (rs Redis) CheckOrCreateVm(key string,vm string) (int64) {
	cvm := client.SAdd(key,vm).Val()
	return cvm
}

func (rs Redis) GetAllDisk(key string) (map[string]string){
	disks := client.HGetAll(key).Val()
	return disks
}

func (rs Redis) CheckOrCreateDisk(key string,field string,value string)(bool){
	cdisk := client.HSet(key,field,value).Val()
	return cdisk
}

func (rs Redis) DelVm(key string,vm string) (int64){
	dvm := client.SRem(key,vm).Val()
	return dvm
}

func (rs Redis) DelDisk(key string,field string) (int64){
	ddisk := client.HDel(key,field).Val()
	return ddisk
}

func (rs Redis) DelAllForKey(key string) (int64){
	dvm := client.Del(key).Val()
	return dvm
}
