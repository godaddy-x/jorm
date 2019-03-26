package rabbitmq

import (
	"fmt"
	"github.com/godaddy-x/jorm/log"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

var (
	pull *PullMQ
)

type Receiver interface {
	QueueName() string
	OnError(err error)
	OnReceive(b []byte) bool
}

// 监听对象
type PullReceiver struct {
	Queue string
}

func (self *PullReceiver) QueueName() string {
	return self.Queue
}

func (self *PullReceiver) OnError(err error) {
	log.Error(err.Error())
}

func (self *PullReceiver) OnReceive(b []byte) bool {
	fmt.Println("接收到数据: ", string(b))
	return true
}

type PullMQ struct {
	group        sync.WaitGroup
	channel      *amqp.Channel
	exchangeName string
	exchangeType string
	receivers    []Receiver
}

func NewPullServer(url, exchangeName, exchangeType string) {
	conn, err := amqp.Dial(url)
	if err != nil {
		panic("RabbitMQ初始化失败: " + err.Error())
	}
	channel, err := conn.Channel()
	if err != nil {
		panic("打开Channel失败: " + err.Error())
	}
	pull = &PullMQ{
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		channel:      channel,
	}
}

func StartPullServer(receivers ...Receiver) {
	for _, v := range receivers {
		pull.receivers = append(pull.receivers, v)
	}
	pull.Start()
}

func AddPullReceiver(receivers ...Receiver) {
	for _, v := range receivers {
		pull.receivers = append(pull.receivers, v)
		go pull.listen(v)
	}
}

func (self *PullMQ) prepareExchange() error {
	return self.channel.ExchangeDeclare(self.exchangeName, self.exchangeType, true, false, false, false, nil)
}

func (self *PullMQ) run() {
	if err := self.prepareExchange(); err != nil {
		log.Error(fmt.Sprintf("初始化交换机[%s]失败", self.exchangeName))
	}
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
	queueName := receiver.QueueName()
	routerKey := receiver.QueueName()
	if _, err := self.channel.QueueDeclare(
		queueName, true, false, false, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("初始化队列 %s 失败: %s", queueName, err.Error()))
	}
	if err := self.channel.QueueBind(queueName, routerKey, self.exchangeName, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("绑定队列 [%s - %s] 到交换机失败: %s", queueName, routerKey, err.Error()))
	}
	go func() {
		for i := 0; i < 10; i++ {
			publish := amqp.Publishing{ContentType: "text/plain", Body: []byte(fmt.Sprintf("[%s]通道测试发送: %d", queueName, i))}
			self.channel.Publish(self.exchangeName, queueName, false, false, publish)
			time.Sleep(100 * time.Millisecond)
		}
	}()
	fmt.Sprintf("队列[%s - %s]消费服务启动成功...", self.exchangeName, queueName)
	self.channel.Qos(1, 0, true)
	if msgs, err := self.channel.Consume(queueName, "", false, false, false, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("获取队列 %s 的消费通道失败: %s", queueName, err.Error()))
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
