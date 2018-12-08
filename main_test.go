package main

import (
	"fmt"
	"github.com/godaddy-x/jorm/cache/redis"
	"github.com/godaddy-x/jorm/sqlc"
	"github.com/godaddy-x/jorm/sqld"
	"github.com/godaddy-x/jorm/util"
	"testing"
)

type User struct {
	Id       int64  `json:"id" bson:"_id" tb:"rbac_user"`
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
	Uid      string `json:"uid" bson:"uid"`
	Utype    int8   `json:"utype" bson:"utype"`
}

type MGUser struct {
	Id    int64  `json:"id" bson:"_id" tb:"rbac_user" mg:"true" ignore:"true"`
	Name  string `json:"name" bson:"name"`
	Ctime int64  `json:"ctime" bson:"ctime" date:"true"`
}

type OwWallet struct {
	Id           int64  `json:"id" bson:"_id" tb:"ow_wallet" mg:"true"`
	AppID        string `json:"appID" bson:"appID"`
	WalletID     string `json:"walletID" bson:"walletID"`
	Alias        string `json:"alias" bson:"alias"`
	IsTrust      int64  `json:"isTrust" bson:"isTrust"`
	PasswordType int64  `json:"passwordType" bson:"passwordType"`
	Password     string `json:"password" bson:"password"`
	AuthKey      string `json:"authKey" bson:"authKey"`
	RootPath     string `json:"rootPath" bson:"rootPath"`
	AccountIndex int64  `json:"accountIndex" bson:"accountIndex"`
	Keystore     string `json:"keystore" bson:"keystore"`
	Applytime    int64  `json:"applytime" bson:"applytime"`
	Succtime     int64  `json:"succtime" bson:"succtime"`
	Dealstate    int64  `json:"dealstate" bson:"dealstate"`
	Ctime        int64  `json:"ctime" bson:"ctime"`
	Utime        int64  `json:"utime" bson:"utime"`
	State        int64  `json:"state" bson:"state"`
}

func init() {
	redis := cache.RedisConfig{}
	if err := util.ReadLocalJsonConfig("resource/redis.json", &redis); err != nil {
		panic(util.AddStr("读取redis配置失败: ", err.Error()))
	}
	manager, err := new(cache.RedisManager).InitConfig(redis)
	if err != nil {
		panic(err.Error())
	}
	manager, err = manager.Client()
	if err != nil {
		panic(err.Error())
	}

	mysql := sqld.MysqlConfig{}
	if err := util.ReadLocalJsonConfig("resource/mysql.json", &mysql); err != nil {
		panic(util.AddStr("读取mysql配置失败: ", err.Error()))
	}
	new(sqld.MysqlManager).InitConfigAndCache(manager, mysql)

	mongo := sqld.MGOConfig{}
	if err := util.ReadLocalJsonConfig("resource/mongo.json", &mongo); err != nil {
		panic(util.AddStr("读取mongo配置失败: ", err.Error()))
	}
	new(sqld.MGOManager).InitConfigAndCache(manager, mongo)
}

func TestMysql(t *testing.T) {
	db, err := new(sqld.MysqlManager).Get(sqld.Option{AutoTx: true, DsName: "TEST", CacheSync: true})
	if err != nil {
		fmt.Print(err.Error())
	} else {
		defer db.Close()
		//wallet := MGUser{Name: "test", Ctime: util.Time()}
		//if err := db.Save(&wallet); err != nil {
		//	fmt.Println(err.Error())
		//}
		//fmt.Println(wallet)
		find := []*MGUser{}
		if err := db.FindList(sqlc.M(MGUser{}).Eq("name", "test"), &find); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(len(find))
	}
}

func TestMongo(t *testing.T) {
	db, err := new(sqld.MGOManager).Get(sqld.Option{DsName: "TEST"})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		wallet := MGUser{}
		if err := db.FindOne(sqlc.M(MGUser{}).Eq("_id", 1068800540239986688).Cache(sqlc.CacheConfig{Key: "mytest", Expire: 30}), &wallet); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(wallet)
	}
}

func TestRedis(t *testing.T) {
	client, err := new(cache.RedisManager).Client()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		//client.Put("redislock:test", "1", 60)
		//if err := client.TryLock("test", func() error {
		//	fmt.Println(i)
		//	return nil
		//}); err != nil {
		//	fmt.Println(err.Error())
		//}
		//s := ""
		//client.Del("test")
		fmt.Println(client.Size("tx.block.coin.BTC"))
	}
}
