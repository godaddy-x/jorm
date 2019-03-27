package rabbitmq

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"sync"
)

var (
	publish_mgrs = make(map[string]*PublishManager)
)

type PublishManager struct {
	mu       sync.Mutex
	conn     *amqp.Connection
	channels map[string]*PublishMQ
}

type PublishMQ struct {
	channel  *amqp.Channel
	mu       sync.Mutex
	exchange string
	queue    string
}

func (self *PublishManager) InitConfig(input ...AmqpConfig) *PublishManager {
	for _, v := range input {
		c, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", v.Username, v.Password, v.Host, v.Port))
		if err != nil {
			panic("RabbitMQ初始化失败: " + err.Error())
		}
		publish_mgr := &PublishManager{
			conn:     c,
			channels: make(map[string]*PublishMQ),
		}
		if len(v.DsName) == 0 {
			v.DsName = MASTER
		}
		publish_mgrs[v.DsName] = publish_mgr
	}
	return self
}

func (self *PublishManager) Client(dsname ...string) (*PublishManager, error) {
	var ds string
	if len(dsname) > 0 && len(dsname[0]) > 0 {
		ds = dsname[0]
	} else {
		ds = MASTER
	}
	manager := publish_mgrs[ds]
	return manager, nil
}

func (self *PublishManager) Publish(data MsgData) error {
	pub, ok := self.channels[data.Exchange+data.Queue]
	if !ok {
		self.mu.Lock()
		defer self.mu.Unlock()
		pub, ok = self.channels[data.Exchange+data.Queue]
		if !ok {
			channel, err := self.conn.Channel()
			if err != nil {
				return err
			}
			pub = &PublishMQ{channel: channel, exchange: data.Exchange, queue: data.Queue,}
			pub.prepareExchange()
			pub.prepareQueue()
			self.channels[data.Exchange+data.Queue] = pub
		}
	}
	return pub.sendToMQ(data)
}

func (self *PublishMQ) sendToMQ(v interface{}) error {
	b, _ := json.Marshal(v);
	data := amqp.Publishing{ContentType: "text/plain", Body: b}
	return self.channel.Publish(self.exchange, self.queue, false, false, data)
}

func (self *PublishMQ) prepareExchange() error {
	return self.channel.ExchangeDeclare(self.exchange, "direct", true, false, false, false, nil)
}

func (self *PublishMQ) prepareQueue() error {
	if _, err := self.channel.QueueDeclare(self.queue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := self.channel.QueueBind(self.queue, self.queue, self.exchange, false, nil); err != nil {
		return err
	}
	return nil
}
