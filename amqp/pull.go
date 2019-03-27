package rabbitmq

import (
	"encoding/json"
	"fmt"
	"github.com/godaddy-x/jorm/log"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

var (
	pull_mgrs = make(map[string]*PullManager)
)

type PullManager struct {
	mu   sync.Mutex
	conn *amqp.Connection
	pull *PullMQ
}

type PullMQ struct {
	group     sync.WaitGroup
	channel   *amqp.Channel
	receivers []Receiver
}

func (self *PullManager) InitConfig(input ...AmqpConfig) *PullManager {
	for _, v := range input {
		c, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", v.Username, v.Password, v.Host, v.Port))
		if err != nil {
			panic("RabbitMQ初始化失败: " + err.Error())
		}
		channel, err := c.Channel()
		if err != nil {
			panic("打开Channel失败: " + err.Error())
		}
		pull_mgr := &PullManager{
			conn: c,
			pull: &PullMQ{
				channel:   channel,
				receivers: make([]Receiver, 0),
			},
		}
		if len(v.DsName) == 0 {
			v.DsName = MASTER
		}
		pull_mgrs[v.DsName] = pull_mgr
	}
	return self
}

func (self *PullManager) Client(dsname ...string) (*PullManager, error) {
	var ds string
	if len(dsname) > 0 && len(dsname[0]) > 0 {
		ds = dsname[0]
	} else {
		ds = MASTER
	}
	manager := pull_mgrs[ds]
	return manager, nil
}

func (self *PullManager) StartPullServer(receivers ...Receiver) {
	for _, v := range receivers {
		self.pull.receivers = append(self.pull.receivers, v)
	}
	self.pull.Start()
}

func (self *PullManager) AddPullReceiver(receivers ...Receiver) {
	for _, v := range receivers {
		self.pull.receivers = append(self.pull.receivers, v)
		go self.pull.listen(v)
	}
}

func (self *PullMQ) prepareExchange(exchange string) error {
	return self.channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
}

func (self *PullMQ) prepareQueue(exchange, queue string) error {
	if _, err := self.channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := self.channel.QueueBind(queue, queue, exchange, false, nil); err != nil {
		return err
	}
	return nil
}

func (self *PullMQ) run() {
	for _, receiver := range self.receivers {
		self.group.Add(1)
		go self.listen(receiver) // 每个接收者单独启动一个goroutine用来初始化queue并接收消息
	}
	self.group.Wait()
	log.Error("消费通道意外关闭,需要重新连接")
	self.channel.Close()
}

func (self *PullMQ) Start() {
	for {
		self.run()
		time.Sleep(3 * time.Second)
	}
}

func (self *PullMQ) listen(receiver Receiver) {
	defer self.group.Done()
	if err := self.prepareExchange(receiver.ExchangeName()); err != nil {
		receiver.OnError(fmt.Errorf("初始化交换机 [%s] 失败: %s", receiver.ExchangeName(), err.Error()))
		return
	}
	if err := self.prepareQueue(receiver.ExchangeName(), receiver.QueueName()); err != nil {
		receiver.OnError(fmt.Errorf("绑定队列 [%s] 到交换机失败: %s", receiver.QueueName(), err.Error()))
		return
	}
	//go func() {
	//	time.Sleep(3 * time.Second)
	//	for i := 0; i < 10; i++ {
	//		cli, _ := new(PublishManager).Client()
	//		v := map[string]interface{}{"test": 1}
	//		cli.Publish(MsgData{
	//			Exchange: receiver.ExchangeName(),
	//			Queue:    receiver.QueueName(),
	//			Content:  &v,
	//		})
	//	}
	//}()
	fmt.Sprintf("队列[%s - %s]消费服务启动成功...", receiver.ExchangeName(), receiver.QueueName())
	self.channel.Qos(1, 0, true)
	if msgs, err := self.channel.Consume(receiver.QueueName(), "", false, false, false, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("获取队列 %s 的消费通道失败: %s", receiver.QueueName(), err.Error()))
	} else {
		for msg := range msgs {
			for !receiver.OnReceive(msg.Body) {
				fmt.Println("receiver 数据处理失败，将要重试")
				time.Sleep(1 * time.Second)
			}
			msg.Ack(false)
		}
	}
}

type Receiver interface {
	ExchangeName() string
	QueueName() string
	OnError(err error)
	OnReceive(b []byte) bool
}

// 监听对象
type PullReceiver struct {
	Exchange string
	Queue    string
	LisData  LisData
	Callback func(msg MsgData) (MsgData, error)
}

func (self *PullReceiver) ExchangeName() string {
	return self.Exchange
}

func (self *PullReceiver) QueueName() string {
	return self.Queue
}

func (self *PullReceiver) OnError(err error) {
	log.Error(err.Error())
}

func (self *PullReceiver) OnReceive(b []byte) bool {
	if b == nil || len(b) == 0 || string(b) == "{}" {
		return true
	}
	log.Debug("消费数据日志", log.String("data", string(b)))
	message := MsgData{}
	if err := json.Unmarshal(b, &message); err != nil {
		log.Error("MQ消费数据转换JSON失败", log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.String("data", string(b)))
	} else if message.Content == nil {
		log.Error("MQ消费数据Content为空", log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.Any("data", message))
	} else if call, err := self.Callback(message); err != nil {
		log.Error("MQ消费数据处理异常", log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.Any("data", call), log.AddError(err))
		if self.LisData.IsNack {
			return false
		}
	}
	return true
}
