package util

/**
 * @author shadow
 * @createby 2018.10.10
 */

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/godaddy-x/jorm/snowflake"
	"github.com/godaddy-x/jorm/sqlc"
	"github.com/json-iterator/go"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	cst_sh, _  = time.LoadLocation("Asia/Shanghai") //上海
	time_formt = "2006-01-02 15:04:05"
	snowflakes = make(map[int64]*snowflake.Node, 0)
	mu         sync.Mutex
)

func init() {
	node, _ := snowflake.NewNode(0)
	snowflakes[0] = node
}

// 对象转JSON字符串
func ObjectToJson(src interface{}) (string, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if result, err := json.Marshal(src); err != nil {
		return "", errors.New("Json字符串转换对象出现异常: " + err.Error())
	} else {
		return string(result), nil
	}
}

// JSON字符串转对象
func JsonToObject(src string, target interface{}) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err := json.Unmarshal([]byte(src), target); err != nil {
		return errors.New("Json字符串转换对象出现异常: " + err.Error())
	}
	return nil
}

// 对象转对象
func JsonToAny(src interface{}, target interface{}) error {
	if src == nil || target == nil {
		return errors.New("参数不能为空")
	}
	str, err := ObjectToJson(src)
	if err != nil {
		return err
	}
	if err := JsonToObject(str, target); err != nil {
		return err
	}
	return nil
}

// JSON字符串转对象
func JsonToObject2(src string, target interface{}) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	d := json.NewDecoder(bytes.NewBuffer([]byte(src)))
	d.UseNumber()
	if err := d.Decode(target); err != nil {
		return errors.New("Json字符串转换对象出现异常: " + err.Error())
	}
	return nil
}

// 对象转对象
func JsonToAny2(src interface{}, target interface{}) error {
	if src == nil || target == nil {
		return errors.New("参数不能为空")
	}
	str, err := ObjectToJson(src)
	if err != nil {
		return err
	}
	if err := JsonToObject2(str, target); err != nil {
		return err
	}
	return nil
}

// 通过反射获取对象Id值
func GetDataID(data interface{}) int64 {
	valueOf := reflect.ValueOf(data)
	return reflect.Indirect(valueOf).FieldByName("Id").Int()
}

// 通过反射获取对象数据表标签
func GetDbAndTb(model interface{}) (string, error) {
	tof := reflect.TypeOf(model)
	vof := reflect.ValueOf(model)
	if tof.Kind() == reflect.Ptr {
		tof = tof.Elem()
		vof = vof.Elem()
	}
	// fmt.Println(vof.Kind().String())
	if vof.Kind().String() != reflect.Struct.String() && vof.Kind().String() != reflect.Ptr.String() {
		return "", errors.New("参数非struct或ptr类型")
	}
	field, ok := tof.FieldByName("Id")
	if !ok {
		return "", errors.New("实体Id字段不能为空")
	}
	tb := field.Tag.Get("tb")
	if tb == "" {
		return "", errors.New("实体数据库表名称标签不能为空")
	}
	return tb, nil
}

// 通过反射获取对象或指针具体类型
func TypeOf(data interface{}) reflect.Type {
	tof := reflect.TypeOf(data)
	if tof.Kind() == reflect.Ptr {
		tof = tof.Elem()
	}
	return tof
}

// 通过反射获取对象或指针具体类型
func ValueOf(data interface{}) reflect.Value {
	vof := reflect.ValueOf(data)
	if vof.Kind() == reflect.Ptr {
		vof = vof.Elem()
	}
	return vof
}

// 检测是否同步mongo
func ValidSyncMongo(model interface{}) (bool, error) {
	typeOf := reflect.TypeOf(model)
	var field reflect.StructField
	var ok bool
	if typeOf.Kind() == reflect.Ptr {
		field, ok = typeOf.Elem().FieldByName("Id")
	} else if typeOf.Kind() == reflect.Struct {
		field, ok = typeOf.FieldByName("Id")
	} else {
		return false, errors.New("实体类型异常")
	}
	if !ok {
		return false, errors.New("实体Id字段不能为空")
	}
	mg := field.Tag.Get("mg")
	if mg == "true" {
		return true, nil
	}
	return false, nil
}

// 通过反射实例化对象
func NewInstance(data interface{}) interface{} {
	tof := reflect.TypeOf(data)
	if tof.Kind() == reflect.Ptr {
		tof = tof.Elem()
	}
	return reflect.New(tof).Interface()
}

// 获取当前时间/毫秒
func Time(t ...time.Time) int64 {
	if len(t) > 0 {
		return t[0].In(cst_sh).UnixNano() / 1e6
	}
	return time.Now().In(cst_sh).UnixNano() / 1e6
}

// 时间戳转time
func Int2Time(t int64) time.Time {
	return time.Unix(t/1000, 0).In(cst_sh)
}

// 时间戳转格式字符串/毫秒
func Time2Str(t int64) string {
	return Int2Time(t).Format(time_formt)
}

// 格式字符串转时间戳/毫秒
func Str2Time(s string) (int64, error) {
	t, err := time.ParseInLocation(time_formt, s, cst_sh)
	if err != nil {
		return 0, err
	}
	return Time(t), nil
}

// 获取当前时间/纳秒
func Nano() int64 {
	return time.Now().In(cst_sh).UnixNano()
}

// 截取字符串 start 起点下标 length 需要截取的长度
func Substr(str string, start int, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0
	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}
	return string(rs[start:end])
}

// 截取字符串 start 起点下标 end 终点下标(不包括)
func Substr2(str string, start int, end int) string {
	rs := []rune(str)
	length := len(rs)
	if start < 0 || start > length {
		return ""
	}
	if end < 0 || end > length {
		return ""
	}
	return string(rs[start:end])
}

// 获取本机内网IP
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Println(err)
		return ""
	}
	for _,
	address := range addrs { // 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// 判定指定字符串是否存在于原字符串
func HasStr(s1 string, s2 string) bool {
	if s1 == s2 {
		return true
	}
	if len(s1) > 0 && len(s2) > 0 && strings.Index(s1, s2) > -1 {
		return true
	}
	return false
}

// 高性能拼接字符串
func AddStr(input ...interface{}) string {
	if input == nil || len(input) == 0 {
		return ""
	}
	var rstr bytes.Buffer
	for e := range input {
		s := input[e]
		if v, b := s.(string); b {
			rstr.WriteString(v)
		} else if v, b := s.(error); b {
			rstr.WriteString(v.Error())
		} else if v, b := s.(bool); b {
			if v {
				rstr.WriteString("true")
			} else {
				rstr.WriteString("false")
			}
		} else {
			rstr.WriteString(AnyToStr(s))
		}
	}
	return rstr.String()
}

// 高性能拼接错误对象
func Error(input ...interface{}) error {
	msg := AddStr(input...)
	return errors.New(msg)
}

// 读取JSON格式配置文件
func ReadJsonConfig(conf []byte, result interface{}) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	err := json.Unmarshal(conf, result)
	if err != nil {
		log.Println("解析配置文件失败: " + err.Error())
		return err
	}
	return nil
}

// string转int
func StrToInt(str string) (int, error) {
	b, err := strconv.Atoi(str)
	if err != nil {
		return 0, errors.New("string转int失败")
	}
	return b, nil
}

// string转int8
func StrToInt8(str string) (int8, error) {
	b, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.New("string转int8失败")
	}
	return int8(b), nil
}

// string转int16
func StrToInt16(str string) (int16, error) {
	b, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.New("string转int16失败")
	}
	return int16(b), nil
}

// string转int32
func StrToInt32(str string) (int32, error) {
	b, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.New("string转int32失败")
	}
	return int32(b), nil
}

// string转int64
func StrToInt64(str string) (int64, error) {
	b, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errors.New("string转int64失败")
	}
	return b, nil
}

// 基础类型 int uint float string bool
// 复杂类型 json
func AnyToStr(any interface{}) string {
	if any == nil {
		return ""
	}
	if str, ok := any.(string); ok {
		return str
	} else if str, ok := any.(int); ok {
		return strconv.FormatInt(int64(str), 10)
	} else if str, ok := any.(int8); ok {
		return strconv.FormatInt(int64(str), 10)
	} else if str, ok := any.(int16); ok {
		return strconv.FormatInt(int64(str), 10)
	} else if str, ok := any.(int32); ok {
		return strconv.FormatInt(int64(str), 10)
	} else if str, ok := any.(int64); ok {
		return strconv.FormatInt(int64(str), 10)
	} else if str, ok := any.(float32); ok {
		return strconv.FormatFloat(float64(str), 'f', 0, 64)
	} else if str, ok := any.(float64); ok {
		return strconv.FormatFloat(float64(str), 'f', 0, 64)
	} else if str, ok := any.(uint); ok {
		return strconv.FormatUint(uint64(str), 10)
	} else if str, ok := any.(uint8); ok {
		return strconv.FormatUint(uint64(str), 10)
	} else if str, ok := any.(uint16); ok {
		return strconv.FormatUint(uint64(str), 10)
	} else if str, ok := any.(uint32); ok {
		return strconv.FormatUint(uint64(str), 10)
	} else if str, ok := any.(uint64); ok {
		return strconv.FormatUint(uint64(str), 10)
	} else if str, ok := any.(bool); ok {
		if str {
			return "True"
		}
		return "False"
	} else {
		if ret, err := ObjectToJson(any); err != nil {
			log.Println("any转json失败: ", err.Error())
			return ""
		} else {
			return ret
		}
	}
	return ""
}

// 深度复制对象
func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return Error("深度复制对象序列化异常: ", err.Error())
	}
	if err := gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst); err != nil {
		return Error("深度复制对象反序列异常: ", err.Error())
	}
	return nil
}

// 65-96大写字母 97-122小写字母
func UpperFirst(str string) string {
	var upperStr string
	vv := []rune(str) // 后文有介绍
	for i := 0; i < len(vv); i++ {
		if i == 0 {
			if vv[i] >= 97 && vv[i] <= 122 { // 小写字母范围
				vv[i] -= 32 // string的码表相差32位
				upperStr += string(vv[i])
			} else {
				return str
			}
		} else {
			upperStr += string(vv[i])
		}
	}
	return upperStr
}

// 65-96大写字母 97-122小写字母
func LowerFirst(str string) string {
	var upperStr string
	vv := []rune(str) // 后文有介绍
	for i := 0; i < len(vv); i++ {
		if i == 0 {
			if vv[i] >= 65 && vv[i] <= 96 { // 大写字母范围
				vv[i] += 32 // string的码表相差32位
				upperStr += string(vv[i])
			} else {
				return str
			}
		} else {
			upperStr += string(vv[i])
		}
	}
	return upperStr
}

// 获取雪花UUID,默认为0区
func GetUUID(sec ...int64) string {
	seed := int64(0)
	if sec != nil && len(sec) > 0 && sec[0] > 0 {
		seed = sec[0]
	}
	node, ok := snowflakes[seed];
	if !ok || node == nil {
		mu.Lock()
		node, ok = snowflakes[seed];
		if !ok || node == nil {
			node, _ = snowflake.NewNode(seed)
			snowflakes[seed] = node
		}
		mu.Unlock()
	}
	return node.Generate().String()
}

func GetUUIDInt64(sec ...int64) int64 {
	uuid := GetUUID(sec...)
	r, _ := StrToInt64(uuid)
	return r
}

// 校验是否跳过入库字段
func ValidIgnore(field reflect.StructField) bool {
	if field.Tag.Get(sqlc.Ignore) == sqlc.True {
		return true
	}
	return false
}

// 校验是否使用date类型
func ValidDate(field reflect.StructField) bool {
	if field.Tag.Get(sqlc.Date) == sqlc.True {
		return true
	}
	return false
}

// 读取文件
func ReadFile(path string) (string, error) {
	if len(path) == 0 {
		return "", Error("文件路径不能为空")
	}
	if b, err := ioutil.ReadFile(path); err != nil {
		return "", Error("读取文件[", path, "]失败: ", err.Error())
	} else {
		return string(b), nil
	}
}

// 读取本地JSON配置文件
func ReadLocalJsonConfig(path string, result interface{}) error {
	str, err := ReadFile(path)
	if err != nil {
		return err
	}
	if err := JsonToObject(str, result); err != nil {
		return err
	}
	return nil
}

func Pow(x, n int64) int64 {
	ret := int64(1) // 结果初始为0次方的值，整数0次方为1。如果是矩阵，则为单元矩阵。
	for n != 0 {
		if n%2 != 0 {
			ret = ret * x
		}
		n /= 2
		x = x * x
	}
	return ret
}

//只用于计算10的n次方，转换string
func PowString(n int) string {
	target := "1"
	for i := 0; i < n; i++ {
		target = AddStr(target, "0")
	}
	return target
}

// 转换成小数
func StrToFloat(str string) (float64, error) {
	if v, err := decimal.NewFromString(str); err != nil {
		return 0, err
	} else {
		r, _ := v.Float64()
		return r, nil
	}
}

// 获取JSON字段
func GetJsonTag(field reflect.StructField) (string, error) {
	json := field.Tag.Get("json")
	if json == "" {
		return "", errors.New("json配置异常")
	}
	return json, nil
}

// MD5加密
func MD5(s string, salt ...string) string {
	if len(salt) > 0 {
		s = salt[0] + s
	}
	has := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", has) //将[]byte转成16进制
}

// SHA256加密
func SHA256(s string, salt ...string) string {
	if len(salt) > 0 {
		s = salt[0] + s
	}
	h := sha256.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs) //将[]byte转成16进制
}

// default base64 - 正向
func Base64Encode(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// url base64 - 正向
func Base64URLEncode(input string) string {
	return base64.URLEncoding.EncodeToString([]byte(input))
}

// default base64 - 逆向
func Base64Decode(input string) string {
	if r, err := base64.StdEncoding.DecodeString(input); err != nil {
		return ""
	} else {
		return string(r)
	}
}

// url base64 - 逆向
func Base64URLDecode(input string) string {
	if r, err := base64.URLEncoding.DecodeString(input); err != nil {
		return ""
	} else {
		return string(r)
	}
}

// 随机获得6位数字
func Random6str() string {
	return fmt.Sprintf("%06v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
}

// 获取项目绝对路径
func GetPath() string {
	if path, err := os.Getwd(); err != nil {
		log.Println(err)
		return ""
	} else {
		return path
	}
}

// 获取当月份开始和结束时间
func GetMonthFirstAndLast() (int64, int64) {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, now.Location())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	return Time(firstOfMonth), Time(lastOfMonth) + 86400000 - 1
}

// 获取指定月份开始和结束时间
func GetAnyMonthFirstAndLast(month int) (int64, int64) {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	cmonth := int(currentMonth)
	offset := month - cmonth
	if month < 1 || month > 12 {
		offset = 0
	}
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, now.Location()).AddDate(0, offset, 0)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	return Time(firstOfMonth), Time(lastOfMonth) + 86400000 - 1
}

// 获取当周开始和结束时间
func GetWeekFirstAndLast() (int64, int64) {
	now := time.Now()
	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, offset)
	first := Time(start)
	return first, first + 604800000 - 1
}

// 获取当天开始和结束时间
func GetDayFirstAndLast() (int64, int64) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	first := Time(start)
	return first, first + 86400000 - 1
}
