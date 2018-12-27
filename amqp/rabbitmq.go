package rabbitmq

import (
	"errors"
	"github.com/godaddy-x/jorm/sqld"
	"github.com/godaddy-x/jorm/util"
	"github.com/streadway/amqp"
	"log"
	"sync"
)

const (
	MASTER = "MASTER"
	DIRECT = "direct"
	DLX    = "x-dead-letter-exchange"
)

var (
	channels      sync.Map
	amqp_sessions = make(map[string]*AmqpManager, 0)
)

type AmqpManager struct {
	DsName  string
	channel *amqp.Channel
}

// Amqp配置参数
type AmqpConfig struct {
	DsName   string
	Host     string
	Port     int
	Username string
	Password string
}

// Amqp消息参数
type MsgData struct {
	Exchange string      `json:"exchange"`
	Queue    string      `json:"queue"`
	Kind     string      `json:"kind"`
	Content  interface{} `json:"content"`
	Type     int64       `json:"type"`
	Delay    int64       `json:"delay"`
	Retries  int64       `json:"retries"`
}

// Amqp延迟发送配置
type DlxConfig struct {
	Exchange string // 交换机
	DlQueue  string // 死信队列
	RtQueue  string // 重读队列
}

// Amqp监听配置参数
type LisData struct {
	Exchange      string
	Queue         string
	Kind          string
	PrefetchCount int
	PrefetchSize  int
	SendMgo       bool
}

// Amqp消息异常日志
type MQErrorLog struct {
	Id       int64       `json:"id" bson:"_id" tb:"mq_error_log" mg:"true"`
	Exchange string      `json:"exchange" bson:"exchange"`
	Queue    string      `json:"queue" bson:"queue"`
	Content  interface{} `json:"content" bson:"content"`
	Type     int64       `json:"type" bson:"type"`
	Delay    int64       `json:"delay" bson:"delay"`
	Retries  int64       `json:"retries" bson:"retries"`
	Error    string      `json:"error" bson:"error"`
	Ctime    int64       `json:"ctime" bson:"ctime"`
	Utime    int64       `json:"utime" bson:"utime"`
	State    int64       `json:"state" bson:"state"`
}

func (self *AmqpManager) InitConfig(input ...AmqpConfig) {
	for e := range input {
		conf := input[e]
		mq, err := amqp.Dial(util.AddStr("amqp://", conf.Username, ":", conf.Password, "@", conf.Host, ":", util.AnyToStr(conf.Port), "/"))
		if err != nil {
			panic("连接RabbitMQ失败,请检查...")
		}
		channel, err := mq.Channel()
		if err != nil {
			panic("创建RabbitMQ Channel失败,请检查...")
		}
		if len(conf.DsName) > 0 {
			amqp_sessions[conf.DsName] = &AmqpManager{DsName: conf.DsName, channel: channel}
		} else {
			amqp_sessions[MASTER] = &AmqpManager{DsName: MASTER, channel: channel}
		}
	}
}

func (self *AmqpManager) Client(dsname ...string) (*AmqpManager, error) {
	var ds string
	if len(dsname) > 0 && len(dsname[0]) > 0 {
		ds = dsname[0]
	} else {
		ds = MASTER
	}
	manager := amqp_sessions[ds]
	if manager.channel == nil {
		return nil, util.Error("amqp数据源[", ds, "]未找到,请检查...")
	}
	return manager, nil
}

func (self *AmqpManager) bindExchangeAndQueue(exchange, queue, kind string, table amqp.Table) error {
	exist, _ := channels.Load(util.AddStr(exchange, ":", queue))
	if exist == nil {
		if len(kind) == 0 {
			kind = DIRECT
		}
		err := self.channel.ExchangeDeclare(exchange, kind, true, false, false, false, nil)
		if err != nil {
			return errors.New(util.AddStr("创建exchange[", exchange, "]失败,请重新尝试..."))
		}
		if _, err = self.channel.QueueDeclare(queue, true, false, false, false, table); err != nil {
			return errors.New(util.AddStr("创建queue[", queue, "]失败,请重新尝试..."))
		}
		if err := self.channel.QueueBind(queue, queue, exchange, false, nil); err != nil {
			return errors.New(util.AddStr("exchange[", exchange, "]和queue[", queue, "]绑定失败,请重新尝试..."))
		}
		channels.Store(util.AddStr(exchange, ":", queue), true)
	}
	return nil
}

// 根据通道发送信息,如通道不存在则自动创建
func (self *AmqpManager) Publish(data MsgData, dlx ...DlxConfig) error {
	if len(data.Exchange) == 0 || len(data.Queue) == 0 {
		return errors.New(util.AddStr("exchange,queue不能为空"))
	}
	if data.Content == nil {
		return errors.New(util.AddStr("content不能为空"))
	}
	if err := self.bindExchangeAndQueue(data.Exchange, data.Queue, data.Kind, nil); err != nil {
		return err
	}
	body, err := util.ObjectToJson(data)
	if err != nil {
		return errors.New("发送失败,消息无法转成JSON字符串: " + err.Error())
	}
	exchange := data.Exchange
	queue := data.Queue
	publish := amqp.Publishing{ContentType: "text/plain", Body: []byte(body)}
	if dlx != nil && len(dlx) > 0 {
		conf := dlx[0]
		if len(conf.Exchange) == 0 {
			return errors.New(util.AddStr("死信交换机不能为空"))
		}
		if len(conf.DlQueue) == 0 {
			return errors.New(util.AddStr("死信队列不能为空"))
		}
		if len(conf.RtQueue) == 0 {
			return errors.New(util.AddStr("重读队列不能为空"))
		}
		if err := self.bindExchangeAndQueue(conf.Exchange, conf.RtQueue, DIRECT, nil); err != nil {
			return err
		}
		if err := self.bindExchangeAndQueue(conf.Exchange, conf.DlQueue, DIRECT, amqp.Table{DLX: conf.RtQueue}); err != nil {
			return err
		}
		if data.Delay <= 0 {
			return errors.New(util.AddStr("延时发送时间必须大于0毫秒"))
		}
		exchange = conf.Exchange
		queue = conf.DlQueue
		publish.Expiration = util.AnyToStr(data.Delay)
	}
	if err := self.channel.Publish(exchange, queue, false, false, publish); err != nil {
		return errors.New(util.AddStr("[", data.Exchange, "][", data.Queue, "][", body, "]发送失败: ", err.Error()))
	}
	return nil
}

// 监听指定队列消息
func (self *AmqpManager) Pull(data LisData, callback func(msg MsgData) (MsgData, error)) (err error) {
	if len(data.Exchange) == 0 || len(data.Queue) == 0 {
		return errors.New(util.AddStr("exchange,queue不能为空"))
	}
	if err := self.bindExchangeAndQueue(data.Exchange, data.Queue, data.Kind, nil); err != nil {
		return err
	}
	log.Println(util.AddStr("exchange[", data.Exchange, "] - queue[", data.Queue, "] MQ监听服务启动成功..."))
	self.channel.Qos(data.PrefetchCount, data.PrefetchSize, true)
	delivery, err := self.channel.Consume(data.Queue, "", false, false, false, false, nil)
	if err != nil {
		log.Println(util.AddStr("exchange[", data.Exchange, "] - queue[", data.Queue, "] MQ监听服务启动失败: ", err.Error()))
		return err
	}
	for d := range delivery {
		body := string(d.Body)
		if len(body) > 0 {
			message := MsgData{}
			if err := util.JsonToObject(body, &message); err != nil {
				log.Println(util.AddStr("exchange[", data.Exchange, "] - queue[", data.Queue, "] 监听处理转换JSON失败: ", err.Error()))
			} else if message.Content == nil {
				log.Println(util.AddStr("exchange[", data.Exchange, "] - queue[", data.Queue, "] 监听处理数据为空"))
			} else {
				call, err := callback(message)
				if err != nil {
					log.Println(util.AddStr("exchange[", call.Exchange, "] - queue[", call.Queue, "] 监听处理异常: ", err.Error()))
					if data.SendMgo {
						uuid, _ := util.StrToInt64(util.GetUUID())
						errlog := MQErrorLog{Id: uuid, Exchange: call.Exchange, Queue: call.Queue, Type: call.Type, Retries: call.Retries, Delay: call.Delay, Content: call.Content, Error: err.Error(), Ctime: util.Time(), Utime: util.Time(), State: 1}
						if mongo, err := new(sqld.MGOManager).Get(); err != nil {
							log.Println(err.Error())
						} else {
							defer mongo.Close()
							if err := mongo.Save(&errlog); err != nil {
								log.Println(err.Error())
							}
						}
					}
				}
			}
		}
		d.Ack(false)
	}
	return nil
}
