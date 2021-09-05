package main

import (
	"testing"
)

func TestNilRemovalFromArray(t *testing.T) {
	avar := "test"
	value := []interface{}{
		nil, 1, "string", readLines, nil, &avar, nil, nil,
	}
	expected := []interface{}{
		1, "string", readLines, &avar,
	}
	transformer := NilRemovalTransformer{
		RemoveNilKeys:     false,
		RemoveNilValues:   false,
		RemoveNilElements: true,
	}

	tvalue, err := transformer.Transform(value)
	if err != nil {
		t.Error(err)
	}
	actual := tvalue.([]interface{})
	if len(actual) != len(expected) ||
		actual[0] != expected[0] ||
		actual[1] != expected[1] ||
		actual[3] != expected[3] {
		t.Errorf("actual and expected array nil removal do not match: %v vs %v", actual, expected)
	}
}

func TestNilKeyRemovalFromMap(t *testing.T) {
	value := map[interface{}]interface{}{
		nil: 1,
		2:   nil,
		"3": 4,
	}
	expected := map[interface{}]interface{}{
		2:   nil,
		"3": 4,
	}
	transformer := NilRemovalTransformer{
		RemoveNilKeys:     true,
		RemoveNilValues:   false,
		RemoveNilElements: false,
	}

	tvalue, err := transformer.Transform(value)
	if err != nil {
		t.Error(err)
	}
	actual := tvalue.(map[interface{}]interface{})
	if len(actual) != len(expected) ||
		actual[2] != expected[2] ||
		actual["3"] != expected["3"] {
		t.Errorf("actual and expected map key nil removal do not match: %v vs %v", actual, expected)
	}

}
func TestNilValueRemovalFromMap(t *testing.T) {
	value := map[interface{}]interface{}{
		nil: 1,
		2:   nil,
		"3": 4,
	}
	expected := map[interface{}]interface{}{
		nil: 1,
		"3": 4,
	}
	transformer := NilRemovalTransformer{
		RemoveNilKeys:     false,
		RemoveNilValues:   true,
		RemoveNilElements: false,
	}
	tvalue, err := transformer.Transform(value)
	if err != nil {
		t.Error(err)
	}
	actual := tvalue.(map[interface{}]interface{})
	if len(actual) != len(expected) ||
		actual[nil] != expected[nil] ||
		actual["3"] != expected["3"] {
		t.Errorf("actual and expected array nil removal do not match: %v vs %v", actual, expected)
	}

}
func TestNilRemovalFromMap(t *testing.T) {
	value := map[interface{}]interface{}{
		nil: 1,
		2:   nil,
		"3": 4,
	}
	expected := map[interface{}]interface{}{
		"3": 4,
	}
	transformer := NilRemovalTransformer{
		RemoveNilKeys:     true,
		RemoveNilValues:   true,
		RemoveNilElements: false,
	}

	tvalue, err := transformer.Transform(value)
	if err != nil {
		t.Error(err)
	}
	actual := tvalue.(map[interface{}]interface{})
	if len(actual) != len(expected) ||
		actual["3"] != expected["3"] {
		t.Errorf("actual and expected array nil removal do not match: %v vs %v", actual, expected)
	}
}

func TestTrivialNilRemoval(t *testing.T) {
	transformer := NilRemovalTransformer{}

	val, err := transformer.Transform(nil)
	if val != nil || err != nil {
		t.Error("nil not transformed into itself")
	}
}

func TestRecursiveNilRemoval(t *testing.T) {
	transformer := NilRemovalTransformer{
		RemoveNilKeys:     true,
		RemoveNilValues:   true,
		RemoveNilElements: true,
	}
	val, err := transformer.Transform(map[interface{}]interface{}{"a": nil, nil: "b", "c": []interface{}{nil, 1}})
	if err != nil {
		t.Errorf("nil not transformed into itself: %s", err)
	}
	actual := val.(map[interface{}]interface{})
	if len(actual) != 1 {
		t.Errorf("incorrect map nil transformation detected: %v", val)
	}
	if len(actual["c"].([]interface{})) != 1 || actual["c"].([]interface{})[0] != 1 {
		t.Errorf("incorrect recursive array nil transformation detected: %v", val)
	}
}
