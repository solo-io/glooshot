package clilog

import (
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/solo-io/glooshot/pkg/pregoutils-clilog/internal/clibufferpool"
	"go.uber.org/zap/buffer"
	///Users/mitch/go/src/github.com/solo-io/glooshot/vendor/go.uber.org/zap/zapcore/memory_encoder.go:127
)

type cliEncoder struct {
}

func (c *cliEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error   { return nil }
func (c *cliEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error { return nil }

// Built-in types.
func (c *cliEncoder) AddBinary(key string, value []byte)          {} // for arbitrary bytes
func (c *cliEncoder) AddByteString(key string, value []byte)      {} // for UTF-8 encoded bytes
func (c *cliEncoder) AddBool(key string, value bool)              {}
func (c *cliEncoder) AddComplex128(key string, value complex128)  {}
func (c *cliEncoder) AddComplex64(key string, value complex64)    {}
func (c *cliEncoder) AddDuration(key string, value time.Duration) {}
func (c *cliEncoder) AddFloat64(key string, value float64)        {}
func (c *cliEncoder) AddFloat32(key string, value float32)        {}
func (c *cliEncoder) AddInt(key string, value int)                {}
func (c *cliEncoder) AddInt64(key string, value int64)            {}
func (c *cliEncoder) AddInt32(key string, value int32)            {}
func (c *cliEncoder) AddInt16(key string, value int16)            {}
func (c *cliEncoder) AddInt8(key string, value int8)              {}
func (c *cliEncoder) AddString(key, value string)                 {}
func (c *cliEncoder) AddTime(key string, value time.Time)         {}
func (c *cliEncoder) AddUint(key string, value uint)              {}
func (c *cliEncoder) AddUint64(key string, value uint64)          {}
func (c *cliEncoder) AddUint32(key string, value uint32)          {}
func (c *cliEncoder) AddUint16(key string, value uint16)          {}
func (c *cliEncoder) AddUint8(key string, value uint8)            {}
func (c *cliEncoder) AddUintptr(key string, value uintptr)        {}

// AddReflected uses reflection to serialize arbitrary objects, so it's slow
// and allocation-heavy.
func (c *cliEncoder) AddReflected(key string, value interface{}) error { return nil }

// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (c *cliEncoder) OpenNamespace(key string) {}

// NewConsoleEncoder creates an encoder whose output is designed for human -
// rather than machine - consumption. It serializes the core log entry data
// (message, level, timestamp, etc.) in a plain-text format and leaves the
// structured context as JSON.
//
// Note that although the console encoder doesn't use the keys specified in the
// encoder configuration, it will omit any element whose key is set to the empty
// string.
func NewCliEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &cliEncoder{}
}

func (c cliEncoder) Clone() zapcore.Encoder {
	return &cliEncoder{}
}

func (c cliEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := clibufferpool.Get()
	line.Write([]byte("hello there"))

	return line, nil
}
