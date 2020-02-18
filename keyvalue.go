package otgorm

import (
	"fmt"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
)

//CreateSpanAttribute creates a KeyValue for use as a span attribute
func CreateSpanAttribute(k string, v interface{}) (kv core.KeyValue) {
	switch x := v.(type) {
	case string:
		return key.New(k).String(v.(string))
	case bool:
		return key.New(k).Bool(v.(bool))
	case int64:
		return key.New(k).Int64(v.(int64))
	case int32:
		return key.New(k).Int32(v.(int32))
	case int:
		return key.New(k).Int(v.(int))
	case float64:
		return key.New(k).Float64(v.(float64))
	case float32:
		return key.New(k).Float32(v.(float32))
	case uint:
		return key.New(k).Uint(v.(uint))
	case uint64:
		return key.New(k).Uint64(v.(uint64))
	case uint32:
		return key.New(k).Uint32(v.(uint32))
	default:
		return key.New("attribute.error").String(fmt.Sprintf("Couldn't convert %s into KeyValue", x))
	}
}
