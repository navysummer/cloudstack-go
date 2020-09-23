package goredis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis"
)

type Redis struct {
	host string
	port string
	db string
	password string
}

type RedisConnet struct {
	ctx context.Context
	client *redis.Client
}

var con *RedisConnet

func (rs Redis) Connect(host string,port string,password string)  (error){
	ctx := context.Background()
	redis_addr := fmt.Sprintf("%s:%s",host,port)
	client := redis.NewClient(&redis.Options{
		Addr:     redis_addr,
		Password: password,
		DB:       0,
	})
	con = &RedisConnet{
		ctx,client,
	}
	_,err := client.Ping(ctx).Result()
	return err
}

func (rs Redis) GetAllVm(key string)([]string){
	vms := con.client.SMembers(con.ctx,key).Val()
	return vms
}

func (rs Redis) CheckOrCreateVm(key string,vm string) (int64) {
	cvm := con.client.SAdd(con.ctx,key,vm).Val()
	return cvm
}

func (rs Redis) GetAllDisk(key string) (map[string]string){
	disks := con.client.HGetAll(con.ctx,key).Val()
	return disks
}

func (rs Redis) CheckOrCreateDisk(key string,field string,value string)(int64){
	cdisk := con.client.HSet(con.ctx,key,field,value).Val()
	return cdisk
}

func (rs Redis) DelVm(key string,vm string) (int64){
	dvm := con.client.SRem(con.ctx,key,vm).Val()
	return dvm
}

func (rs Redis) DelDisk(key string,field string) (int64){
	ddisk := con.client.HDel(con.ctx,key,field).Val()
	return ddisk
}

func (rs Redis) DelAllForKey(key string) (int64){
	dvm := con.client.Del(con.ctx,key).Val()
	return dvm
}
