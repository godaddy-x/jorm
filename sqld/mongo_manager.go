package sqld

import (
	"github.com/godaddy-x/jorm/cache"
	"github.com/godaddy-x/jorm/log"
	"github.com/godaddy-x/jorm/sqlc"
	"github.com/godaddy-x/jorm/util"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"time"
)

var (
	mgo_sessions = make(map[string]*MGOManager)
	mgo_slowlog  *zap.Logger
)

type CountResult struct {
	Total int64 `bson:"COUNT_BY"`
}

const (
	COUNT_BY = "COUNT_BY"
	MASTER   = "MASTER"
)

/********************************** 数据库配置参数 **********************************/

// 数据库配置
type MGOConfig struct {
	DBConfig
	Addrs     []string
	Direct    bool
	Timeout   int64
	Database  string
	Username  string
	Password  string
	PoolLimit int
}

// 数据库管理器
type MGOManager struct {
	DBManager
	Session *mgo.Session
}

func (self *MGOManager) Get(option ...Option) (*MGOManager, error) {
	if option != nil && len(option) > 0 {
		if err := self.GetDB(option[0]); err != nil {
			return nil, err
		}
		return self, nil
	} else {
		if err := self.GetDB(); err != nil {
			return nil, err
		}
		return self, nil
	}
}

// 获取mongo的数据库连接
func (self *MGOManager) GetDatabase(copySession *mgo.Session, data interface{}) (*mgo.Collection, error) {
	tb, err := util.GetDbAndTb(data)
	if err != nil {
		return nil, err
	}
	database := copySession.DB("")
	if database == nil {
		return nil, self.Error("获取mongo数据库失败")
	}
	collection := database.C(tb)
	if collection == nil {
		return nil, self.Error("获取mongo数据集合失败")
	}
	return collection, nil
}

func (self *MGOManager) GetDB(option ...Option) error {
	var ds string
	if option != nil && len(option) > 0 {
		ops := option[0]
		ds = ops.DsName
		self.Option = ops
	}
	if len(ds) == 0 {
		ds = MASTER
	}
	if len(mgo_sessions) > 0 {
		manager := mgo_sessions[ds]
		if manager == nil {
			return self.Error(util.AddStr("mongo数据源[", ds, "]未找到,请检查..."))
		}
		self.Debug = manager.Debug
		self.SlowQuery = manager.SlowQuery
		self.SlowLogPath = manager.SlowLogPath
		self.Session = manager.Session
		self.CacheManager = manager.CacheManager
	} else {
		self.CacheSync = false
		log.Println("mongo session is nil")
	}
	return nil
}

func (self *MGOManager) InitConfig(input ...MGOConfig) error {
	return self.buildByConfig(nil, input...)
}

func (self *MGOManager) InitConfigAndCache(manager cache.ICache, input ...MGOConfig) error {
	return self.buildByConfig(manager, input...)
}

func (self *MGOManager) buildByConfig(manager cache.ICache, input ...MGOConfig) error {
	for _, conf := range input {
		dialInfo := mgo.DialInfo{
			Addrs:     conf.Addrs,
			Direct:    conf.Direct,
			Timeout:   time.Second * time.Duration(conf.Timeout),
			Database:  conf.Database,
			PoolLimit: conf.PoolLimit,
		}
		if len(conf.Username) > 0 {
			dialInfo.Username = conf.Username
		}
		if len(conf.Password) > 0 {
			dialInfo.Password = conf.Password
		}
		session, err := mgo.DialWithInfo(&dialInfo)
		if err != nil {
			panic("mongo连接初始化失败: " + err.Error())
		}
		session.SetSocketTimeout(3 * time.Minute)
		session.SetMode(mgo.Monotonic, true)
		session.SetSyncTimeout(0)
		if len(conf.DsName) == 0 {
			self.DsName = MASTER
		} else {
			self.DsName = conf.DsName
		}
		if err != nil {
			panic("redis数据源[" + self.DsName + "]类型异常失败")
		}
		dbmgr := DBManager{}
		dbmgr.CacheManager = manager
		dbmgr.Debug = conf.Debug
		dbmgr.SlowQuery = conf.SlowQuery
		dbmgr.SlowLogPath = conf.SlowLogPath
		mgomgr := &MGOManager{DBManager: dbmgr, Session: session}
		mgomgr.initSlowLog()
		mgo_sessions[self.DsName] = mgomgr
	}
	if len(mgo_sessions) == 0 {
		panic("mongo连接初始化失败: 数据源为0")
	}
	return nil
}

func (self *MGOManager) initSlowLog() {
	if self.SlowQuery == 0 || len(self.SlowLogPath) == 0 {
		return
	}
	if mgo_slowlog == nil {
		mgo_slowlog = log.InitNewLog(&log.ZapConfig{
			Level:   "warn",
			Console: false,
			FileConfig: &log.FileConfig{
				Compress:   true,
				Filename:   self.SlowLogPath,
				MaxAge:     7,
				MaxBackups: 7,
				MaxSize:    512,
			}})
		log.Println("MGO查询监控日志服务启动成功...")
	}
}

func (self *MGOManager) getSlowLog() *zap.Logger {
	return mgo_slowlog
}

// 保存或更新数据到mongo集合
func (self *MGOManager) Save(datas ...interface{}) error {
	if datas == nil || len(datas) == 0 {
		return self.Error("参数列表不能为空")
	}
	start := util.Time()
	defer self.debug("Save/Update", &datas, start)
	copySession := self.Session.Copy()
	defer copySession.Close()
	var db *mgo.Collection
	var err error
	saveObjs := make([]interface{}, 0, len(datas))
	for _, data := range datas {
		if data == nil {
			return self.Error("参数元素不能为空")
		}
		if reflect.ValueOf(data).Kind() != reflect.Ptr {
			return self.Error("参数值必须为指针类型")
		}
		if db == nil {
			db, err = self.GetDatabase(copySession, data)
			if err != nil {
				return self.Error(err)
			}
		}
		objectId := util.GetDataID(data)
		if objectId == 0 {
			objectId = util.GetUUIDInt64()
			v := reflect.ValueOf(data).Elem()
			v.FieldByName("Id").Set(reflect.ValueOf(objectId))
		}
		pipe, err := self.buildPipeCondition(sqlc.M(nil).Eq("_id", objectId), true)
		if err != nil {
			return self.Error(util.AddStr("mongo构建查询命令失败: ", err.Error()))
		}
		result := CountResult{}
		if err := db.Pipe(pipe).One(&result); err != nil {
			if err != mgo.ErrNotFound {
				return self.Error(util.Error("[Mongo.Count]查询数据失败: ", err))
			}
		}
		if result.Total == 0 {
			saveObjs = append(saveObjs, data)
			continue
		}
		if err := db.UpdateId(objectId, data); err != nil {
			return self.Error(util.AddStr("mongo更新数据失败: ", err))
		}
	}
	if len(saveObjs) > 0 {
		if err := db.Insert(saveObjs ...); err != nil {
			return self.Error(util.AddStr("mongo保存数据失败: ", err))
		}
	}
	return nil
}

// 保存或更新数据到mongo集合
func (self *MGOManager) Update(datas ...interface{}) error {
	return self.Save(datas...)
}

func (self *MGOManager) Delete(datas ...interface{}) error {
	if datas == nil || len(datas) == 0 {
		return self.Error("参数列表不能为空")
	}
	start := util.Time()
	defer self.debug("Delete", &datas, start)
	copySession := self.Session.Copy()
	defer copySession.Close()
	var db *mgo.Collection
	var err error
	delIds := make([]interface{}, 0, len(datas))
	for _, data := range datas {
		if data == nil {
			return self.Error("参数元素不能为空")
		}
		if reflect.ValueOf(data).Kind() != reflect.Ptr {
			return self.Error("参数值必须为指针类型")
		}
		if db == nil {
			db, err = self.GetDatabase(copySession, data)
			if err != nil {
				return self.Error(err)
			}
		}
		objectId := util.GetDataID(data)
		if objectId == 0 {
			continue
		}
		delIds = append(delIds, objectId)
	}
	if len(delIds) > 0 {
		if _, err := db.RemoveAll(bson.M{"_id": bson.M{"$in": delIds}}); err != nil {
			return self.Error(util.AddStr("删除数据ID失败", err))
		}
	}
	return nil
}

// 统计数据
func (self *MGOManager) Count(cnd *sqlc.Cnd) (int64, error) {
	start := util.Time()
	if cnd.Model == nil {
		return 0, self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	var ok bool
	var pageTotal int64
	if isc, hasv, err := self.getByCache(cnd, &pageTotal); err != nil {
		return 0, err
	} else if isc && hasv {
		ok = isc
		defer self.debug("Count by Cache", make([]interface{}, 0), start)
	} else if isc && !hasv {
		defer self.putByCache(cnd, &pageTotal)
	}
	if !ok {
		copySession := self.Session.Copy()
		defer copySession.Close()
		db, err := self.GetDatabase(copySession, cnd.Model)
		if err != nil {
			return 0, self.Error(err)
		}
		pipe, err := self.buildPipeCondition(cnd, true)
		if err != nil {
			return 0, self.Error(util.AddStr("mongo构建查询命令失败: ", err.Error()))
		}
		defer self.debug("Count", pipe, start)
		result := CountResult{}
		err = db.Pipe(pipe).One(&result)
		if err != nil {
			if err == mgo.ErrNotFound {
				return 0, nil
			}
			return 0, self.Error(util.AddStr("mongo查询数据失败: ", err.Error()))
		}
		pageTotal = result.Total
	}
	if pageTotal > 0 && cnd.Pagination.PageSize > 0 {
		var pageCount int64
		if pageTotal%cnd.Pagination.PageSize == 0 {
			pageCount = pageTotal / cnd.Pagination.PageSize
		} else {
			pageCount = pageTotal/cnd.Pagination.PageSize + 1
		}
		cnd.Pagination.PageCount = pageCount
	} else {
		cnd.Pagination.PageCount = 0
	}
	cnd.Pagination.PageTotal = pageTotal
	return pageTotal, nil
}

// 查询单条数据
func (self *MGOManager) FindOne(cnd *sqlc.Cnd, data interface{}) error {
	start := util.Time()
	var elem = cnd.Model
	if elem == nil {
		return self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	tof := util.TypeOf(elem)
	if tof.Kind() != reflect.Struct && tof.Kind() != reflect.Ptr {
		return self.Error("ORM对象类型必须为struct或ptr")
	}
	if data == nil {
		return self.Error("返回值不能为空")
	}
	if reflect.TypeOf(data).Kind() != reflect.Ptr {
		return self.Error("返回值必须为指针类型")
	}
	if util.TypeOf(data).Kind() != reflect.Struct {
		return self.Error("返回结果必须为struct类型")
	}
	if isc, hasv, err := self.getByCache(cnd, data); err != nil {
		return err
	} else if isc && hasv {
		defer self.debug("FindOne by Cache", make([]interface{}, 0), start)
		return nil
	} else if isc && !hasv {
		defer self.putByCache(cnd, data)
	}
	copySession := self.Session.Copy()
	defer copySession.Close()
	db, err := self.GetDatabase(copySession, elem)
	if err != nil {
		return self.Error(err)
	}
	pipe, err := self.buildPipeCondition(cnd, false)
	if err != nil {
		return self.Error(util.AddStr("mongo构建查询命令失败: ", err.Error()))
	}
	defer self.debug("FindOne", pipe, start)
	if len(cnd.Summaries) > 0 {
		hasId := false
		for k, _ := range cnd.Summaries {
			if k == "_id" {
				hasId = true
				break
			}
		}
		if hasId {
			result := map[string]interface{}{}
			err = db.Pipe(pipe).One(&result)
			if err != nil {
				if err != mgo.ErrNotFound {
					return self.Error(util.AddStr("mongo查询数据失败: ", err.Error()))
				}
			}
			idv, _ := result["id"]
			result["_id"] = idv
			if err := util.JsonToAny(&result, data); err != nil {
				return self.Error(util.AddStr("mongo查询数据转换失败: ", err.Error()))
			}
			return nil
		}
	}
	err = db.Pipe(pipe).One(data)
	if err != nil {
		if err != mgo.ErrNotFound {
			return self.Error(util.AddStr("mongo查询数据失败: ", err.Error()))
		}
	}
	return nil
}

// 查询多条数据
func (self *MGOManager) FindList(cnd *sqlc.Cnd, data interface{}) error {
	start := util.Time()
	var elem = cnd.Model
	if elem == nil {
		return self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	tof := util.TypeOf(elem)
	if tof.Kind() != reflect.Struct && tof.Kind() != reflect.Ptr {
		return self.Error("ORM对象类型必须为struct或ptr")
	}
	if data == nil {
		return self.Error("返回值不能为空")
	}
	if reflect.TypeOf(data).Kind() != reflect.Ptr {
		return self.Error("返回值必须为指针类型")
	}
	if util.TypeOf(data).Kind() != reflect.Slice {
		return self.Error("返回结果必须为数组类型")
	}
	if isc, hasv, err := self.getByCache(cnd, data); err != nil {
		return err
	} else if isc && hasv {
		defer self.debug("FindList by Cache", make([]interface{}, 0), start)
		return nil
	} else if isc && !hasv {
		defer self.putByCache(cnd, data)
	}
	copySession := self.Session.Copy()
	defer copySession.Close()
	db, err := self.GetDatabase(copySession, elem)
	if err != nil {
		return self.Error(err)
	}
	pipe, err := self.buildPipeCondition(cnd, false)
	if err != nil {
		return self.Error(util.AddStr("mongo构建查询命令失败: ", err.Error()))
	}
	defer self.debug("FindList", pipe, start)
	err = db.Pipe(pipe).All(data)
	if err != nil {
		if err != mgo.ErrNotFound {
			return self.Error(util.AddStr("mongo查询数据失败: ", err.Error()))
		}
	}
	return nil
}

// 根据条件更新数据
func (self *MGOManager) UpdateByCnd(cnd *sqlc.Cnd) error {
	start := util.Time()
	if cnd.Model == nil {
		return self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	copySession := self.Session.Copy()
	defer copySession.Close()
	db, err := self.GetDatabase(copySession, cnd.Model)
	if err != nil {
		return self.Error(err)
	}
	match := buildMongoMatch(cnd)
	upset := buildMongoUpset(cnd)
	if len(upset) == 0 {
		return util.Error("更新条件不能为空")
	}
	a := bson.M{}
	b := bson.M{}
	if err := util.JsonToAny(&match, &a); err != nil {
		return err
	}
	if err := util.JsonToAny(&upset, &b); err != nil {
		return err
	}
	defer self.debug("UpdateByCnd", map[string]interface{}{"match": match, "upset": upset}, start)
	_, err = db.UpdateAll(a, b)
	if err != nil {
		return self.Error(util.AddStr("mongo按条件数据失败: ", err.Error()))
	}
	return nil
}

func (self *MGOManager) Close() error {
	// self.Session.Close()
	return nil
}

// 获取缓存结果集
func (self *MGOManager) getByCache(cnd *sqlc.Cnd, data interface{}) (bool, bool, error) {
	config := cnd.CacheConfig
	if config.Open && len(config.Key) > 0 {
		if self.CacheManager == nil {
			return true, false, self.Error("缓存管理器尚未初始化")
		}
		b, err := self.CacheManager.Get(config.Prefix+config.Key, data);
		return true, b, self.Error(err)
	}
	return false, false, nil
}

// 缓存结果集
func (self *MGOManager) putByCache(cnd *sqlc.Cnd, data interface{}) error {
	config := cnd.CacheConfig
	if config.Open && len(config.Key) > 0 {
		if err := self.CacheManager.Put(config.Prefix+config.Key, data, config.Expire); err != nil {
			return self.Error(err)
		}
	}
	return nil
}

// 获取最终pipe条件集合,包含$match $project $sort $skip $limit,未实现$group
func (self *MGOManager) buildPipeCondition(cnd *sqlc.Cnd, iscount bool) ([]interface{}, error) {
	match := buildMongoMatch(cnd)
	project := buildMongoProject(cnd)
	sortby := buildMongoSortBy(cnd)
	aggregate := buildSummary(cnd)
	pageinfo := buildMongoLimit(cnd)
	pipe := make([]interface{}, 0)
	if len(match) > 0 {
		tmp := make(map[string]interface{})
		tmp["$match"] = match
		pipe = append(pipe, tmp)
	}
	if len(project) > 0 {
		tmp := make(map[string]interface{})
		tmp["$project"] = project
		pipe = append(pipe, tmp)
	}
	if len(sortby) > 0 {
		tmp := make(map[string]interface{})
		tmp["$sort"] = sortby
		pipe = append(pipe, tmp)
	}
	if len(aggregate) > 0 {
		pipe = append(pipe, aggregate)
	}
	if !iscount && pageinfo != nil {
		tmp := make(map[string]interface{})
		tmp["$skip"] = pageinfo[0]
		pipe = append(pipe, tmp)
		tmp = make(map[string]interface{})
		tmp["$limit"] = pageinfo[1]
		pipe = append(pipe, tmp)
		if !cnd.CacheConfig.Open && !cnd.Pagination.IsOffset {
			pageTotal, err := self.Count(cnd)
			if err != nil {
				return nil, err
			}
			var pageCount int64
			if pageTotal%cnd.Pagination.PageSize == 0 {
				pageCount = pageTotal / cnd.Pagination.PageSize
			} else {
				pageCount = pageTotal/cnd.Pagination.PageSize + 1
			}
			cnd.Pagination.PageTotal = pageTotal
			cnd.Pagination.PageCount = pageCount
		}
	}
	if iscount {
		tmp := make(map[string]interface{})
		tmp["$count"] = COUNT_BY
		pipe = append(pipe, tmp)
	}
	return pipe, nil
}

// 构建mongo逻辑条件命令
func buildMongoMatch(cnd *sqlc.Cnd) map[string]interface{} {
	var query = make(map[string]interface{})
	condits := cnd.Conditions
	for e := range condits {
		condit := condits[e]
		key := condit.Key
		value := condit.Value
		values := condit.Values
		switch condit.Logic {
		// case condition
		case sqlc.EQ_:
			query[key] = value
		case sqlc.NOT_EQ_:
			tmp := make(map[string]interface{})
			tmp["$ne"] = value
			query[key] = tmp
		case sqlc.LT_:
			tmp := make(map[string]interface{})
			tmp["$lt"] = value
			query[key] = tmp
		case sqlc.LTE_:
			tmp := make(map[string]interface{})
			tmp["$lte"] = value
			query[key] = tmp
		case sqlc.GT_:
			tmp := make(map[string]interface{})
			tmp["$gt"] = value
			query[key] = tmp
		case sqlc.GTE_:
			tmp := make(map[string]interface{})
			tmp["$gte"] = value
			query[key] = tmp
		case sqlc.IS_NULL_:
			query[key] = nil
		case sqlc.IS_NOT_NULL_:
			tmp := make(map[string]interface{})
			tmp["$ne"] = nil
			query[key] = tmp
		case sqlc.BETWEEN_:
			tmp := make(map[string]interface{})
			tmp["$gte"] = values[0]
			tmp["$lte"] = values[1]
			query[key] = tmp
		case sqlc.NOT_BETWEEN_:
			// unsupported
		case sqlc.IN_:
			tmp := make(map[string]interface{})
			tmp["$in"] = values
			query[key] = tmp
		case sqlc.NOT_IN_:
			tmp := make(map[string]interface{})
			tmp["$nin"] = values
			query[key] = tmp
		case sqlc.LIKE_:
			tmp := make(map[string]interface{})
			tmp["$regex"] = value
			query[key] = tmp
		case sqlc.NO_TLIKE_:
			// unsupported
		case sqlc.OR_:
			array := make([]interface{}, 0)
			for e := range values {
				cnd, ok := values[e].(*sqlc.Cnd)
				if !ok {
					continue
				}
				tmp := buildMongoMatch(cnd)
				array = append(array, tmp)
			}
			query["$or"] = array
		}
	}
	return query
}

// 构建mongo字段更新命令
func buildMongoUpset(cnd *sqlc.Cnd) map[string]interface{} {
	query := make(map[string]interface{})
	if len(cnd.UpdateKV) > 0 {
		query["$set"] = cnd.UpdateKV
	}
	return query
}

// 构建mongo字段筛选命令
func buildMongoProject(cnd *sqlc.Cnd) map[string]int {
	var project = make(map[string]int)
	anyFields := cnd.AnyFields
	for e := range anyFields {
		project[anyFields[e]] = 1
	}
	return project
}

// 构建mongo排序命令
func buildMongoSortBy(cnd *sqlc.Cnd) map[string]int {
	var sortby = make(map[string]int)
	orderbys := cnd.Orderbys
	for e := range orderbys {
		orderby := orderbys[e]
		if orderby.Value == sqlc.DESC_ {
			sortby[orderby.Key] = -1
		} else if orderby.Value == sqlc.ASC_ {
			sortby[orderby.Key] = 1
		}
	}
	return sortby
}

// 构建mongo聚合命令
func buildSummary(cnd *sqlc.Cnd) map[string]interface{} {
	var query = make(map[string]interface{})
	if len(cnd.Summaries) == 0 {
		return nil
	}
	tmp := make(map[string]interface{})
	tmp["_id"] = 0
	for k, v := range cnd.Summaries {
		key := k
		if key == "_id" {
			key = "id"
		}
		if v == sqlc.SUM_ {
			tmp[key] = map[string]interface{}{"$sum": util.AddStr("$", k)}
		} else if v == sqlc.MAX_ {
			tmp[key] = map[string]interface{}{"$max": util.AddStr("$", k)}
		} else if v == sqlc.MIN_ {
			tmp[key] = map[string]interface{}{"$min": util.AddStr("$", k)}
		} else if v == sqlc.AVG_ {
			tmp[key] = map[string]interface{}{"$avg": util.AddStr("$", k)}
		} else {
			return query
		}
	}
	query["$group"] = tmp
	return query
}

// 构建mongo分页命令
func buildMongoLimit(cnd *sqlc.Cnd) []int64 {
	pg := cnd.Pagination
	if pg.PageNo == 0 && pg.PageSize == 0 {
		return nil
	}
	if pg.PageSize <= 0 {
		pg.PageSize = 10
	}
	if pg.IsOffset {
		return []int64{pg.PageNo, pg.PageSize}
	} else {
		pageNo := pg.PageNo
		pageSize := pg.PageSize
		return []int64{(pageNo - 1) * pageSize, pageSize}
	}
	return nil
}

func (self *MGOManager) debug(title string, pipe interface{}, start int64) {
	cost := util.Time() - start
	if self.SlowQuery > 0 && cost > self.SlowQuery {
		if title == "Count" || title == "FindOne" || title == "FindList" {
			l := self.getSlowLog()
			if l != nil {
				l.Warn(title, log.Int64("cost", cost), log.Any("pipe", pipe))
			}
		}
	}
	if self.Debug {
		str, _ := util.ObjectToJson(pipe)
		log.Println(util.AddStr("mongo debug -> ", title, ": ", str, " --- cost: ", util.AnyToStr(util.Time()-start)))
	}
}
