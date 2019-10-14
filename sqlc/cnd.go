package sqlc

import (
	"github.com/godaddy-x/jorm/dialect"
)

/**
 * @author shadow
 * @createby 2018.10.10
 */

// 数据库操作类型枚举
const (
	EQ_ = iota
	NOT_EQ_
	LT_
	LTE_
	GT_
	GTE_
	IS_NULL_
	IS_NOT_NULL_
	BETWEEN_
	NOT_BETWEEN_
	IN_
	NOT_IN_
	LIKE_
	NO_TLIKE_
	OR_
	ORDER_BY_
	ASC_
	DESC_
	LEFT_
	RIGHT_
	INNER_
	SUM_
	AVG_
	MIN_
	MAX_
)

var (
	Id     = "Id"
	Ignore = "ignore"
	Bson   = "bson"
	Json   = "json"
	Mg     = "mg"
	True   = "true"
	BsonId = "id"
	Date   = "date"
)

// 数据库操作逻辑条件对象
type Condition struct {
	Logic  int
	Key    string
	Value  interface{}
	Values []interface{}
}

// 连接表条件对象
type JoinCond struct {
	Type  int
	Table string
	Alias string
	On    string
}

// 主表条件对象
type FromCond struct {
	Table string
	Alias string
}

// 数据库操作汇总逻辑条件对象
type Cnd struct {
	Conditions  []Condition
	AnyFields   []string
	Distincts   []string
	Groupbys    []string
	Orderbys    []Condition
	Aggregates  []Condition
	UpdateKV    map[string]interface{}
	Model       interface{}
	Pagination  dialect.Dialect
	FromCond    FromCond
	JoinCond    []JoinCond
	CacheConfig CacheConfig
}

// 缓存结果集参数
type CacheConfig struct {
	Open   bool
	Prefix string
	Key    string
	Expire int
}

// args[0]=对象类型 args[1]=数据表别名
func M(model interface{}) *Cnd {
	return &Cnd{
		Conditions: make([]Condition, 0),
		AnyFields:  make([]string, 0),
		Distincts:  make([]string, 0),
		Groupbys:   make([]string, 0),
		Orderbys:   make([]Condition, 0),
		UpdateKV:   make(map[string]interface{}),
		Model:      model,
	}
}

// 保存基础命令操作
func addDefaultCondit(cnd *Cnd, condit Condition) *Cnd {
	cnd.Conditions = append(cnd.Conditions, condit)
	return cnd
}

// =
func (self *Cnd) Eq(key string, value interface{}) *Cnd {
	condit := Condition{EQ_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// <>
func (self *Cnd) NotEq(key string, value interface{}) *Cnd {
	condit := Condition{NOT_EQ_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// <
func (self *Cnd) Lt(key string, value interface{}) *Cnd {
	condit := Condition{LT_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// <=
func (self *Cnd) Lte(key string, value interface{}) *Cnd {
	condit := Condition{LTE_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// >
func (self *Cnd) Gt(key string, value interface{}) *Cnd {
	condit := Condition{GT_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// >=
func (self *Cnd) Gte(key string, value interface{}) *Cnd {
	condit := Condition{GTE_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// is null
func (self *Cnd) IsNull(key string) *Cnd {
	condit := Condition{IS_NULL_, key, nil, nil}
	return addDefaultCondit(self, condit)
}

// is not null
func (self *Cnd) IsNotNull(key string) *Cnd {
	condit := Condition{IS_NOT_NULL_, key, nil, nil}
	return addDefaultCondit(self, condit)
}

// between
func (self *Cnd) Between(key string, value1 interface{}, value2 interface{}) *Cnd {
	condit := Condition{BETWEEN_, key, nil, []interface{}{value1, value2}}
	return addDefaultCondit(self, condit)
}

// not between
func (self *Cnd) NotBetween(key string, value1 interface{}, value2 interface{}) *Cnd {
	condit := Condition{NOT_BETWEEN_, key, nil, []interface{}{value1, value2}}
	return addDefaultCondit(self, condit)
}

// in
func (self *Cnd) In(key string, values ...interface{}) *Cnd {
	condit := Condition{IN_, key, nil, values}
	return addDefaultCondit(self, condit)
}

// not in
func (self *Cnd) NotIn(key string, values ...interface{}) *Cnd {
	condit := Condition{NOT_IN_, key, nil, values}
	return addDefaultCondit(self, condit)
}

// aggregate
func (self *Cnd) Aggregate(agg int, key string) *Cnd {
	condit := Condition{agg, key, nil, nil}
	self.Aggregates = append(self.Aggregates, condit)
	return self
}

// like
func (self *Cnd) Like(key string, value interface{}) *Cnd {
	condit := Condition{LIKE_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// not like
func (self *Cnd) NotLike(key string, value interface{}) *Cnd {
	condit := Condition{NO_TLIKE_, key, value, nil}
	return addDefaultCondit(self, condit)
}

// or
func (self *Cnd) Or(cnds ...interface{}) *Cnd {
	condit := Condition{OR_, "", nil, cnds}
	return addDefaultCondit(self, condit)
}

// 复杂查询设定首个from table as
func (self *Cnd) From(fromTable string) *Cnd {
	self.FromCond = FromCond{fromTable, ""}
	return self
}

// left join
func (self *Cnd) Join(join int, table string, on string) *Cnd {
	if len(table) == 0 || len(on) == 0 {
		return self
	}
	self.JoinCond = append(self.JoinCond, JoinCond{join, table, "", on})
	return self
}

// limit,以页数跨度查询
func (self *Cnd) Limit(pageNo int64, pageSize int64) *Cnd {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 || pageSize > 5000 {
		pageSize = 10
	}
	self.Pagination = dialect.Dialect{pageNo, pageSize, 0, 0, true, false}
	return self
}

// offset,以下标跨度查询
func (self *Cnd) Offset(offset int64, limit int64) *Cnd {
	if offset <= 0 {
		offset = 0
	}
	if limit <= 0 || limit > 5000 {
		limit = 10
	}
	self.Pagination = dialect.Dialect{offset, limit, 0, 0, true, true}
	return self
}

// 筛选字段去重
func (self *Cnd) Distinct(keys ...string) *Cnd {
	for e := range keys {
		self.Distincts = append(self.Distincts, keys[e])
	}
	return self
}

// 按字段分组
func (self *Cnd) Groupby(keys ...string) *Cnd {
	for e := range keys {
		self.Groupbys = append(self.Groupbys, keys[e])
	}
	return self
}

// 按字段排序
func (self *Cnd) Orderby(key string, sortby int) *Cnd {
	condit := Condition{ORDER_BY_, key, sortby, nil}
	self.Orderbys = append(self.Orderbys, condit)
	return self
}

// 筛选指定字段查询
func (self *Cnd) Fields(keys ...string) *Cnd {
	for e := range keys {
		self.AnyFields = append(self.AnyFields, keys[e])
	}
	return self
}

// 缓存指定结果集
func (self *Cnd) Cache(config CacheConfig) *Cnd {
	self.CacheConfig = config
	self.CacheConfig.Open = true
	return self
}

// 指定更新字段
func (self *Cnd) UpdateKeyValue(keys []string, values ...interface{}) *Cnd {
	if len(keys) == 0 || len(keys) != len(values) {
		println("keys和values参数下标不对等")
		return self
	}
	for i := 0; i < len(keys); i++ {
		self.UpdateKV[keys[i]] = values[i]
	}
	return self
}
