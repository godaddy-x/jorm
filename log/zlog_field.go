package log

import (
	"fmt"
	"go.uber.org/zap"
	"time"

	"go.uber.org/zap/zapcore"
)

func Skip() zap.Field {
	return zap.Skip()
}

func Binary(key string, val []byte) zap.Field {
	return zap.Binary(key, val)
}

func Bool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

func ByteString(key string, val []byte) zap.Field {
	return zap.ByteString(key, val)
}

func Complex128(key string, val complex128) zap.Field {
	return zap.Complex128(key, val)
}

func Complex64(key string, val complex64) zap.Field {
	return zap.Complex64(key, val)
}

func Float64(key string, val float64) zap.Field {
	return zap.Float64(key, val)
}

func Float32(key string, val float32) zap.Field {
	return zap.Float32(key, val)
}

func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

func Int32(key string, val int32) zap.Field {
	return zap.Int32(key, val)
}

func Int16(key string, val int16) zap.Field {
	return zap.Int16(key, val)
}

func Int8(key string, val int8) zap.Field {
	return zap.Int8(key, val)
}

func String(key string, val string) zap.Field {
	return zap.String(key, val)
}

func Uint(key string, val uint) zap.Field {
	return zap.Uint(key, val)
}

func Uint64(key string, val uint64) zap.Field {
	return zap.Uint64(key, val)
}

func Uint32(key string, val uint32) zap.Field {
	return zap.Uint32(key, val)
}

func Uint16(key string, val uint16) zap.Field {
	return zap.Uint16(key, val)
}

func Uint8(key string, val uint8) zap.Field {
	return zap.Uint8(key, val)
}

func Uintptr(key string, val uintptr) zap.Field {
	return zap.Uintptr(key, val)
}

func Reflect(key string, val interface{}) zap.Field {
	return zap.Reflect(key, val)
}

func Namespace(key string) zap.Field {
	return zap.Namespace(key)
}

func Stringer(key string, val fmt.Stringer) zap.Field {
	return zap.Stringer(key, val)
}

func Time(key string, val time.Time) zap.Field {
	return zap.Time(key, val)
}

func Stack(key string) zap.Field {
	return zap.Stack(key)
}

func Duration(key string, val time.Duration) zap.Field {
	return zap.Duration(key, val)
}

func Object(key string, val zapcore.ObjectMarshaler) zap.Field {
	return zap.Object(key, val)
}

func Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}