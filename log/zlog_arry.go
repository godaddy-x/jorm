package log

import (
	"go.uber.org/zap"
	"time"

	"go.uber.org/zap/zapcore"
)

func Array(key string, val zapcore.ArrayMarshaler) zap.Field {
	return zap.Array(key, val)
}

func Bools(key string, bs []bool) zap.Field {
	return zap.Bools(key, bs)
}

func ByteStrings(key string, bss [][]byte) zap.Field {
	return zap.ByteStrings(key, bss)
}

func Complex128s(key string, nums []complex128) zap.Field {
	return zap.Complex128s(key, nums)
}

func Complex64s(key string, nums []complex64) zap.Field {
	return zap.Complex64s(key, nums)
}

func Durations(key string, ds []time.Duration) zap.Field {
	return zap.Durations(key, ds)
}

func Float64s(key string, nums []float64) zap.Field {
	return zap.Float64s(key, nums)
}

func Float32s(key string, nums []float32) zap.Field {
	return zap.Float32s(key, nums)
}

func Ints(key string, nums []int) zap.Field {
	return zap.Ints(key, nums)
}

func Int64s(key string, nums []int64) zap.Field {
	return zap.Int64s(key, nums)
}

func Int32s(key string, nums []int32) zap.Field {
	return zap.Int32s(key, nums)
}

func Int16s(key string, nums []int16) zap.Field {
	return zap.Int16s(key, nums)
}

func Int8s(key string, nums []int8) zap.Field {
	return zap.Int8s(key, nums)
}

func Strings(key string, ss []string) zap.Field {
	return zap.Strings(key, ss)
}

func Times(key string, ts []time.Time) zap.Field {
	return zap.Times(key, ts)
}

func Uints(key string, nums []uint) zap.Field {
	return zap.Uints(key, nums)
}

func Uint64s(key string, nums []uint64) zap.Field {
	return zap.Uint64s(key, nums)
}

func Uint32s(key string, nums []uint32) zap.Field {
	return zap.Uint32s(key, nums)
}

func Uint16s(key string, nums []uint16) zap.Field {
	return zap.Uint16s(key, nums)
}

func Uint8s(key string, nums []uint8) zap.Field {
	return zap.Uint8s(key, nums)
}

func Uintptrs(key string, us []uintptr) zap.Field {
	return zap.Uintptrs(key, us)
}

func AddError(errs ...error) zap.Field {
	return Errors("error", errs...)
}

func Errors(key string, errs ...error) zap.Field {
	return zap.Errors(key, errs)
}
