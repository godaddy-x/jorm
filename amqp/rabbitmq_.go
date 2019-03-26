package rabbitmq

import (
	"fmt"
	"github.com/godaddy-x/jorm/log"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

var (
	pullMQ *RabbitMQ
)

// Receiver 观察者模式需要的接口
// 观察者用于接收指定的queue到来的数据
type Receiver interface {
	QueueName() string     // 获取接收者需要监听的队列
	OnError(error)         // 处理遇到的错误，当RabbitMQ对象发生了错误，他需要告诉接收者处理错误
	OnReceive([]byte) bool // 处理收到的消息, 这里需要告知RabbitMQ对象消息是否处理成功
}

// 监听对象
type SimpleReceiver struct {
	Queue string
}

func (mq *SimpleReceiver) QueueName() string {
	return mq.Queue
}

func (mq *SimpleReceiver) OnError(err error) {
	log.Error(err.Error())
}

func (mq *SimpleReceiver) OnReceive(b []byte) bool {
	fmt.Println("接收到数据: ", string(b))
	return true
}

// RabbitMQ 用于管理和维护rabbitmq的对象
type RabbitMQ struct {
	group        sync.WaitGroup
	channel      *amqp.Channel
	exchangeName string // exchange的名称
	exchangeType string // exchange的类型
	receivers    []Receiver
}

// New 创建一个新的操作RabbitMQ的对象
func NewPullMQ(url, exchangeName, exchangeType string) {
	// 这里可以根据自己的需要去定义
	conn, err := amqp.Dial(url)
	if err != nil {
		panic("RabbitMQ初始化失败: " + err.Error())
	}
	channel, err := conn.Channel()
	if err != nil {
		panic("打开Channel失败: " + err.Error())
	}
	pullMQ = &RabbitMQ{
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		channel:      channel,
	}
}

// 启动项目监听
func StartServer(receivers ...Receiver) {
	for _, v := range receivers {
		pullMQ.receivers = append(pullMQ.receivers, v)
	}
	pullMQ.Start()
}

// RegisterReceiver 注册一个用于接收指定队列指定路由的数据接收者
func AddReceiver(receivers ...Receiver) {
	for _, v := range receivers {
		pullMQ.receivers = append(pullMQ.receivers, v)
		go pullMQ.listen(v)
	}
}

// prepareExchange 准备rabbitmq的Exchange
func (self *RabbitMQ) prepareExchange() error {
	return self.channel.ExchangeDeclare(self.exchangeName, self.exchangeType, true, false, false, false, nil)
}

// run 开始获取连接并初始化相关操作
func (self *RabbitMQ) run() {
	// 初始化Exchange
	if err := self.prepareExchange(); err != nil {
		panic(err)
	}
	// 启动监听器和注册消费函数
	for _, receiver := range self.receivers {
		self.group.Add(1)
		go self.listen(receiver) // 每个接收者单独启动一个goroutine用来初始化queue并接收消息
	}
	self.group.Wait()
	log.Error("rabbitmq意外关闭,需要重新连接")
	// 理论上mq.run()在程序的执行过程中是不会结束的
	// 一旦结束就说明所有的接收者都退出了，那么意味着程序与rabbitmq的连接断开
	// 那么则需要重新连接，这里尝试销毁当前连接
	self.channel.Close()
}

// Start 启动Rabbitmq的客户端
func (self *RabbitMQ) Start() {
	for {
		self.run()
		// 一旦连接断开,那么需要隔一段时间去重连,这里最好有一个时间间隔
		time.Sleep(3 * time.Second)
	}
}

// Listen 监听指定路由发来的消息
// 这里需要针对每一个接收者启动一个goroutine来执行listen
// 该方法负责从每一个接收者监听的队列中获取数据，并负责重试
func (self *RabbitMQ) listen(receiver Receiver) {
	defer self.group.Done()
	// 这里获取每个接收者需要监听的队列和路由
	queueName := receiver.QueueName()
	routerKey := receiver.QueueName()
	// 初始化Queue
	if _, err := self.channel.QueueDeclare(
		queueName, true, false, false, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("初始化队列 %s 失败: %s", queueName, err.Error()))
	}
	// 将Queue绑定到Exchange上去
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
	// 获取消费通道
	self.channel.Qos(1, 0, true) // 确保rabbitmq会一个一个发消息
	if msgs, err := self.channel.Consume(queueName, "", false, false, false, false, nil); err != nil {
		receiver.OnError(fmt.Errorf("获取队列 %s 的消费通道失败: %s", queueName, err.Error()))
	} else {
		// 使用callback消费数据
		for msg := range msgs {
			// 当接收者消息处理失败的时候，
			// 比如网络问题导致的数据库连接失败，redis连接失败等等这种
			// 通过重试可以成功的操作，那么这个时候是需要重试的
			// 直到数据处理成功后再返回，然后才会回复rabbitmq ack
			for !receiver.OnReceive(msg.Body) {
				fmt.Println("receiver 数据处理失败，将要重试")
				time.Sleep(1 * time.Second)
			}
			// 确认收到本条消息, multiple必须为false
			msg.Ack(false)
		}
	}
}
