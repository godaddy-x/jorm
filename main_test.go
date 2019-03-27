package main

import (
	"errors"
	"fmt"
	"github.com/godaddy-x/jorm/amqp"
	c2 "github.com/godaddy-x/jorm/cache/mc"
	"github.com/godaddy-x/jorm/cache/redis"
	"github.com/godaddy-x/jorm/exception"
	"github.com/godaddy-x/jorm/gauth"
	"github.com/godaddy-x/jorm/jwt"
	"github.com/godaddy-x/jorm/sqlc"
	"github.com/godaddy-x/jorm/sqld"
	"github.com/godaddy-x/jorm/util"
	"testing"
	"time"
)

type User struct {
	Id       int64  `json:"id" bson:"_id" tb:"rbac_user"`
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
	Uid      string `json:"uid" bson:"uid"`
	Utype    int8   `json:"utype" bson:"utype"`
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

func TestMysql1(t *testing.T) {
	mysql, err := new(sqld.MysqlManager).Get(sqld.Option{AutoTx: true, DsName: "TEST", CacheSync: true})
	if err != nil {
		panic(err)
	}
	defer mysql.Close()
	tx := mysql.Tx
	tx.Exec("update set xx = xx where xx= xx", nil)

}

func TestMysql(t *testing.T) {
	db, err := new(sqld.MysqlManager).Get(sqld.Option{AutoTx: true, DsName: "TEST", CacheSync: true})
	if err != nil {
		fmt.Print(err.Error())
	} else {
		defer db.Close()
		wallet := OwWallet{WalletID: util.GetUUID(), Ctime: util.Time()}
		wallet1 := OwWallet{WalletID: util.GetUUID(), Ctime: util.Time()}
		wallet2 := OwWallet{WalletID: util.GetUUID(), Ctime: util.Time()}
		if err := db.Save(&wallet, &wallet1, &wallet2); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(wallet)
		find := []*OwWallet{}
		if err := db.FindList(sqlc.M(OwWallet{}).Eq("walletID", "test").Limit(1, 10), &find); err != nil {
			panic(err)
		}
		fmt.Println(len(find))
		if c, err := db.Count(sqlc.M(OwWallet{})); err != nil {
			panic(err)
		} else {
			fmt.Println("count: ", c)
		}
	}
}

func TestMongo(t *testing.T) {
	//db, err := new(sqld.MGOManager).Get(sqld.Option{DsName: "TEST"})
	//if err != nil {
	//	fmt.Println(err.Error())
	//} else {
	//	wallet := MGUser{}
	//	if err := db.FindOne(sqlc.M(MGUser{}).Eq("_id", 1068800540239986688).Cache(sqlc.CacheConfig{Key: "mytest", Expire: 30}), &wallet); err != nil {
	//		fmt.Println(err.Error())
	//	}
	//	fmt.Println(wallet)
	//}
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

func TestJWT(t *testing.T) {
	subject := &jwt.Subject{
		Payload: &jwt.Payload{Sub: "admin", Iss: jwt.JWT, Exp: jwt.HALF_HOUR, Rxp: jwt.TWO_WEEK, Nbf: 0},
	}
	secret := "123456789"
	result, _ := subject.Generate(secret)
	fmt.Println(result.AccessToken)
	fmt.Println(result.RefreshToken)
	fmt.Println(result.AccessTime)
	if err := subject.Valid(result.AccessToken, secret, false); err != nil {
		fmt.Println(err)
	}
	result1, err := subject.Refresh(result.AccessToken, result.RefreshToken, secret, result.AccessTime)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(result1.AccessToken)
		fmt.Println(result1.RefreshToken)
		fmt.Println(result1.AccessTime)
		if err := subject.Valid(result1.AccessToken, secret, false); err != nil {
			fmt.Println(err)
		}
	}
}

func TestEX(t *testing.T) {
	e := ex.Try{ex.UNKNOWN, "sss", errors.New("ss"), nil}
	s := ex.Catch(e)
	fmt.Println(s)
}

func TestGA(t *testing.T) {
	// 生成种子
	seed := gauth.GenerateSeed()
	fmt.Println("种子: ", seed)
	// 通过种子生成密钥
	key, _ := gauth.GenerateSecretKey(seed)
	fmt.Println("密钥: ", key)
	// 通过密钥+时间生成验证码
	rs := gauth.GetNewCode(key, time.Now().Unix())
	fmt.Println("验证码: ", rs)
	fmt.Println("开始睡眠延迟中,请耐心等待...")
	time.Sleep(5 * time.Second)
	// 校验已有验证码
	fmt.Println("校验结果: ", gauth.ValidCode(key, rs))
}

func TestMC(t *testing.T) {
	mconfig := c2.MemcacheConfig{
		Host:        "192.168.27.160",
		Port:        11211,
		MaxIdle:     500,
		IdleTimeout: 10,
	}
	manager, err := new(c2.MemcacheManager).InitConfig(mconfig)
	if err != nil {
		panic(err.Error())
	}
	manager, err = manager.Client()
	if err != nil {
		panic(err.Error())
	}
	for i := 0; i < 10000; i++ {
		fmt.Println(i, "------")
		s := ""
		manager.Put("mytest", "test", 5)
		fmt.Println(manager.Get("mytest", &s))
		fmt.Println(s)
		manager.Del("mytest")
		fmt.Println(manager.Get("mytest", &s))
		fmt.Println(s)
	}
}

func TestMQPull(t *testing.T) {
	exchange := "my.test.exchange101"
	queue := "my.test.queue101"
	input := rabbitmq.AmqpConfig{
		Username: "admin",
		Password: "admin",
		Host:     "192.168.27.160",
		Port:     5672,
	}
	new(rabbitmq.PublishManager).InitConfig(input)
	new(rabbitmq.PullManager).InitConfig(input)
	mq, _ := new(rabbitmq.PullManager).Client()
	mq.StartPullServer(
		&rabbitmq.PullReceiver{
			Exchange: exchange,
			Queue:    queue,
			LisData:  rabbitmq.LisData{},
			Callback: func(msg rabbitmq.MsgData) (rabbitmq.MsgData, error) {
				return rabbitmq.MsgData{}, nil
			},
		},
	)
	//mq.AddPullReceiver(&rabbitmq.PullReceiver{
	//	Exchange: "my.test.exchange1010",
	//	Queue:    "my.test.queue1010",
	//})
}

func TestMQPublish(t *testing.T) {
	input := rabbitmq.AmqpConfig{
		Username: "admin",
		Password: "admin",
		Host:     "192.168.27.160",
		Port:     5672,
	}
	mq := new(rabbitmq.PublishManager).InitConfig(input)
	for ; ; {
		cli, _ := mq.Client()
		v := map[string]interface{}{"test": 1}
		cli.Publish(rabbitmq.MsgData{
			Exchange: "my.test.exchange88",
			Queue:    "my.test.queue88",
			Content:  &v,
		})
	}
}
