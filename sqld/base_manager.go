package sqld

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/godaddy-x/jorm/cache"
	"github.com/godaddy-x/jorm/dialect"
	"github.com/godaddy-x/jorm/sqlc"
	"github.com/godaddy-x/jorm/util"
	"log"
	"reflect"
)

var (
	RDBs = map[string]*RDBManager{}
)

/********************************** 数据库配置参数 **********************************/

// 数据库配置
type DBConfig struct {
	Host      string // 地址IP
	Port      int    // 数据库端口
	Database  string // 数据库名称
	Username  string // 账号
	Password  string // 密码
	Debug     bool   // debug模式
	CacheSync bool   // 是否缓存数据
	DsName    string // 数据源名称
	Node      int    // 节点
	AutoID    bool   // 自主ID模式
}

// 数据选项
type Option struct {
	Node         int          // 节点
	AutoID       bool         // 自主ID模式
	AutoTx       bool         // 是否自动事务提交 false.否 true.是
	DsName       string       // 数据源,分库时使用
	CacheSync    bool         // 是否数据缓存,比如redis,mongo等
	CacheManager cache.ICache // 缓存管理器
}

// 数据库管理器
type DBManager struct {
	Option
	Debug        bool          // debug模式
	CacheManager cache.ICache  // 缓存管理器
	CacheObject  []interface{} // 需要缓存的数据 CacheSync为true时有效
	Errors       []error       // 错误异常记录
}

/********************************** 数据库ORM实现 **********************************/

// orm数据库接口
type IDBase interface {
	// 初始化数据库配置
	InitConfig(input interface{}) error
	// 获取数据库管理器
	GetDB(option ...Option) error
	// 保存数据
	Save(datas ...interface{}) error
	// 更新数据
	Update(datas ...interface{}) error
	// 删除数据
	Delete(datas ...interface{}) error
	// 统计数据
	Count(cnd *sqlc.Cnd) (int64, error)
	// 按ID查询单条数据
	FindById(data interface{}) error
	// 按条件查询单条数据
	FindOne(cnd *sqlc.Cnd, data interface{}) error
	// 按条件查询数据
	FindList(cnd *sqlc.Cnd, data interface{}) error
	// 按复杂条件查询数据
	FindComplex(cnd *sqlc.Cnd, data interface{}) error
	// 构建数据表别名
	BuildCondKey(cnd *sqlc.Cnd, key string) string
	// 构建逻辑条件
	BuildWhereCase(cnd *sqlc.Cnd) (bytes.Buffer, []interface{})
	// 构建分组条件
	BuilGroupBy(cnd *sqlc.Cnd) string
	// 构建排序条件
	BuilSortBy(cnd *sqlc.Cnd) string
	// 构建分页条件
	BuildPagination(cnd *sqlc.Cnd, sql string, values []interface{}) (string, error)
	// 数据库操作缓存异常
	Error(data interface{}) error
}

func (self *DBManager) InitConfig(input interface{}) error {
	return util.Error("No implementation method [InitConfig] was found")
}

func (self *DBManager) GetDB(option ...Option) error {
	return util.Error("No implementation method [GetDB] was found")
}

func (self *DBManager) Save(datas ...interface{}) error {
	return util.Error("No implementation method [Save] was found")
}

func (self *DBManager) Update(datas ...interface{}) error {
	return util.Error("No implementation method [Update] was found")
}

func (self *DBManager) Delete(datas ...interface{}) error {
	return util.Error("No implementation method [Delete] was found")
}

func (self *DBManager) Count(cnd *sqlc.Cnd) (int64, error) {
	return 0, util.Error("No implementation method [Count] was found")
}

func (self *DBManager) FindById(data interface{}) error {
	return util.Error("No implementation method [FindById] was found")
}

func (self *DBManager) FindOne(cnd *sqlc.Cnd, data interface{}) error {
	return util.Error("No implementation method [FindOne] was found")
}

func (self *DBManager) FindList(cnd *sqlc.Cnd, data interface{}) error {
	return util.Error("No implementation method [FindList] was found")
}

func (self *DBManager) FindComplex(cnd *sqlc.Cnd, data interface{}) error {
	return util.Error("No implementation method [FindComplex] was found")
}

func (self *DBManager) Close() error {
	return util.Error("No implementation method [Close] was found")
}

func (self *DBManager) BuildCondKey(cnd *sqlc.Cnd, key string) string {
	log.Println("No implementation method [BuildCondKey] was found")
	return ""
}

func (self *DBManager) BuildWhereCase(cnd *sqlc.Cnd) (bytes.Buffer, []interface{}) {
	log.Println("No implementation method [BuildWhereCase] was found")
	var b bytes.Buffer
	return b, nil
}

func (self *DBManager) BuilGroupBy(cnd *sqlc.Cnd) string {
	log.Println("No implementation method [BuilGroupBy] was found")
	return ""
}

func (self *DBManager) BuilSortBy(cnd *sqlc.Cnd) string {
	log.Println("No implementation method [BuilSortBy] was found")
	return ""
}

func (self *DBManager) BuildPagination(cnd *sqlc.Cnd, sql string, values []interface{}) (string, error) {
	return "", util.Error("No implementation method [BuildPagination] was found")
}

func (self *DBManager) Error(data interface{}) error {
	if err, ok := data.(error); ok {
		self.Errors = append(self.Errors, err)
		return err
	} else if err, ok := data.(string); ok {
		err := util.Error(err)
		self.Errors = append(self.Errors, err)
		return err
	}
	return nil
}

/********************************** 关系数据库ORM默认实现 -> MySQL(如需实现其他类型数据库则自行实现IDBase接口) **********************************/

// 关系数据库连接管理器
type RDBManager struct {
	DBManager
	Db *sql.DB
	Tx *sql.Tx
}

func (self *RDBManager) GetDB(option ...Option) error {
	var ds string
	if option != nil && len(option) > 0 {
		ds = option[0].DsName
	}
	if len(ds) == 0 {
		ds = MASTER
	}
	rdb := RDBs[ds]
	if rdb == nil {
		return util.Error("SQL数据源[", ds, "]未找到,请检查...")
	}
	self.Db = rdb.Db
	self.Debug = rdb.Debug
	self.CacheSync = rdb.CacheSync
	self.CacheManager = rdb.CacheManager
	if option != nil && len(option) > 0 {
		ops := option[0]
		self.CacheSync = ops.CacheSync
		if ops.AutoTx {
			if tx, err := self.Db.Begin(); err != nil {
				return util.Error("数据库开启事务失败: ", err.Error())
			} else {
				self.AutoTx = ops.AutoTx
				self.Tx = tx
			}
		}
		ops.DsName = ds
		self.Option = ops
	}
	return nil
}

func (self *RDBManager) Save(datas ...interface{}) error {
	if datas == nil || len(datas) == 0 {
		return util.Error("参数不能为空")
	}
	var stmt *sql.Stmt
	var svsql string
	for e := range datas {
		data := datas[e]
		start := util.Time()
		if data == nil {
			return self.Error("参数不能为空")
		}
		if reflect.ValueOf(data).Kind() != reflect.Ptr {
			return self.Error("参数值必须为指针类型")
		}
		var fieldPart1, fieldPart2 bytes.Buffer
		var valuePart = make([]interface{}, 0)
		var idValue reflect.Value
		tof := reflect.TypeOf(data).Elem()
		vof := reflect.ValueOf(data).Elem()
		for i := 0; i < tof.NumField(); i++ {
			field := tof.Field(i)
			value := vof.Field(i)
			if util.ValidIgnore(field) {
				continue
			}
			if field.Name == sqlc.Id {
				if self.AutoID {
					fieldPart1.WriteString(field.Tag.Get(sqlc.Json))
					fieldPart1.WriteString(",")
					fieldPart2.WriteString("?,")
					if valueID, err := util.StrToInt64(util.GetUUID(int64(self.Node))); err != nil {
						return err
					} else {
						valuePart = append(valuePart, valueID)
						value.SetInt(valueID)
					}
				} else {
					idValue = value
				}
				continue
			}
			kind := value.Kind()
			if kind == reflect.String {
				valuePart = append(valuePart, value.String())
			} else if kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 {
				rt := value.Int()
				if kind == reflect.Int64 && rt > 0 && util.ValidDate(field) {
					valuePart = append(valuePart, util.Time2Str(rt))
				} else {
					valuePart = append(valuePart, rt)
				}
			} else if !value.IsNil() && kind == reflect.Slice {
				if str, err := util.ObjectToJson(value.Interface()); err != nil {
					return self.Error(util.AddStr("字段[", field.Name, "]转换失败: ", err.Error()))
				} else {
					valuePart = append(valuePart, str)
				}
			} else if !value.IsNil() && kind == reflect.Map {
				if str, err := util.ObjectToJson(value.Interface()); err != nil {
					return self.Error(util.AddStr("字段[", field.Name, "]转换失败: ", err.Error()))
				} else {
					valuePart = append(valuePart, str)
				}
			} else if !value.IsNil() && kind == reflect.Slice {
				if str, err := util.ObjectToJson(value.Interface()); err != nil {
					return self.Error(util.AddStr("字段[", field.Name, "]转换失败: ", err.Error()))
				} else {
					valuePart = append(valuePart, str)
				}
			} else if value.IsNil() {
				continue
			} else {
				str, err := util.ObjectToJson(value.Interface());
				if err != nil {
					fmt.Println("字段输出json失败: " + value.String())
				}
				fmt.Println(util.AddStr("警告: 不支持的字段[", field.Name, "]类型[", kind.String(), "] --- ", str))
				continue
			}
			fieldPart1.WriteString(field.Tag.Get(sqlc.Bson))
			fieldPart1.WriteString(",")
			fieldPart2.WriteString("?,")
		}
		s1 := fieldPart1.String()
		s2 := fieldPart2.String()
		var sqlbuf bytes.Buffer
		sqlbuf.WriteString("insert into ")
		if tb, err := util.GetDbAndTb(data); err != nil {
			return self.Error(err)
		} else {
			sqlbuf.WriteString(tb)
		}
		sqlbuf.WriteString(" (")
		sqlbuf.WriteString(util.Substr(s1, 0, len(s1)-1))
		sqlbuf.WriteString(")")
		sqlbuf.WriteString(" values (")
		sqlbuf.WriteString(util.Substr(s2, 0, len(s2)-1))
		sqlbuf.WriteString(")")
		if len(svsql) == 0 {
			svsql = sqlbuf.String()
			var err error
			defer self.debug("Save", svsql, valuePart, start)
			if self.AutoTx {
				stmt, err = self.Tx.Prepare(svsql)
			} else {
				stmt, err = self.Db.Prepare(svsql)
			}
			if err != nil {
				return self.Error(util.AddStr("预编译sql[", svsql, "]失败: ", err.Error()))
			}
			defer stmt.Close()
		}
		ret, err := stmt.Exec(valuePart...)
		if err != nil {
			return self.Error(util.AddStr("保存数据失败: ", err.Error()))
		}
		if rowsAffected, err := ret.RowsAffected(); err != nil {
			return self.Error(util.AddStr("保存数据失败: ", err.Error()))
		} else if rowsAffected <= 0 {
			return self.Error(util.AddStr("保存数据失败: 受影响行数 -> ", util.AnyToStr(rowsAffected)))
		}
		if !self.AutoID {
			if lastInsertId, err := ret.LastInsertId(); err != nil {
				return self.Error(util.AddStr("保存数据失败: ", err.Error()))
			} else {
				if lastInsertId > 0 {
					idValue.SetInt(lastInsertId)
				}
			}
		}
	}
	return self.AddCacheSync(datas...)
}

func (self *RDBManager) Update(datas ...interface{}) error {
	if datas == nil || len(datas) == 0 {
		return util.Error("参数不能为空")
	}
	for e := range datas {
		data := datas[e]
		start := util.Time()
		if data == nil {
			return self.Error("参数不能为空")
		}
		if reflect.ValueOf(data).Kind() != reflect.Ptr {
			return self.Error("参数值必须为指针类型")
		}
		var fieldPart1, fieldPart2 bytes.Buffer
		var valuePart = make([]interface{}, 0)
		var idValue reflect.Value
		tof := reflect.TypeOf(data).Elem()
		vof := reflect.ValueOf(data).Elem()
		for i := 0; i < tof.NumField(); i++ {
			field := tof.Field(i)
			value := vof.Field(i)
			if util.ValidIgnore(field) {
				continue
			}
			if field.Name == sqlc.Id {
				if value.Kind() != reflect.Int64 {
					return self.Error("实体ID必须为int64类型")
				}
				idValue = value
				fieldPart2.WriteString("id = ?,")
				continue
			}
			kind := value.Kind()
			if kind == reflect.String {
				valuePart = append(valuePart, value.String())
			} else if kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 {
				rt := value.Int()
				if kind == reflect.Int64 && rt > 0 && util.ValidDate(field) {
					valuePart = append(valuePart, util.Time2Str(rt))
				} else {
					valuePart = append(valuePart, rt)
				}
			} else if !value.IsNil() && kind == reflect.Slice {
				if str, err := util.ObjectToJson(value.Interface()); err != nil {
					return self.Error(util.AddStr("字段[", field.Name, "]转换失败: ", err.Error()))
				} else {
					valuePart = append(valuePart, str)
				}
			} else if !value.IsNil() && kind == reflect.Map {
				if str, err := util.ObjectToJson(value.Interface()); err != nil {
					return self.Error(util.AddStr("字段[", field.Name, "]转换失败: ", err.Error()))
				} else {
					valuePart = append(valuePart, str)
				}
			} else if !value.IsNil() && kind == reflect.Slice {
				if str, err := util.ObjectToJson(value.Interface()); err != nil {
					return self.Error(util.AddStr("字段[", field.Name, "]转换失败: ", err.Error()))
				} else {
					valuePart = append(valuePart, str)
				}
			} else if value.IsNil() {
				continue
			} else {
				str, err := util.ObjectToJson(value.Interface());
				if err != nil {
					fmt.Println("字段输出json失败: " + value.String())
				}
				fmt.Println(util.AddStr("警告: 不支持的字段[", field.Name, "]类型[", kind.String(), "] --- ", str))
				continue
			}
			fieldPart1.WriteString(" ")
			fieldPart1.WriteString(field.Tag.Get(sqlc.Bson))
			fieldPart1.WriteString(" = ?,")
		}
		valuePart = append(valuePart, idValue.Int())
		s1 := fieldPart1.String()
		s2 := fieldPart2.String()
		var sqlbuf bytes.Buffer
		sqlbuf.WriteString("update ")
		if tb, err := util.GetDbAndTb(data); err != nil {
			return self.Error(err)
		} else {
			sqlbuf.WriteString(tb)
		}
		sqlbuf.WriteString(" set")
		sqlbuf.WriteString(util.Substr(s1, 0, len(s1)-1))
		sqlbuf.WriteString(" where ")
		sqlbuf.WriteString(util.Substr(s2, 0, len(s2)-1))
		defer self.debug("Update", sqlbuf.String(), valuePart, start)
		var stmt *sql.Stmt
		var err error
		if self.AutoTx {
			stmt, err = self.Tx.Prepare(sqlbuf.String())
		} else {
			stmt, err = self.Db.Prepare(sqlbuf.String())
		}
		if err != nil {
			return self.Error(util.AddStr("预编译sql[", sqlbuf.String(), "]失败: ", err.Error()))
		}
		defer stmt.Close()
		if _, err := stmt.Exec(valuePart...); err != nil {
			return self.Error(util.AddStr("更新数据失败: ", err.Error()))
		}
	}
	return self.AddCacheSync(datas...)
}

func (self *RDBManager) Delete(datas ...interface{}) error {
	return util.Error("No implementation method [Delete] was found")
}

// 根据条件统计查询
func (self *RDBManager) Count(cnd *sqlc.Cnd) (int64, error) {
	start := util.Time()
	var elem = cnd.Model
	if elem == nil {
		return 0, self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	var fieldPart1, fieldPart2 bytes.Buffer
	var valuePart = make([]interface{}, 0)
	fieldPart1.WriteString("count(1)")
	part, args := self.BuildWhereCase(cnd)
	for e := range args {
		valuePart = append(valuePart, args[e])
	}
	if part.Len() > 0 {
		fieldPart2.WriteString("where")
		s := part.String()
		fieldPart2.WriteString(util.Substr(s, 0, len(s)-3))
	}
	s1 := fieldPart1.String()
	s2 := fieldPart2.String()
	var sqlbuf bytes.Buffer
	sqlbuf.WriteString("select ")
	sqlbuf.WriteString(s1)
	sqlbuf.WriteString(" from ")
	if tb, err := util.GetDbAndTb(elem); err != nil {
		return 0, self.Error(err)
	} else {
		sqlbuf.WriteString(tb)
	}
	sqlbuf.WriteString(" ")
	sqlbuf.WriteString(util.Substr(s2, 0, len(s2)-1))
	defer self.debug("Count", sqlbuf.String(), valuePart, start)
	var stmt *sql.Stmt
	var err error
	if self.AutoTx {
		stmt, err = self.Tx.Prepare(sqlbuf.String())
	} else {
		stmt, err = self.Db.Prepare(sqlbuf.String())
	}
	if err != nil {
		return 0, self.Error(util.AddStr("预编译sql[", sqlbuf.String(), "]失败: ", err.Error()))
	}
	defer stmt.Close()
	rows, err := stmt.Query(valuePart...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return 0, self.Error(util.AddStr("查询失败: ", err.Error()))
	}
	var pageTotal int64
	for rows.Next() {
		if err := rows.Scan(&pageTotal); err != nil {
			return 0, self.Error(util.AddStr("匹配结果异常: ", err.Error()))
		}
	}
	if err := rows.Err(); err != nil {
		return 0, self.Error(util.Error("读取查询结果失败: ", err.Error()))
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

// 按ID查询单条数据
func (self *RDBManager) FindById(data interface{}) error {
	start := util.Time()
	if data == nil {
		return self.Error("ORM对象不能为空")
	}
	if reflect.ValueOf(data).Kind() != reflect.Ptr {
		return self.Error("返回值必须为指针类型")
	}
	var fieldPart1, fieldPart2 bytes.Buffer
	var valuePart = make([]interface{}, 0)
	var fieldArray []reflect.StructField
	if vid := util.GetDataID(data); vid <= 0 {
		return self.Error("对象ID值不能为空")
	} else {
		valuePart = append(valuePart, vid)
	}
	tof := util.TypeOf(data)
	for i := 0; i < tof.NumField(); i++ {
		field := tof.Field(i)
		if util.ValidIgnore(field) {
			continue
		}
		fname := field.Tag.Get(sqlc.Bson)
		if len(fname) == 0 {
			return self.Error(util.AddStr("字段[", field.Name, "]无效bson标签"))
		}
		if field.Name == sqlc.Id {
			fieldPart1.WriteString(" id,")
			fieldArray = append(fieldArray, field)
			continue
		} else {
			fieldPart1.WriteString(" ")
			fieldPart1.WriteString(fname)
			fieldPart1.WriteString(",")
			fieldArray = append(fieldArray, field)
			continue
		}
	}
	fieldPart2.WriteString(" where id = ?,")
	s1 := fieldPart1.String()
	s2 := fieldPart2.String()
	var sqlbuf bytes.Buffer
	sqlbuf.WriteString("select ")
	sqlbuf.WriteString(util.Substr(s1, 0, len(s1)-1))
	sqlbuf.WriteString(" from ")
	if tb, err := util.GetDbAndTb(data); err != nil {
		return self.Error(err)
	} else {
		sqlbuf.WriteString(tb)
	}
	sqlbuf.WriteString(" ")
	sqlbuf.WriteString(util.Substr(s2, 0, len(s2)-1))
	defer self.debug("FindById", sqlbuf.String(), valuePart, start)
	var stmt *sql.Stmt
	var err error
	if self.AutoTx {
		stmt, err = self.Tx.Prepare(sqlbuf.String())
	} else {
		stmt, err = self.Db.Prepare(sqlbuf.String())
	}
	if err != nil {
		return self.Error(util.AddStr("预编译sql[", sqlbuf.String(), "]失败: ", err.Error()))
	}
	defer stmt.Close()
	rows, err := stmt.Query(valuePart...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return self.Error(util.AddStr("查询失败: ", err.Error()))
	}
	columns, err := rows.Columns()
	if err != nil && len(columns) != len(fieldArray) {
		return self.Error(util.AddStr("读取查询结果列长度失败: ", err.Error()))
	}
	raws, err := EchoResultRows(rows, len(columns))
	if err != nil {
		return self.Error(util.AddStr("读取查询结果失败: ", err.Error()))
	}
	if len(raws) <= 0 {
		return nil
	}
	if str, err := DataToMap(fieldArray, raws[0]); err != nil {
		return self.Error(err)
	} else if err := util.JsonToObject(str, data); err != nil {
		return self.Error(err)
	}
	return nil
}

// 按条件查询单条数据
func (self *RDBManager) FindOne(cnd *sqlc.Cnd, data interface{}) error {
	start := util.Time()
	if data == nil {
		return self.Error("返回值不能为空")
	}
	if reflect.TypeOf(data).Kind() != reflect.Ptr {
		return self.Error("返回值必须为指针类型")
	}
	var fieldPart1, fieldPart2 bytes.Buffer
	var valuePart = make([]interface{}, 0)
	var fieldArray []reflect.StructField
	var elem = cnd.Model
	if elem == nil {
		return self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	if util.TypeOf(data).Kind() != reflect.Struct {
		return self.Error("返回结果必须为struct类型")
	}
	tof := util.TypeOf(elem)
	if tof.Kind() != reflect.Struct && tof.Kind() != reflect.Ptr {
		return self.Error("ORM对象类型必须为struct或ptr")
	}
	for i := 0; i < tof.NumField(); i++ {
		field := tof.Field(i)
		if util.ValidIgnore(field) {
			continue
		}
		fname := field.Tag.Get(sqlc.Bson)
		if len(fname) == 0 {
			return self.Error(util.AddStr("字段[", field.Name, "]无效bson标签"))
		}
		if cnd != nil && len(cnd.AnyFields) > 0 {
			for e := range cnd.AnyFields {
				if field.Name == sqlc.Id && cnd.AnyFields[e] == sqlc.BsonId {
					fieldPart1.WriteString(" id,")
					fieldArray = append(fieldArray, field)
					continue
				}
				if cnd.AnyFields[e] == fname {
					fieldPart1.WriteString(" ")
					fieldPart1.WriteString(fname)
					fieldPart1.WriteString(",")
					fieldArray = append(fieldArray, field)
					continue
				}
			}
		} else {
			if field.Name == sqlc.Id {
				fieldPart1.WriteString(" id,")
				fieldArray = append(fieldArray, field)
				continue
			} else {
				fieldPart1.WriteString(" ")
				fieldPart1.WriteString(fname)
				fieldPart1.WriteString(",")
				fieldArray = append(fieldArray, field)
				continue
			}
		}
	}
	part, args := self.BuildWhereCase(cnd)
	for e := range args {
		valuePart = append(valuePart, args[e])
	}
	if part.Len() > 0 {
		fieldPart2.WriteString("where")
		s := part.String()
		fieldPart2.WriteString(util.Substr(s, 0, len(s)-3))
	}
	s1 := fieldPart1.String()
	s2 := fieldPart2.String()
	var sqlbuf bytes.Buffer
	sqlbuf.WriteString("select ")
	sqlbuf.WriteString(util.Substr(s1, 0, len(s1)-1))
	sqlbuf.WriteString(" from ")
	if tb, err := util.GetDbAndTb(elem); err != nil {
		return self.Error(err)
	} else {
		sqlbuf.WriteString(tb)
	}
	sqlbuf.WriteString(" ")
	sqlbuf.WriteString(util.Substr(s2, 0, len(s2)-1))
	sortby := self.BuilSortBy(cnd)
	if len(sortby) > 0 {
		sqlbuf.WriteString(sortby)
	}
	cnd.Pagination = dialect.Dialect{PageNo: 1, PageSize: 1}
	limitSql, err := self.BuildPagination(cnd, sqlbuf.String(), valuePart);
	if err != nil {
		return self.Error(err)
	}
	defer self.debug("FindOne", sqlbuf.String(), valuePart, start)
	var stmt *sql.Stmt
	if self.AutoTx {
		stmt, err = self.Tx.Prepare(limitSql)
	} else {
		stmt, err = self.Db.Prepare(limitSql)
	}
	if err != nil {
		return self.Error(util.AddStr("预编译sql[", sqlbuf.String(), "]失败: ", err.Error()))
	}
	defer stmt.Close()
	rows, err := stmt.Query(valuePart...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return self.Error(util.AddStr("查询失败: ", err.Error()))
	}
	columns, err := rows.Columns()
	if err != nil && len(columns) != len(fieldArray) {
		return self.Error(util.AddStr("读取查询结果列长度失败: ", err.Error()))
	}
	raws, err := EchoResultRows(rows, len(columns))
	if err != nil {
		return self.Error(util.AddStr("读取查询结果失败: ", err.Error()))
	}
	if len(raws) <= 0 {
		return nil
	}
	if str, err := DataToMap(fieldArray, raws[0]); err != nil {
		return self.Error(err)
	} else if err := util.JsonToObject(str, data); err != nil {
		return self.Error(err)
	}
	return nil
}

func (self *RDBManager) FindList(cnd *sqlc.Cnd, data interface{}) error {
	start := util.Time()
	if cnd == nil {
		return self.Error("条件参数不能为空")
	}
	if data == nil {
		return self.Error("返回值不能为空")
	}
	if reflect.TypeOf(data).Kind() != reflect.Ptr {
		return self.Error("返回值必须为指针类型")
	}
	var fieldPart1, fieldPart2 bytes.Buffer
	var valuePart = make([]interface{}, 0)
	var fieldArray []reflect.StructField
	var elem = cnd.Model
	if elem == nil {
		return self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	if util.TypeOf(data).Kind() != reflect.Slice {
		return self.Error("返回结果必须为数组类型")
	}
	tof := util.TypeOf(elem)
	if tof.Kind() != reflect.Struct && tof.Kind() != reflect.Ptr {
		return self.Error("ORM对象类型必须为struct或ptr")
	}
	for i := 0; i < tof.NumField(); i++ {
		field := tof.Field(i)
		if util.ValidIgnore(field) {
			continue
		}
		fname := field.Tag.Get(sqlc.Bson)
		if len(fname) == 0 {
			return self.Error(util.AddStr("字段[", field.Name, "]无效bson标签"))
		}
		if cnd != nil && len(cnd.AnyFields) > 0 {
			for e := range cnd.AnyFields {
				if field.Name == sqlc.Id && cnd.AnyFields[e] == sqlc.BsonId {
					fieldPart1.WriteString(" id,")
					fieldArray = append(fieldArray, field)
					continue
				}
				if cnd.AnyFields[e] == fname {
					fieldPart1.WriteString(" ")
					fieldPart1.WriteString(fname)
					fieldPart1.WriteString(",")
					fieldArray = append(fieldArray, field)
					continue
				}
			}
		} else {
			if field.Name == sqlc.Id {
				fieldPart1.WriteString(" id,")
				fieldArray = append(fieldArray, field)
				continue
			} else {
				fieldPart1.WriteString(" ")
				fieldPart1.WriteString(fname)
				fieldPart1.WriteString(",")
				fieldArray = append(fieldArray, field)
				continue
			}
		}
	}
	part, args := self.BuildWhereCase(cnd)
	for e := range args {
		valuePart = append(valuePart, args[e])
	}
	if part.Len() > 0 {
		fieldPart2.WriteString("where")
		s := part.String()
		fieldPart2.WriteString(util.Substr(s, 0, len(s)-3))
	}
	s1 := fieldPart1.String()
	s2 := fieldPart2.String()
	var sqlbuf bytes.Buffer
	sqlbuf.WriteString("select ")
	sqlbuf.WriteString(util.Substr(s1, 0, len(s1)-1))
	sqlbuf.WriteString(" from ")
	if tb, err := util.GetDbAndTb(elem); err != nil {
		return self.Error(err)
	} else {
		sqlbuf.WriteString(tb)
	}
	sqlbuf.WriteString(" ")
	sqlbuf.WriteString(util.Substr(s2, 0, len(s2)-1))
	sortby := self.BuilSortBy(cnd)
	if len(sortby) > 0 {
		sqlbuf.WriteString(sortby)
	}
	limitSql, err := self.BuildPagination(cnd, sqlbuf.String(), valuePart);
	if err != nil {
		return self.Error(err)
	}
	defer self.debug("FindList", sqlbuf.String(), valuePart, start)
	var stmt *sql.Stmt
	if self.AutoTx {
		stmt, err = self.Tx.Prepare(limitSql)
	} else {
		stmt, err = self.Db.Prepare(limitSql)
	}
	if err != nil {
		return self.Error(util.AddStr("预编译sql[", sqlbuf.String(), "]失败: ", err.Error()))
	}
	defer stmt.Close()
	rows, err := stmt.Query(valuePart...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return self.Error(util.AddStr("查询失败: ", err.Error()))
	}
	columns, err := rows.Columns()
	if err != nil && len(columns) != len(fieldArray) {
		return self.Error(util.AddStr("读取查询结果列长度失败: ", err.Error()))
	}
	raws, err := EchoResultRows(rows, len(columns))
	if err != nil {
		return self.Error(util.AddStr("读取查询结果失败: ", err.Error()))
	}
	resultv := reflect.ValueOf(data)
	if resultv.Kind() != reflect.Ptr || resultv.Elem().Kind() != reflect.Slice {
		return self.Error("列表结果参数必须是指针类型")
	}
	slicev := resultv.Elem()
	slicev = slicev.Slice(0, slicev.Cap())
	for i := 0; i < len(raws); i++ {
		str, err := DataToMap(fieldArray, raws[i])
		if err != nil {
			return self.Error(err)
		}
		model := util.NewInstance(elem)
		if err := util.JsonToObject(str, &model); err != nil {
			return self.Error(err)
		}
		slicev = reflect.Append(slicev, reflect.ValueOf(model))
	}
	slicev = slicev.Slice(0, slicev.Cap())
	resultv.Elem().Set(slicev.Slice(0, len(raws)))
	return nil
}

func (self *RDBManager) FindComplex(cnd *sqlc.Cnd, data interface{}) error {
	start := util.Time()
	if cnd == nil {
		return self.Error("条件参数不能为空")
	}
	if data == nil {
		return self.Error("返回值不能为空")
	}
	if reflect.ValueOf(data).Kind() != reflect.Ptr {
		return self.Error("返回值必须为指针类型")
	}
	if len(cnd.AnyFields) == 0 {
		return self.Error("查询字段不能为空")
	}
	var fieldPart1, fieldPart2 bytes.Buffer
	var valuePart = make([]interface{}, 0)
	var elem = cnd.Model
	if elem == nil {
		return self.Error("ORM对象类型不能为空,请通过M(...)方法设置对象类型")
	}
	tof := util.TypeOf(elem)
	if tof.Kind() != reflect.Struct && tof.Kind() != reflect.Ptr {
		return self.Error("ORM对象类型必须为struct或ptr")
	}
	for i := 0; i < len(cnd.AnyFields); i++ {
		fieldPart1.WriteString(" ")
		fieldPart1.WriteString(cnd.AnyFields[i])
		fieldPart1.WriteString(",")
		continue
	}
	part, args := self.BuildWhereCase(cnd)
	for e := range args {
		valuePart = append(valuePart, args[e])
	}
	if part.Len() > 0 {
		fieldPart2.WriteString("where")
		s := part.String()
		fieldPart2.WriteString(util.Substr(s, 0, len(s)-3))
	}
	s1 := fieldPart1.String()
	s2 := fieldPart2.String()
	var sqlbuf bytes.Buffer
	sqlbuf.WriteString("select ")
	sqlbuf.WriteString(util.Substr(s1, 0, len(s1)-1))
	sqlbuf.WriteString(" from ")
	sqlbuf.WriteString(cnd.FromCond.Table)
	sqlbuf.WriteString(" ")
	if len(cnd.JoinCond) > 0 {
		for e := range cnd.JoinCond {
			cond := cnd.JoinCond[e]
			if len(cond.Table) == 0 || len(cond.On) == 0 {
				continue
			}
			if cond.Type == sqlc.LEFT_ {
				sqlbuf.WriteString(" left join ")
			} else if cond.Type == sqlc.RIGHT_ {
				sqlbuf.WriteString(" right join ")
			} else if cond.Type == sqlc.INNER_ {
				sqlbuf.WriteString(" inner join ")
			} else {
				continue
			}
			sqlbuf.WriteString(cond.Table)
			sqlbuf.WriteString(" on ")
			sqlbuf.WriteString(cond.On)
			sqlbuf.WriteString(" ")
		}
	}
	sqlbuf.WriteString(util.Substr(s2, 0, len(s2)-1))
	groupby := self.BuilGroupBy(cnd)
	if len(groupby) > 0 {
		sqlbuf.WriteString(" ")
		sqlbuf.WriteString(groupby)
	}
	sortby := self.BuilSortBy(cnd)
	if len(sortby) > 0 {
		sqlbuf.WriteString(" ")
		sqlbuf.WriteString(sortby)
	}
	limitSql, err := self.BuildPagination(cnd, sqlbuf.String(), valuePart);
	if err != nil {
		return self.Error(err)
	}
	defer self.debug("FindComplex", sqlbuf.String(), valuePart, start)
	var stmt *sql.Stmt
	if self.AutoTx {
		stmt, err = self.Tx.Prepare(limitSql)
	} else {
		stmt, err = self.Db.Prepare(limitSql)
	}
	if err != nil {
		return self.Error(util.AddStr("预编译sql[", sqlbuf.String(), "]失败: ", err.Error()))
	}
	defer stmt.Close()
	rows, err := stmt.Query(valuePart...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return self.Error(util.AddStr("查询失败: ", err.Error()))
	}
	columns, err := rows.Columns()
	if err != nil && len(columns) != len(cnd.AnyFields) {
		return self.Error(util.AddStr("读取查询结果列长度失败: ", err.Error()))
	}
	raws, err := EchoResultRows(rows, len(columns))
	if err != nil {
		return self.Error(util.AddStr("读取查询结果失败: ", err.Error()))
	}
	var fieldArray []reflect.StructField
	for e := range columns {
		for i := 0; i < tof.NumField(); i++ {
			field := tof.Field(i)
			fname := field.Tag.Get(sqlc.Json)
			if len(fname) == 0 {
				return self.Error(util.AddStr("字段[", field.Name, "]无效json标签"))
			}
			if columns[e] == fname {
				fieldArray = append(fieldArray, field)
				continue
			}
		}
	}
	if reflect.TypeOf(data).Elem().Kind() == reflect.Slice {
		resultv := reflect.ValueOf(data)
		if resultv.Kind() != reflect.Ptr || resultv.Elem().Kind() != reflect.Slice {
			return self.Error("列表结果参数必须是指针类型")
		}
		slicev := resultv.Elem()
		slicev = slicev.Slice(0, slicev.Cap())
		for i := 0; i < len(raws); i++ {
			str, err := DataToMap(fieldArray, raws[i])
			if err != nil {
				return self.Error(err)
			}
			model := util.NewInstance(elem)
			if err := util.JsonToObject(str, &model); err != nil {
				return self.Error(err)
			}
			slicev = reflect.Append(slicev, reflect.ValueOf(model))
		}
		slicev = slicev.Slice(0, slicev.Cap())
		resultv.Elem().Set(slicev.Slice(0, len(raws)))
	} else {
		if len(raws) <= 0 {
			return nil
		}
		if str, err := DataToMap(fieldArray, raws[0]); err != nil {
			return self.Error(err)
		} else if err := util.JsonToObject(str, data); err != nil {
			return self.Error(err)
		}
	}
	return nil
}

func (self *RDBManager) Close() error {
	if self.AutoTx && self.Tx != nil {
		if self.Errors != nil && len(self.Errors) > 0 {
			if err := self.Tx.Rollback(); err != nil {
				log.Println("事务回滚失败: ", err.Error())
			}
		} else {
			if err := self.Tx.Commit(); err != nil {
				log.Println("事务提交失败: ", err.Error())
			}
		}
	}
	if self.CacheSync && len(self.CacheObject) > 0 {
		for e := range self.CacheObject {
			if err := self.mongoSyncData(self.CacheObject[e]); err != nil {
				log.Print(err.Error())
			}
		}
	}
	return nil
}

// mongo同步数据
func (self *RDBManager) mongoSyncData(data interface{}) error {
	if sync, err := util.ValidSyncMongo(data); err != nil {
		return util.Error("实体字段异常: ", err.Error())
	} else if sync {
		mongo, err := new(MGOManager).Get(self.Option);
		if err != nil {
			return util.Error("获取mongo连接失败: ", err.Error())
		}
		defer mongo.Close()
		if err := mongo.Save(data); err != nil {
			if s, e := util.ObjectToJson(data); e != nil {
				return util.Error("同步mongo数据失败,JSON对象转换失败: ", e.Error())
			} else {
				return util.Error("同步mongo数据失败: ", s, ", 异常: ", err.Error())
			}
		}
	}
	return nil
}

// 结果集根据对象字段类型填充到map实例
func DataToMap(fieldArray []reflect.StructField, raw [][]byte) (string, error) {
	result := make(map[string]interface{})
	for i := range fieldArray {
		field := fieldArray[i]
		var f string
		if field.Name == sqlc.Id {
			f = sqlc.BsonId
		} else {
			f = field.Tag.Get(sqlc.Bson)
		}
		kind := field.Type.String()
		vs := string(raw[i])
		if len(vs) == 0 {
			continue
		}
		if kind == "int" {
			if i0, err := util.StrToInt(string(raw[i])); err != nil {
				return "", util.Error("对象字段[", f, "]转换int失败")
			} else {
				result[f] = i0
			}
		} else if kind == "int8" {
			if i8, err := util.StrToInt8(string(raw[i])); err != nil {
				return "", util.Error("对象字段[", f, "]转换int8失败")
			} else {
				result[f] = i8
			}
		} else if kind == "int16" {
			if i16, err := util.StrToInt16(string(raw[i])); err != nil {
				return "", util.Error("对象字段[", f, "]转换int16失败")
			} else {
				result[f] = i16
			}
		} else if kind == "int32" {
			if i32, err := util.StrToInt32(string(raw[i])); err != nil {
				return "", util.Error("对象字段[", f, "]转换int32失败")
			} else {
				result[f] = i32
			}
		} else if kind == "int64" {
			if util.ValidDate(field) {
				if i64, err := util.Str2Time(string(raw[i])); err != nil {
					return "", util.Error("对象字段[", f, "]转换int64失败: ", err.Error())
				} else {
					result[f] = i64
				}
			} else {
				if i64, err := util.StrToInt64(string(raw[i])); err != nil {
					return "", util.Error("对象字段[", f, "]转换int64失败")
				} else {
					result[f] = i64
				}
			}
		} else if kind == "string" {
			result[f] = string(raw[i])
		} else if kind == "[]string" {
			array := make([]string, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "[]int" {
			array := make([]int, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "[]int8" {
			array := make([]int8, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "[]int16" {
			array := make([]int16, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "[]int32" {
			array := make([]int32, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "[]int64" {
			array := make([]int64, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "[]interface {}" {
			array := make([]interface{}, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "[]string" {
			array := make([]string, 0)
			if err := util.JsonToObject(string(raw[i]), &array); err != nil {
				return "", err
			}
			result[f] = array
		} else if kind == "map[string]interface {}" {
			mmp := make(map[string]interface{})
			if err := util.JsonToObject(string(raw[i]), &mmp); err != nil {
				return "", err
			}
			result[f] = mmp
		} else if kind == "map[string]int" {
			mmp := make(map[string]int)
			if err := util.JsonToObject(string(raw[i]), &mmp); err != nil {
				return "", err
			}
			result[f] = mmp
		} else if kind == "map[string]int8" {
			mmp := make(map[string]int8)
			if err := util.JsonToObject(string(raw[i]), &mmp); err != nil {
				return "", err
			}
			result[f] = mmp
		} else if kind == "map[string]int16" {
			mmp := make(map[string]int16)
			if err := util.JsonToObject(string(raw[i]), &mmp); err != nil {
				return "", err
			}
			result[f] = mmp
		} else if kind == "map[string]int32" {
			mmp := make(map[string]int32)
			if err := util.JsonToObject(string(raw[i]), &mmp); err != nil {
				return "", err
			}
			result[f] = mmp
		} else if kind == "map[string]int64" {
			mmp := make(map[string]int64)
			if err := util.JsonToObject(string(raw[i]), &mmp); err != nil {
				return "", err
			}
			result[f] = mmp
		} else if kind == "map[string]string" {
			mmp := make(map[string]string)
			if err := util.JsonToObject(string(raw[i]), &mmp); err != nil {
				return "", err
			}
			result[f] = mmp
		} else {
			return "", util.Error("对象字段[", f, "]转换失败([", kind, "]类型不支持)")
		}
	}
	return util.ObjectToJson(result)
}

// 输出查询结果集
func EchoResultRows(rows *sql.Rows, len int) ([][][]byte, error) {
	raws := make([][][]byte, 0)
	for rows.Next() {
		result := make([][]byte, len)
		dest := make([]interface{}, len)
		for i, _ := range result {
			dest[i] = &result[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, util.Error("数据结果匹配异常: ", err.Error())
		}
		raws = append(raws, result)
	}
	if err := rows.Err(); err != nil {
		return nil, util.Error("读取查询结果失败: ", err.Error())
	}
	return raws, nil
}

// 基础条件判定是否有数据库别名
func (self *RDBManager) BuildCondKey(cnd *sqlc.Cnd, key string) string {
	var fieldPart bytes.Buffer
	fieldPart.WriteString(" ")
	fieldPart.WriteString(key)
	return fieldPart.String()
}

// 构建where条件
func (self *RDBManager) BuildWhereCase(cnd *sqlc.Cnd) (bytes.Buffer, []interface{}) {
	var fieldPart bytes.Buffer
	var valuePart []interface{}
	if cnd == nil {
		return fieldPart, valuePart
	}
	for e := range cnd.Conditions {
		condit := cnd.Conditions[e]
		key := condit.Key
		value := condit.Value
		values := condit.Values
		switch condit.Logic {
		// case condition
		case sqlc.EQ_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" = ? and")
			valuePart = append(valuePart, value)
		case sqlc.NOT_EQ_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" <> ? and")
			valuePart = append(valuePart, value)
		case sqlc.LT_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" < ? and")
			valuePart = append(valuePart, value)
		case sqlc.LTE_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" <= ? and")
			valuePart = append(valuePart, value)
		case sqlc.GT_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" > ? and")
			valuePart = append(valuePart, value)
		case sqlc.GTE_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" >= ? and")
			valuePart = append(valuePart, value)
		case sqlc.IS_NULL_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" is null and")
		case sqlc.IS_NOT_NULL_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" is not null and")
		case sqlc.BETWEEN_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" between ? and ? and")
			valuePart = append(valuePart, values[0])
			valuePart = append(valuePart, values[1])
		case sqlc.NOT_BETWEEN_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" not between ? and ? and")
			valuePart = append(valuePart, values[0])
			valuePart = append(valuePart, values[1])
		case sqlc.IN_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" in(")
			var buf bytes.Buffer
			for e := range values {
				buf.WriteString("?,")
				valuePart = append(valuePart, values[e])
			}
			s := buf.String()
			fieldPart.WriteString(util.Substr(s, 0, len(s)-1))
			fieldPart.WriteString(") and")
		case sqlc.NOT_IN_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" not in(")
			var buf bytes.Buffer
			for e := range values {
				buf.WriteString("?,")
				valuePart = append(valuePart, values[e])
			}
			s := buf.String()
			fieldPart.WriteString(util.Substr(s, 0, len(s)-1))
			fieldPart.WriteString(") and")
		case sqlc.LIKE_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" like concat('%',?,'%') and")
			valuePart = append(valuePart, value)
		case sqlc.NO_TLIKE_:
			fieldPart.WriteString(self.BuildCondKey(cnd, key))
			fieldPart.WriteString(" not like concat('%',?,'%') and")
			valuePart = append(valuePart, value)
		case sqlc.OR_:
			var orpart bytes.Buffer
			var args []interface{}
			for e := range values {
				cnd, ok := values[e].(*sqlc.Cnd)
				if !ok {
					continue
				}
				buf, arg := self.BuildWhereCase(cnd)
				s := buf.String()
				s = util.Substr(s, 0, len(s)-3)
				orpart.WriteString(s)
				orpart.WriteString(" or")
				for e := range arg {
					args = append(args, arg[e])
				}
			}
			s := orpart.String()
			s = util.Substr(s, 0, len(s)-3)
			fieldPart.WriteString(" (")
			fieldPart.WriteString(s)
			fieldPart.WriteString(") and")
			for e := range args {
				valuePart = append(valuePart, args[e])
			}
		}
	}
	return fieldPart, valuePart
}

// 构建分组命令
func (self *RDBManager) BuilGroupBy(cnd *sqlc.Cnd) string {
	if cnd == nil || len(cnd.Groupbys) <= 0 {
		return ""
	}
	var groupby = bytes.Buffer{}
	groupby.WriteString(" group by")
	for e := range cnd.Groupbys {
		if len(cnd.Groupbys[e]) == 0 {
			continue
		}
		groupby.WriteString(" ")
		groupby.WriteString(cnd.Groupbys[e])
		groupby.WriteString(",")
	}
	s := groupby.String()
	s = util.Substr(s, 0, len(s)-1)
	return s
}

// 构建排序命令
func (self *RDBManager) BuilSortBy(cnd *sqlc.Cnd) string {
	if cnd == nil || len(cnd.Orderbys) <= 0 {
		return ""
	}
	var sortby = bytes.Buffer{}
	sortby.WriteString(" order by")
	for e := range cnd.Orderbys {
		orderby := cnd.Orderbys[e]
		sortby.WriteString(" ")
		sortby.WriteString(orderby.Key)
		if orderby.Value == sqlc.DESC_ {
			sortby.WriteString(" desc,")
		} else if orderby.Value == sqlc.ASC_ {
			sortby.WriteString(" asc,")
		}
	}
	s := sortby.String()
	s = util.Substr(s, 0, len(s)-1)
	return s
}

// 构建分页命令
func (self *RDBManager) BuildPagination(cnd *sqlc.Cnd, sqlbuf string, values []interface{}) (string, error) {
	start := util.Time()
	if cnd == nil {
		return sqlbuf, nil
	}
	pagination := cnd.Pagination
	if pagination.PageNo == 0 && pagination.PageSize == 0 {
		return sqlbuf, nil
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	dialect := dialect.MysqlDialect{pagination}
	limitSql, err := dialect.GetLimitSql(sqlbuf)
	if err != nil {
		return "", err
	}
	if !dialect.IsOffset {
		countSql, err := dialect.GetCountSql(sqlbuf)
		defer self.debug("PageCountSql", countSql, values, start)
		if err != nil {
			return "", err
		}
		var rows *sql.Rows
		if self.AutoTx {
			rows, err = self.Tx.Query(countSql, values...)
		} else {
			rows, err = self.Db.Query(countSql, values...)
		}
		if rows != nil {
			defer rows.Close()
		}
		if err != nil {
			return "", self.Error(util.AddStr("Count查询失败: ", err.Error()))
		}
		var pageTotal int64
		for rows.Next() {
			if err := rows.Scan(&pageTotal); err != nil {
				return "", self.Error(util.AddStr("匹配结果异常: ", err.Error()))
			}
		}
		if err := rows.Err(); err != nil {
			return "", self.Error(util.Error("读取查询结果失败: ", err.Error()))
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
	return limitSql, nil
}

// 添加缓存同步对象
func (self *RDBManager) AddCacheSync(models ...interface{}) error {
	if self.CacheSync && models != nil && len(models) > 0 {
		for e := range models {
			self.CacheObject = append(self.CacheObject, models[e])
		}
	}
	return nil
}

func (self *RDBManager) debug(title, sql string, values interface{}, start int64) {
	if self.Debug {
		str, _ := util.ObjectToJson(values)
		log.Println(util.AddStr("mysql debug -> ", title, ": ", sql, " --- ", str, " --- cost: ", util.AnyToStr(util.Time()-start)))
	}
}
