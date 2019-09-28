package consul

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/godaddy-x/jorm/amqp"
	"github.com/godaddy-x/jorm/util"
	consulapi "github.com/hashicorp/consul/api"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"reflect"
	"strings"
	"time"
)

var (
	DefaultHost     = "consulx.com:8500"
	consul_sessions = make(map[string]*ConsulManager)
)

type ConsulManager struct {
	Host    string
	Consulx *consulapi.Client
	Config  *ConsulConfig
}

// Consulx配置参数
type ConsulConfig struct {
	DsName       string
	Node         string
	Host         string
	Domain       string
	CheckPort    int
	RpcPort      int
	ListenProt   int
	Protocol     string
	Logger       string
	Timeout      string
	Interval     string
	DestroyAfter string
}

// RPC日志
type MonitorLog struct {
	ConsulHost  string
	RpcHost     string
	RpcPort     int
	Protocol    string
	ServiceName string
	MethodName  string
	BeginTime   int64
	CostTime    int64
	Errors      []string
}

func (self *ConsulManager) InitConfig(input ...ConsulConfig) (*ConsulManager, error) {
	for e := range input {
		conf := input[e]
		config := consulapi.DefaultConfig()
		config.Address = conf.Host
		client, err := consulapi.NewClient(config)
		if err != nil {
			panic(util.AddStr("连接", conf.Host, "Consul配置中心失败: ", err.Error()))
		}
		manager := &ConsulManager{Consulx: client, Host: conf.Host}
		data, err := manager.GetKV(conf.Node, client)
		if err != nil {
			panic(err)
		}
		result := &ConsulConfig{}
		if err := util.ReadJsonConfig(data, result); err != nil {
			panic(err)
		}
		result.Node = conf.Node
		manager.Config = result
		if len(conf.DsName) == 0 {
			consul_sessions[conf.Host] = manager
		} else {
			consul_sessions[conf.DsName] = manager
		}
	}
	if len(consul_sessions) == 0 {
		log.Println("consul连接初始化失败: 数据源为0")
	}
	return self, nil
}

func (self *ConsulManager) Client(dsname ...string) (*ConsulManager, error) {
	var ds string
	if len(dsname) > 0 && len(dsname[0]) > 0 {
		ds = dsname[0]
	} else {
		ds = DefaultHost
	}
	manager := consul_sessions[ds]
	if manager == nil {
		return nil, util.Error("consul数据源[", ds, "]未找到,请检查...")
	}
	return manager, nil
}

// 通过Consul中心获取指定配置数据
func (self *ConsulManager) GetKV(key string, consulx ...*consulapi.Client) ([]byte, error) {
	client := self.Consulx
	if len(consulx) > 0 {
		client = consulx[0]
	}
	kv := client.KV()
	if kv == nil {
		return nil, util.Error("Consul配置[", key, "]没找到,请检查...")
	}
	k, _, err := kv.Get(key, nil)
	if err != nil {
		return nil, util.Error("Consul配置数据[", key, "]读取异常,请检查...")
	}
	if k == nil || k.Value == nil || len(k.Value) == 0 {
		return nil, util.Error("Consul配置数据[", key, "]读取为空,请检查...")
	}
	return k.Value, nil
}

// 读取节点JSON配置
func (self *ConsulManager) ReadJsonConfig(node string, result interface{}) error {
	if data, err := self.GetKV(node); err != nil {
		return err
	} else {
		return util.ReadJsonConfig(data, result)
	}
}

// 中心注册接口服务
func (self *ConsulManager) AddRegistration(name string, iface interface{}) {
	tof := reflect.TypeOf(iface)
	vof := reflect.ValueOf(iface)
	registration := new(consulapi.AgentServiceRegistration)
	ip := util.GetLocalIP()
	if ip == "" {
		panic("内网IP读取失败,请检查...")
	}
	orname := reflect.Indirect(vof).Type().Name()
	sname := orname
	methods := ","
	for m := 0; m < tof.NumMethod(); m++ {
		method := tof.Method(m)
		methods = util.AddStr(methods, method.Name, ",")
		sname = util.AddStr(sname, "/", method.Name)
	}
	if len(methods) == 0 {
		panic(util.AddStr("服务对象[", sname, "]尚未有方法..."))
	}
	sname = util.AddStr(ip, "/", sname)
	registration.ID = sname
	registration.Name = util.AddStr(ip, "/", orname)
	registration.Tags = []string{name}
	registration.Address = ip
	registration.Port = self.Config.RpcPort
	meta := make(map[string]string)
	meta["host"] = ip + ":" + util.AnyToStr(self.Config.ListenProt)
	meta["protocol"] = self.Config.Protocol
	meta["version"] = "1.0.0"
	meta["methods"] = methods
	registration.Meta = meta
	registration.Check = &consulapi.AgentServiceCheck{HTTP: fmt.Sprintf("http://%s:%d%s", registration.Address, self.Config.CheckPort, "/check"), Timeout: self.Config.Timeout, Interval: self.Config.Interval, DeregisterCriticalServiceAfter: self.Config.DestroyAfter,}
	// 启动RPC服务
	log.Println(util.AddStr("[", util.GetLocalIP(), "]", " - [", name, "] - 启动成功"))
	if err := self.Consulx.Agent().ServiceRegister(registration); err != nil {
		panic(util.AddStr("Consul注册[", name, "]服务失败: ", err.Error()))
	}
	rpc.Register(iface)
}

// 开启并监听服务
func (self *ConsulManager) StartListenAndServe() {
	l, e := net.Listen(self.Config.Protocol, util.AddStr(":", util.AnyToStr(self.Config.ListenProt)))
	if e != nil {
		panic("Consul监听服务异常: " + e.Error())
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Print("Error: accept rpc connection", err.Error())
				continue
			}
			go func(conn net.Conn) {
				buf := bufio.NewWriter(conn)
				srv := &gobServerCodec{
					rwc:    conn,
					dec:    gob.NewDecoder(conn),
					enc:    gob.NewEncoder(buf),
					encBuf: buf,
				}
				err = rpc.ServeRequest(srv)
				if err != nil {
					log.Print("Error: server rpc request", err.Error())
				}
				srv.Close()
			}(conn)
		}
	}()
	http.HandleFunc("/check", self.healthCheck)
	http.ListenAndServe(fmt.Sprintf(":%d", self.Config.CheckPort), nil)
}

// 获取RPC服务,并执行访问 args参数不可变,reply参数可变
func (self *ConsulManager) CallService(srv string, args interface{}, reply interface{}) error {
	if srv == "" {
		return errors.New("类名+方法名不能为空")
	}
	sarr := strings.Split(srv, ".")
	if len(sarr) != 2 {
		return errors.New("类名.方法名格式有误")
	}
	srvName := sarr[0]
	methodName := sarr[1]
	if len(srvName) == 0 || len(methodName) == 0 {
		return errors.New("类名或方法名不能为空")
	}
	monitor := MonitorLog{
		ConsulHost:  self.Config.Host,
		ServiceName: srvName,
		MethodName:  methodName,
		BeginTime:   util.Time(),
	}
	services, err := self.Consulx.Agent().Services()
	if err != nil {
		return errors.New(util.AddStr("读取RPC服务失败: ", err.Error()))
	}
	if len(services) == 0 {
		return errors.New("没有找到任何RPC服务")
	}
	srvs := make([]*consulapi.AgentService, 0)
	for _, v := range services {
		if util.HasStr(v.ID, util.AddStr("/", srvName, "/")) {
			srvs = append(srvs, v)
		}
	}
	if len(srvs) == 0 {
		return errors.New(util.AddStr("[", srvName, "]无可用服务"))
	}
	agent := srvs[0]
	meta := agent.Meta
	if len(meta) < 4 {
		return errors.New(util.AddStr("[[", agent.ID, "]服务参数异常,请检查..."))
	}
	methods := meta["methods"]
	s := "," + methodName + ","
	if !util.HasStr(methods, s) {
		return errors.New(util.AddStr("[", agent.ID, "][", methodName, "]无效,请检查..."))
	}
	host := meta["host"]
	protocol := meta["protocol"]
	if len(host) == 0 || len(protocol) == 0 {
		return errors.New("meta参数异常")
	} else {
		s := strings.Split(host, ":")
		port, _ := util.StrToInt(s[1])
		monitor.RpcHost = s[0]
		monitor.RpcPort = port
		monitor.Protocol = protocol
	}
	conn, err := net.DialTimeout(protocol, host, time.Second*10)
	if err != nil {
		log.Println(util.AddStr("[", agent.ID, "][", host, "]", "[", srv, "]连接失败: ", err.Error()))
		err := errors.New(util.AddStr("[", agent.ID, "][", host, "]", "[", srv, "]连接失败: ", err.Error()))
		self.rpcMonitor(monitor, err)
		return err
	}
	encBuf := bufio.NewWriter(conn)
	codec := &gobClientCodec{conn, gob.NewDecoder(conn), gob.NewEncoder(encBuf), encBuf}
	cli := rpc.NewClientWithCodec(codec)
	err1 := cli.Call(srv, args, reply)
	err2 := cli.Close()
	if err1 != nil && err2 != nil {
		msg := util.AddStr("[", host, "]", "[", srv, "]访问失败: ", err1.Error(), ";", err2.Error())
		log.Println(msg)
		return self.rpcMonitor(monitor, errors.New(util.AddStr(err1.Error(), " -------- ", err2.Error())))
	}
	if err1 != nil {
		msg := util.AddStr("[", host, "]", "[", srv, "]访问失败: ", err1.Error())
		log.Println(msg)
		return self.rpcMonitor(monitor, err1)
	} else if err2 != nil {
		msg := util.AddStr("[", host, "]", "[", srv, "]访问失败: ", err2.Error())
		log.Println(msg)
		return self.rpcMonitor(monitor, err2)
	}
	return self.rpcMonitor(monitor, nil)
}

// 输出RPC监控日志
func (self *ConsulManager) rpcMonitor(monitor MonitorLog, err error) error {
	monitor.CostTime = util.Time() - monitor.BeginTime
	if err != nil {
		monitor.Errors = []string{err.Error()}
		log.Println(util.ObjectToJson(monitor))
	}
	if self.Config.Logger == "local" {
		log.Println(util.ObjectToJson(monitor))
	} else if self.Config.Logger == "amqp" {
		if client, err := new(rabbitmq.PublishManager).Client(); err != nil {
			return err
		} else if err := client.Publish(rabbitmq.MsgData{Exchange: "stdrpc.exchange", Queue: "stdrpc.monitor", Content: monitor}); err != nil {
			return err
		}
	}
	return err
}

// 接口服务健康检查
func (self *ConsulManager) healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "consulCheck")
}
