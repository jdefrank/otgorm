package otgorm

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

//CreateSpanAttribute creates a KeyValue for use as a span attribute
func CreateSpanAttribute(k string, v interface{}) (kv attribute.KeyValue) {
	switch x := v.(type) {
	case string:
		return attribute.Key(k).String(v.(string))
	case bool:
		return attribute.Key(k).Bool(v.(bool))
	case int64:
		return attribute.Key(k).Int64(v.(int64))
	case int32:
		return attribute.Key(k).Int64(int64(v.(int32)))
	case int:
		return attribute.Key(k).Int(v.(int))
	case float64:
		return attribute.Key(k).Float64(v.(float64))
	case float32:
		return attribute.Key(k).Float64(float64(v.(float32)))
	case uint:
		return attribute.Key(k).Int(int(v.(uint)))
	case uint64:
		return attribute.Key(k).Int64(v.(int64))
	case uint32:
		return attribute.Key(k).Int64(int64(v.(int32)))
	default:
		return attribute.Key("attribute.error").String(fmt.Sprintf("Couldn't convert %s into KeyValue", x))
	}
}
