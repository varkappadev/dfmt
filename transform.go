package main

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// A transformer accepts arbitrary data and applies some rules to it.
//
// If the data is modifiable (such as a map), it may do so directly.
// If this is not desirable, a deep copy should be passed instead.
type Transformer interface {
	Transform(interface{}) (interface{}, error)
}

// A nop transformer -- doing nothing by design.
type NopTransformer struct{}

func (t NopTransformer) Transform(data interface{}) (interface{}, error) {
	return data, nil
}

// A transformer removing certain nil keys and values from maps or elements from slices.
type NilRemovalTransformer struct {
	RemoveNilKeys     bool
	RemoveNilValues   bool
	RemoveNilElements bool
}

func (t NilRemovalTransformer) Transform(data interface{}) (interface{}, error) {
	if data == nil {
		return data, nil
	}
	var sliceSelector SliceSelector
	if t.RemoveNilElements {
		sliceSelector = NonNilElementSelector
	} else {
		sliceSelector = nil // nop indicator
	}
	var kvSelector KeyValueSelector
	if t.RemoveNilKeys && t.RemoveNilValues {
		kvSelector = NonNilKeyValueSelector
	} else if t.RemoveNilKeys {
		kvSelector = NonNilKeySelector
	} else if t.RemoveNilValues {
		kvSelector = NonNilValueSelector
	} else {
		kvSelector = nil // nop indicator
	}
	cTransformer := NewConfigurableTransformer(nil, nil, nil, sliceSelector, kvSelector)
	return cTransformer.Transform(data)
}

// Modifies or converts strings and returns either the original string or the modified one.
type StringConverter func(s string) interface{}

// Modifies or converts double-precision floats and returns the original or modified one.
type Float64Converter func(f float64) interface{}

// Modifies or converts complex numbers and returns the original or modified one.
type Complex128Converter func(c complex128) interface{}

// Returns true if the element should be kept in the slice/array.
type SliceSelector func(element interface{}) bool

// Returns true if the key/value pair should be kept in the map.
type KeyValueSelector func(key interface{}, value interface{}) bool

// An internal customisable transformer that accepts selector and converter functions
// and applies them to the input.
type callingTransformer struct {
	stringTransformer     StringConverter
	float64Transformer    Float64Converter
	complex128Transformer Complex128Converter
	sliceSelector         SliceSelector
	kvSelector            KeyValueSelector
}

func (t callingTransformer) Transform(data interface{}) (interface{}, error) {
	if data == nil {
		return data, nil
	}
	return t.transformInterface(data)
}

func (t callingTransformer) transformInterface(data interface{}) (interface{}, error) {
	switch d := data.(type) {
	case string:
		return t.stringTransformer(d), nil
	case float64:
		return t.float64Transformer(d), nil
	case float32:
		return t.float64Transformer(float64(d)), nil
	case complex128:
		return t.complex128Transformer(d), nil
	case complex64:
		return t.complex128Transformer(complex128(d)), nil
	default:
		if isNil(data) {
			return nil, nil
		}
		itype := reflect.TypeOf(data)
		switch itype.Kind() {
		case reflect.Map:
			return t.transformMap(reflect.ValueOf(data))
		case reflect.Slice, reflect.Array:
			return t.transformSlice(data.([]interface{}))
		default:
			return data, nil
		}
	}
}

func (t callingTransformer) transformMap(data reflect.Value) (interface{}, error) {
	if data.Kind() != reflect.Map {
		return nil, fmt.Errorf("transformMap was unexpectedly invoked on something other than a map")
	}

	keys := data.MapKeys()
	for _, k := range keys {
		if isNil(data.MapIndex(k).Interface()) {
			continue // do not remove nil values here by accident
		}
		d, err := t.transformInterface(data.MapIndex(k).Interface())
		if err != nil {
			return data.Interface(), err
		}
		data.SetMapIndex(k, reflect.ValueOf(d))
	}

	for _, k := range data.MapKeys() {
		v := data.MapIndex(k)
		if !t.kvSelector(k.Interface(), v.Interface()) {
			data.SetMapIndex(k, reflect.Value{})
		}
	}
	return data.Interface(), nil
}

func (t callingTransformer) transformSlice(data []interface{}) (interface{}, error) {
	if isNil(data) {
		return data, nil
	}
	for n := 0; n < len(data); n++ {
		d, err := t.transformInterface(data[n])
		if err != nil {
			return data, err
		}
		data[n] = d
	}
	for n := 0; n < len(data); n++ {
		if !t.sliceSelector(data[n]) {
			if n == 0 {
				data = data[1:]
			} else if n == (len(data) - 1) {
				data = data[:n]
			} else {
				data = append(data[:n], data[n+1:]...)
			}
			n--
		}
	}
	return data, nil
}

// A transformer that applies other transformers in sequence.
type TransformerPipeline struct {
	Transformers []Transformer
}

func (m TransformerPipeline) Transform(value interface{}) (interface{}, error) {
	var err error
	for _, t := range m.Transformers {
		if t == nil {
			continue // ignore silently
		}
		value, err = t.Transform(value)
		if err != nil {
			return value, err
		}
	}
	return value, nil
}

func NewMultiTransformer(transformers ...Transformer) Transformer {
	var transformer TransformerPipeline = TransformerPipeline{
		Transformers: make([]Transformer, 0),
	}
	for _, t := range transformers {
		if t != nil {
			transformer.Transformers = append(transformer.Transformers, t)
		}
	}
	return transformer
}

func NewConfigurableTransformer(s StringConverter, f Float64Converter, c Complex128Converter,
	es SliceSelector, kv KeyValueSelector) Transformer {
	if s == nil && f == nil && c == nil && es == nil && kv == nil {
		return NopTransformer{}
	}

	transformer := callingTransformer{}
	if s == nil {
		transformer.stringTransformer = func(s string) interface{} { return s }
	} else {
		transformer.stringTransformer = s
	}

	if f == nil {
		transformer.float64Transformer = func(f float64) interface{} { return f }
	} else {
		transformer.float64Transformer = f
	}

	if c == nil {
		transformer.complex128Transformer = func(c complex128) interface{} { return c }
	} else {
		transformer.complex128Transformer = c
	}

	if es == nil {
		transformer.sliceSelector = func(element interface{}) bool { return true }
	} else {
		transformer.sliceSelector = es
	}

	if kv == nil {
		transformer.kvSelector = func(key, value interface{}) bool { return true }
	} else {
		transformer.kvSelector = kv
	}

	return transformer
}

func NonNilElementSelector(element interface{}) bool {
	return !isNil(element)
}

func NonEmptySliceOrArrayAsElementSelector(element interface{}) bool {
	value := reflect.ValueOf(element)
	switch value.Kind() {
	case reflect.Array, reflect.Slice:
		return value.Len() > 0
	default:
		return true
	}
}

func NonNilKeySelector(key interface{}, value interface{}) bool {
	return !isNil(key)
}

func NonNilValueSelector(key interface{}, value interface{}) bool {
	return !isNil(value)
}

func NonNilKeyValueSelector(key interface{}, value interface{}) bool {
	return !isNil(key) && !isNil(value)
}

func StringToFiniteNumberParser(s string) interface{} {
	return CustomStringNumberParser(s, 64, 64, true)
}

func CustomStringNumberParser(s string, intbits int, floatbits int, finiteOnly bool) interface{} {
	if s == "" {
		return s
	}
	i, err := strconv.ParseInt(s, 10, intbits)
	if err == nil {
		return i
	}
	f, err := strconv.ParseFloat(s, floatbits)
	if err != nil {
		return s
	} else if !finiteOnly {
		return f
	} else if !(math.IsInf(f, 0) || math.IsNaN(f)) {
		return f
	} else {
		return s
	}
}
