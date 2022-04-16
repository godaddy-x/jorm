package rabbitmq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/godaddy-x/jorm/log"
	"github.com/godaddy-x/jorm/util"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

var (
	pull_mgrs = make(map[string]*PullManager)
)

type PullManager struct {
	mu        sync.Mutex
	conn      *amqp.Connection
	receivers []*PullReceiver
}

func (self *PullManager) InitConfig(input ...AmqpConfig) *PullManager {
	for _, v := range input {
		c, err := amqp.DialConfig(fmt.Sprintf("amqp://%s:%s@%s:%d/", v.Username, v.Password, v.Host, v.Port), amqp.Config{ChannelMax: 999999})
		if err != nil {
			panic("RabbitMQ初始化失败: " + err.Error())
		}
		pull_mgr := &PullManager{
			conn:      c,
			receivers: make([]*PullReceiver, 0),
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

func (self *PullManager) AddPullReceiver(receivers ...*PullReceiver) {
	for _, v := range receivers {
		go self.start(v)
	}
}

func (self *PullManager) start(receiver *PullReceiver) {
	self.receivers = append(self.receivers, receiver)
	for {
		receiver.group.Add(1)
		go self.listen(receiver)
		receiver.group.Wait()
		log.Error("mq channel close: 消费通道意外关闭,需要重新连接", 0)
		if err := receiver.channel.Close(); err != nil {
			log.Error("close mq channel fail", 0, log.AddError(err))
		}
		time.Sleep(3 * time.Second)
	}
}

func (self *PullManager) listen(receiver *PullReceiver) {
	defer receiver.group.Done()
	channel, err := self.conn.Channel()
	if err != nil {
		log.Error("初始化Channel异常", 0, log.AddError(err))
	} else {
		receiver.channel = channel
	}
	exchange := receiver.ExchangeName()
	queue := receiver.QueueName()
	log.Error(fmt.Sprintf("消费队列[%s - %s]服务启动成功...", exchange, queue), 0)
	// testSend(exchange, queue)
	if err := self.prepareExchange(channel, exchange); err != nil {
		receiver.OnError(fmt.Errorf("初始化交换机 [%s] 失败: %s", exchange, err.Error()))
		return
	}
	if err := self.prepareQueue(channel, exchange, queue); err != nil {
		receiver.OnError(fmt.Errorf("绑定队列 [%s] 到交换机失败: %s", queue, err.Error()))
		return
	}
	count := receiver.LisData.PrefetchCount
	if count == 0 {
		count = 1
	}
	size := receiver.LisData.PrefetchSize
	channel.Qos(count, size, false)
	if msgs, err := channel.Consume(queue, "", false, false, false, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("获取队列 %s 的消费通道失败: %s", queue, err.Error()))
	} else {
		for msg := range msgs {
			for !receiver.OnReceive(msg.Body) {
				//fmt.Println("receiver 数据处理失败，将要重试")
				time.Sleep(3 * time.Second)
			}
			msg.Ack(false)
		}
	}
}

func (self *PullManager) prepareExchange(channel *amqp.Channel, exchange string) error {
	return channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
}

func (self *PullManager) prepareQueue(channel *amqp.Channel, exchange, queue string) error {
	if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := channel.QueueBind(queue, queue, exchange, false, nil); err != nil {
		return err
	}
	return nil
}

func testSend(exchange, queue string) {
	go func() {
		time.Sleep(3 * time.Second)
		for i := 0; i < 6; i++ {
			cli, _ := new(PublishManager).Client()
			v := map[string]interface{}{"test": 1}
			cli.Publish(MsgData{
				Exchange: exchange,
				Queue:    queue,
				Content:  &v,
			})
		}
	}()
}

//type Receiver interface {
//	Group() *sync.WaitGroup
//	Channel() *amqp.Channel
//	ExchangeName() string
//	QueueName() string
//	OnError(err error)
//	OnReceive(b []byte) bool
//}

func (self *PullReceiver) Channel() *amqp.Channel {
	return self.channel
}

func (self *PullReceiver) ExchangeName() string {
	return self.Exchange
}

func (self *PullReceiver) QueueName() string {
	return self.Queue
}

func (self *PullReceiver) OnError(err error) {
	log.Error("channel pull fail", 0, log.AddError(err))
}

// 监听对象
type PullReceiver struct {
	group    sync.WaitGroup
	channel  *amqp.Channel
	Exchange string
	Queue    string
	LisData  LisData
	Callback func(msg MsgData) (MsgData, error)
}

func (self *PullReceiver) OnReceive(b []byte) bool {
	if b == nil || len(b) == 0 || string(b) == "{}" {
		return true
	}
	if log.IsDebug() {
		defer log.Debug("MQ消费数据日志", util.Time(), log.String("message", string(b)))
	}
	message := MsgData{}
	if err := jsonUnmarshal(b, &message); err != nil {
		defer log.Error("MQ消费数据转换JSON失败", util.Time(), log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.String("message", string(b)))
	} else if message.Content == nil {
		defer log.Error("MQ消费数据Content为空", util.Time(), log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.Any("message", message))
	} else if call, err := self.Callback(message); err != nil {
		defer log.Error("MQ消费数据处理异常", util.Time(), log.String("exchange", self.Exchange), log.String("queue", self.Queue), log.Any("message", call), log.AddError(err))
		if self.LisData.IsNack {
			return false
		}
	}
	return true
}

func jsonUnmarshal(data []byte, v interface{}) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	d := json.NewDecoder(bytes.NewBuffer(data))
	d.UseNumber()
	return d.Decode(v)
}
