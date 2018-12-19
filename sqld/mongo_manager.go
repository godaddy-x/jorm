package sqld

import (
	"github.com/godaddy-x/jorm/cache"
	"github.com/godaddy-x/jorm/sqlc"
	"github.com/godaddy-x/jorm/util"
	"gopkg.in/mgo.v2"
	"log"
	"reflect"
	"time"
)

var (
	mgo_sessions = make(map[string]*MGOManager)
)

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
		return nil, util.Error("获取mongo数据库失败")
	}
	collection := database.C(tb)
	if collection == nil {
		return nil, util.Error("获取mongo数据集合失败")
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
			return util.Error("mongo数据源[", ds, "]未找到,请检查...")
		}
		self.Debug = manager.Debug
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
	for e := range input {
		conf := input[e]
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
		if len(conf.DsName) == 0 {
			self.DsName = MASTER
		} else {
			self.DsName = conf.DsName
		}
		if err != nil {
			panic("redis数据源[" + self.DsName + "]类型异常失败")
		}
		mgo_sessions[self.DsName] = &MGOManager{DBManager: DBManager{Debug: conf.Debug, CacheManager: manager}, Session: session}
	}
	if len(mgo_sessions) == 0 {
		panic("mongo连接初始化失败: 数据源为0")
	}
	return nil
}

// 保存或更新数据到mongo集合
func (self *MGOManager) Save(datas ...interface{}) error {
	if datas == nil || len(datas) == 0 {
		return util.Error("参数不能为空")
	}
	for e := range datas {
		start := util.Time()
		data := datas[e]
		if data == nil {
			return util.Error("数据实体为空")
		}
		if reflect.ValueOf(data).Kind() != reflect.Ptr {
			return self.Error("参数值必须为指针类型")
		}
		objectId := util.GetDataID(data)
		if objectId == 0 {
			v := reflect.ValueOf(data).Elem()
			uuid, _ := util.StrToInt64(util.GetUUID())
			v.FieldByName("Id").Set(reflect.ValueOf(uuid))
		}
		copySession := self.Session.Copy()
		defer copySession.Close()
		db, err := self.GetDatabase(copySession, data)
		if err != nil {
			return err
		}
		defer self.debug("Save/Update", data, start)
		newObject := util.NewInstance(data)
		err = db.FindId(objectId).One(newObject)
		if err == nil { // 更新数据
			err = db.UpdateId(objectId, data)
			if err != nil {
				return util.Error("mongo更新数据失败: ", err.Error())
			}
			return nil
		} else { // 新增数据
			err = db.Insert(data)
			if err != nil {
				return util.Error("mongo保存数据失败: ", err.Error())
			}
			return nil
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
		return util.Error("参数不能为空")
	}
	for e := range datas {
		data := datas[e]
		start := util.Time()
		if data == nil {
			return util.Error("数据实体为空")
		}
		if reflect.ValueOf(data).Kind() != reflect.Ptr {
			return self.Error("参数值必须为指针类型")
		}
		copySession := self.Session.Copy()
		defer copySession.Close()
		db, err := self.GetDatabase(copySession, data)
		if err != nil {
			return err
		}
		defer self.debug("Delete", data, start)
		if err := db.RemoveId(util.GetDataID(data)); err != nil {
			return util.Error("删除数据ID失败")
		}
	}
	return nil
}

// 统计数据
func (self *MGOManager) Count(cnd *sqlc.Cnd) (int64, error) {
	start := util.Time()
	if cnd.Model == nil {
		return 0, util.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	var ok bool
	var pageTotal int64
	if isc, hasv, err := self.getByCache(cnd, &pageTotal); err != nil {
		return 0, err
	} else if isc && hasv {
		defer self.debug("FindOne by Cache", make([]interface{}, 0), start)
		ok = true
	} else if isc && !hasv {
		defer self.putByCache(cnd, &pageTotal)
	}
	if !ok {
		copySession := self.Session.Copy()
		defer copySession.Close()
		db, err := self.GetDatabase(copySession, cnd.Model)
		if err != nil {
			return 0, err
		}
		pipe, err := self.buildPipeCondition(cnd, true)
		if err != nil {
			return 0, util.Error("mongo构建查询命令失败: ", err.Error())
		}
		defer self.debug("Count", pipe, start)
		result := make(map[string]int64)
		err = db.Pipe(pipe).One(&result)
		if err != nil {
			if err.Error() == "not found" {
				return 0, nil
			}
			return 0, util.Error("mongo查询数据失败: ", err.Error())
		}
		pageTotal, ok = result[COUNT_BY]
		if !ok {
			return 0, util.Error("获取记录数失败")
		}
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
		return err
	}
	pipe, err := self.buildPipeCondition(cnd, false)
	if err != nil {
		return util.Error("mongo构建查询命令失败: ", err.Error())
	}
	defer self.debug("FindOne", pipe, start)
	err = db.Pipe(pipe).One(data)
	if err != nil {
		if err.Error() != "not found" {
			return util.Error("mongo查询数据失败: ", err.Error())
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
		return err
	}
	pipe, err := self.buildPipeCondition(cnd, false)
	if err != nil {
		return util.Error("mongo构建查询命令失败: ", err.Error())
	}
	defer self.debug("FindList", pipe, start)
	err = db.Pipe(pipe).All(data)
	if err != nil {
		if err.Error() != "not found" {
			return util.Error("mongo查询数据失败: ", err.Error())
		}
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
			return true, false, util.Error("缓存管理器尚未初始化")
		}
		b, err := self.CacheManager.Get(config.Prefix+config.Key, data);
		return true, b, err
	}
	return false, false, nil
}

// 缓存结果集
func (self *MGOManager) putByCache(cnd *sqlc.Cnd, data interface{}) error {
	config := cnd.CacheConfig
	if config.Open && len(config.Key) > 0 {
		if err := self.CacheManager.Put(config.Prefix+config.Key, data, config.Expire); err != nil {
			return err
		}
	}
	return nil
}

// 获取最终pipe条件集合,包含$match $project $sort $skip $limit,未实现$group
func (self *MGOManager) buildPipeCondition(cnd *sqlc.Cnd, iscount bool) ([]interface{}, error) {
	match := buildMongoMatch(cnd)
	project := buildMongoProject(cnd)
	sortby := buildMongoSortBy(cnd)
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
	if self.Debug {
		str, _ := util.ObjectToJson(pipe)
		log.Println(util.AddStr("mongo debug -> ", title, ": ", str, " --- cost: ", util.AnyToStr(util.Time()-start)))
	}
}
