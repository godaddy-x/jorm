package consul

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/godaddy-x/jorm/log"
	"github.com/godaddy-x/jorm/util"
	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"math/rand"
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
	consul_slowlog  *zap.Logger
)

type ConsulManager struct {
	Host      string
	Consulx   *consulapi.Client
	Config    *ConsulConfig
	Selection func([]*consulapi.AgentService) *consulapi.AgentService
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
	SlowQuery    int64  // 0.不开启筛选 >0开启筛选查询 毫秒
	SlowLogPath  string // 慢查询写入地址
}

// RPC日志
type MonitorLog struct {
	ConsulHost  string
	RpcHost     string
	RpcPort     int
	Protocol    string
	AgentID     string
	ServiceName string
	MethodName  string
	BeginTime   int64
	CostTime    int64
	Error       error
}

// RPC参数对象
type CallInfo struct {
	Tags     []string    // 服务标签名称
	Domain   string      // 自定义访问域名,为空时自动填充内网IP
	Iface    interface{} // 接口实现类实例
	Package  string      // RPC服务包名
	Service  string      // RPC服务名称
	Method   string      // RPC方法名称
	Protocol string      // RPC访问协议,默认TCP
	Request  interface{} // 请求参数对象
	Response interface{} // 响应参数对象
	Timeout  int64       // 连接请求超时,默认10秒
}

func (self *ConsulManager) InitConfig(input ...ConsulConfig) (*ConsulManager, error) {
	for _, conf := range input {
		config := consulapi.DefaultConfig()
		config.Address = conf.Host
		client, err := consulapi.NewClient(config)
		if err != nil {
			panic(util.AddStr("连接Consul[", conf.Host, "]配置中心失败: ", err.Error()))
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
		manager.initSlowLog()
		if len(result.DsName) == 0 {
			consul_sessions[conf.Host] = manager
		} else {
			consul_sessions[result.DsName] = manager
		}
	}
	if len(consul_sessions) == 0 {
		log.Println("consul连接初始化失败: 数据源为0")
	}
	return self, nil
}

func (self *ConsulManager) initSlowLog() {
	if self.Config.SlowQuery == 0 || len(self.Config.SlowLogPath) == 0 {
		return
	}
	if consul_slowlog == nil {
		consul_slowlog = log.InitNewLog(&log.ZapConfig{
			Level:   "warn",
			Console: false,
			FileConfig: &log.FileConfig{
				Compress:   true,
				Filename:   self.Config.SlowLogPath,
				MaxAge:     7,
				MaxBackups: 7,
				MaxSize:    512,
			}})
		fmt.Println("Consul监控日志服务启动成功...")
	}
}

func (self *ConsulManager) getSlowLog() *zap.Logger {
	return consul_slowlog
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
func (self *ConsulManager) AddRegistration(name string, iface interface{}, ipname ...string) {
	tof := reflect.TypeOf(iface)
	vof := reflect.ValueOf(iface)
	registration := new(consulapi.AgentServiceRegistration)
	ip := util.GetLocalIP()
	if ip == "" {
		panic("内网IP读取失败,请检查...")
	}
	if ipname != nil && len(ipname) > 0 && len(ipname[0]) > 0 {
		ip = ipname[0]
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

func (self *ConsulManager) RemoveService(serviceIDs ...string) {
	services, err := self.Consulx.Agent().Services()
	if err != nil {
		panic(err)
	}
	if len(serviceIDs) > 0 {
		for _, v := range services {
			for _, ID := range serviceIDs {
				if ID == v.ID {
					if err := self.Consulx.Agent().ServiceDeregister(v.ID); err != nil {
						log.Println(err)
					}
					log.Println("移除服务成功: ", v.Service, " - ", v.ID)
				}
			}
		}
		return
	}
	for _, v := range services {
		if err := self.Consulx.Agent().ServiceDeregister(v.ID); err != nil {
			log.Println(err)
		}
		log.Println("移除服务成功: ", v.Service, " - ", v.ID)
	}
}

// 根据服务名获取可用列表
func (self *ConsulManager) GetService(service string) ([]*consulapi.AgentService, error) {
	result := []*consulapi.AgentService{}
	services, err := self.Consulx.Agent().Services()
	if err != nil {
		return result, err
	}
	if len(service) == 0 {
		for _, v := range services {
			result = append(result, v)
		}
		return result, nil
	}
	for _, v := range services {
		if service == v.Service {
			result = append(result, v)
		}
	}
	return result, nil
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

	var agentID, host, protocol string
	for _, agent := range srvs {
		meta := agent.Meta
		if len(meta) < 4 {
			log.Warn("服务参数异常", 0, log.String("ID", agent.ID))
			continue
		}
		methods := meta["methods"]
		s := "," + methodName + ","
		if !util.HasStr(methods, s) {
			log.Warn("服务方法无效", 0, log.String("ID", agent.ID), log.String("method", methodName))
			continue
		}
		agentID = agent.ID
		host = meta["host"]
		protocol = meta["protocol"]
		if len(host) == 0 || len(protocol) == 0 {
			log.Warn("服务meta参数异常", 0, log.String("ID", agent.ID), log.String("method", methodName))
			continue
		} else {
			s := strings.Split(host, ":")
			port, _ := util.StrToInt(s[1])
			monitor.RpcHost = s[0]
			monitor.RpcPort = port
			monitor.Protocol = protocol
			monitor.AgentID = agentID
			break
		}
	}
	if len(host) == 0 || len(protocol) == 0 {
		return util.Error("没有找到任何RPC服务代理参数")
	}
	defer self.rpcMonitor(monitor, err, args, reply)
	var conn net.Conn
	var err1, err2 error
	conn, err = net.DialTimeout(protocol, host, time.Second*10)
	if err != nil {
		log.Error("consul服务连接失败", 0, log.String("ID", agentID), log.String("host", host), log.String("srv", srv), log.AddError(err))
		return util.Error("[", host, "]", "[", srv, "]连接失败: ", err)
	}
	encBuf := bufio.NewWriter(conn)
	codec := &gobClientCodec{conn, gob.NewDecoder(conn), gob.NewEncoder(encBuf), encBuf}
	cli := rpc.NewClientWithCodec(codec)
	err1 = cli.Call(srv, args, reply)
	err2 = cli.Close()
	if err1 != nil {
		log.Error("consul服务访问失败", 0, log.String("ID", agentID), log.String("host", host), log.String("srv", srv), log.AddError(err1))
		return util.Error("[", host, "]", "[", srv, "]访问失败: ", err1)
	} else if err2 != nil {
		log.Error("consul服务关闭失败", 0, log.String("ID", agentID), log.String("host", host), log.String("srv", srv), log.AddError(err2))
		return util.Error("[", host, "]", "[", srv, "]关闭失败: ", err2)
	}
	return nil
}

// 中心注册接口服务
func (self *ConsulManager) AddRPC(callInfo ...*CallInfo) {
	if len(callInfo) == 0 {
		panic("服务对象列表为空,请检查...")
	}
	services, err := self.GetService("")
	if err != nil {
		panic(err)
	}
	for _, info := range callInfo {
		tof := reflect.TypeOf(info.Iface)
		vof := reflect.ValueOf(info.Iface)
		registration := new(consulapi.AgentServiceRegistration)
		addr := util.GetLocalIP()
		if addr == "" {
			panic("内网IP读取失败,请检查...")
		}
		if len(info.Domain) > 0 {
			addr = info.Domain
		}
		srvName := reflect.Indirect(vof).Type().Name()
		if len(info.Package) > 0 {
			srvName = util.AddStr(info.Package, ".", srvName)
		}
		methods := []string{}
		for m := 0; m < tof.NumMethod(); m++ {
			method := tof.Method(m)
			methods = append(methods, method.Name)
		}
		if len(methods) == 0 {
			panic(util.AddStr("服务对象[", srvName, "]尚未有方法..."))
		}
		exist := false
		for _, v := range services {
			if v.Service == srvName && v.Address == addr {
				exist = true
				log.Println(util.AddStr("Consul服务[", v.Service, "][", v.Address, "]已存在,跳过注册"))
				break
			}
		}
		if exist {
			continue
		}
		registration.ID = util.GetUUID()
		registration.Name = srvName
		registration.Tags = info.Tags
		registration.Address = addr
		registration.Port = self.Config.RpcPort
		registration.Meta = make(map[string]string)
		registration.Check = &consulapi.AgentServiceCheck{HTTP: fmt.Sprintf("http://%s:%d%s", registration.Address, self.Config.CheckPort, "/check"), Timeout: self.Config.Timeout, Interval: self.Config.Interval, DeregisterCriticalServiceAfter: self.Config.DestroyAfter,}
		// 启动RPC服务
		log.Println(util.AddStr("Consul服务[", registration.Name, "][", registration.Address, "]注册成功"))
		if err := self.Consulx.Agent().ServiceRegister(registration); err != nil {
			panic(util.AddStr("Consul注册[", srvName, "]服务失败: ", err.Error()))
		}
		rpc.Register(info.Iface)
	}
}

// 获取RPC服务,并执行访问 args参数不可变,reply参数可变
func (self *ConsulManager) CallRPC(callInfo *CallInfo) error {
	if callInfo.Service == "" {
		return errors.New("服务名称为空")
	}
	if callInfo.Method == "" {
		return errors.New("方法名称为空")
	}
	if callInfo.Request == nil {
		return errors.New("请求对象为空")
	}
	if callInfo.Response == nil {
		return errors.New("响应对象为空")
	}
	if len(callInfo.Protocol) == 0 {
		callInfo.Protocol = "tcp"
	}
	if callInfo.Timeout == 0 {
		callInfo.Timeout = 10
	}
	serviceName := callInfo.Service
	if len(callInfo.Package) > 0 {
		serviceName = util.AddStr(callInfo.Package, ".", callInfo.Service)
	}
	services, err := self.GetService(serviceName)
	if err != nil {
		return util.Error("读取[", serviceName, "]服务失败: ", err)
	}
	if len(services) == 0 {
		return util.Error("没有找到可用[", serviceName, "]服务")
	}
	var service *consulapi.AgentService
	if self.Selection == nil { // 选取规则为空则默认随机
		r := rand.New(rand.NewSource(util.GetUUIDInt64()))
		service = services[r.Intn(len(services))]
	} else {
		service = self.Selection(services)
	}
	monitor := MonitorLog{
		ConsulHost:  self.Config.Host,
		ServiceName: serviceName,
		MethodName:  callInfo.Method,
		RpcPort:     service.Port,
		RpcHost:     service.Address,
		AgentID:     service.ID,
		BeginTime:   util.Time(),
	}
	defer self.rpcMonitor(monitor, err, callInfo.Request, callInfo.Response)
	var conn net.Conn
	var err1, err2 error
	conn, err = net.DialTimeout(callInfo.Protocol, util.AddStr(service.Address, ":", self.Config.ListenProt), time.Second*time.Duration(callInfo.Timeout))
	if err != nil {
		log.Error("consul服务连接失败", 0, log.String("ID", service.ID), log.String("host", service.Address), log.String("srv", service.Service), log.AddError(err))
		return util.Error("[", service.Address, "]", "[", service.Service, "]连接失败: ", err)
	}
	encBuf := bufio.NewWriter(conn)
	codec := &gobClientCodec{conn, gob.NewDecoder(conn), gob.NewEncoder(encBuf), encBuf}
	cli := rpc.NewClientWithCodec(codec)
	err1 = cli.Call(util.AddStr(callInfo.Service, ".", callInfo.Method), callInfo.Request, callInfo.Response)
	err2 = cli.Close()
	if err1 != nil {
		log.Error("consul服务访问失败", 0, log.String("ID", service.ID), log.String("host", service.Address), log.String("srv", service.Service), log.AddError(err1))
		return util.Error("[", service.Address, "]", "[", service.Service, "]访问失败: ", err1)
	} else if err2 != nil {
		log.Error("consul服务关闭失败", 0, log.String("ID", service.ID), log.String("host", service.Address), log.String("srv", service.Service), log.AddError(err2))
		return util.Error("[", service.Address, "]", "[", service.Service, "]关闭失败: ", err2)
	}
	return nil
}

// 输出RPC监控日志
func (self *ConsulManager) rpcMonitor(monitor MonitorLog, err error, args interface{}, reply interface{}) error {
	monitor.CostTime = util.Time() - monitor.BeginTime
	if err != nil {
		monitor.Error = err
		log.Println(util.ObjectToJson(monitor))
		return nil
	}
	if self.Config.SlowQuery > 0 && monitor.CostTime > self.Config.SlowQuery {
		l := self.getSlowLog()
		if l != nil {
			l.Warn("consul monitor", log.Int64("cost", monitor.CostTime), log.Any("service", monitor), log.Any("request", args), log.Any("response", reply))
		}
	}
	return err
}

// 接口服务健康检查
func (self *ConsulManager) healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "consulCheck")
}
