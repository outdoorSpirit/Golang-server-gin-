package lib

import (
	"fmt"

	"encoding/json"
)

type MaybeJson interface {
	Interface() interface{}

	Get(key string) MaybeJson

	At(index int) MaybeJson

	Iterate(func(interface{}, MaybeJson))

	String(def string) string

	AsString() (string, error)

	Int64(def int64) int64

	AsInt64() (int64, error)

	Float64(def float64) float64

	AsFloat64() (float64, error)

	Bool(def bool) bool

	AsBool() (bool, error)

	IsNull() bool

	IsEmpty() bool

	IsValid() bool
}

func AsJson(j interface{}) MaybeJson {
	if j == nil {
		return jsonNull{jsonEmpty{}}
	}

	switch v := j.(type) {
	case bool:
		return jsonBool{v, jsonEmpty{}}
	case float64:
		return jsonNumber{v, jsonEmpty{}}
	case string:
		return jsonString{v, jsonEmpty{}}
	case []interface{}:
		return jsonArray{v, jsonEmpty{}}
	case map[string]interface{}:
		return jsonObject{v, jsonEmpty{}}
	default:
		return jsonEmpty{}
	}
}

type jsonObject struct {
	object map[string]interface{}
	jsonEmpty
}

func (j jsonObject) Interface() interface{} {
	return j.object
}

func (j jsonObject) Get(key string) MaybeJson {
	if v, has := j.object[key]; has {
		return AsJson(v)
	} else {
		return jsonEmpty{}
	}
}

func (j jsonObject) Iterate(f func(interface{}, MaybeJson)) {
	for k, v := range j.object {
		f(k, AsJson(v))
	}
}

func (j jsonObject) IsEmpty() bool {
	return len(j.object) == 0
}

func (j jsonObject) IsValid() bool {
	return true
}

type jsonArray struct {
	array []interface{}
	jsonEmpty
}

func (j jsonArray) Interface() interface{} {
	return j.array
}

func (j jsonArray) At(index int) MaybeJson {
	if len(j.array) > index {
		return AsJson(j.array[index])
	} else {
		return jsonEmpty{}
	}
}

func (j jsonArray) Iterate(f func(interface{}, MaybeJson)) {
	for i, v := range j.array {
		f(i, AsJson(v))
	}
}

func (j jsonArray) IsEmpty() bool {
	return len(j.array) == 0
}

func (j jsonArray) IsValid() bool {
	return true
}

type jsonBool struct {
	value bool
	jsonEmpty
}

func (j jsonBool) Interface() interface{} {
	return j.value
}

func (j jsonBool) Bool(def bool) bool {
	return j.value
}

func (j jsonBool) AsBool() (bool, error) {
	return j.value, nil
}

func (j jsonBool) IsEmpty() bool {
	return false
}

func (j jsonBool) IsValid() bool {
	return true
}

type jsonNumber struct {
	value float64
	jsonEmpty
}

func (j jsonNumber) Interface() interface{} {
	return j.value
}

func (j jsonNumber) Int64(def int64) int64 {
	return int64(j.value)
}

func (j jsonNumber) AsInt64() (int64, error) {
	return int64(j.value), nil
}

func (j jsonNumber) Float64(def float64) float64 {
	return j.value
}

func (j jsonNumber) AsFloat64() (float64, error) {
	return j.value, nil
}

func (j jsonNumber) IsEmpty() bool {
	return false
}

func (j jsonNumber) IsValid() bool {
	return true
}

type jsonString struct {
	value string
	jsonEmpty
}

func (j jsonString) Interface() interface{} {
	return j.value
}

func (j jsonString) String(def string) string {
	return j.value
}

func (j jsonString) AsString() (string, error) {
	return j.value, nil
}

func (j jsonString) IsEmpty() bool {
	return false
}

func (j jsonString) IsValid() bool {
	return true
}

// jsonのnull。
type jsonNull struct {
	jsonEmpty
}

func (j jsonNull) IsNull() bool {
	return true
}

func (j jsonNull) IsEmpty() bool {
	return true
}

func (j jsonNull) IsValid() bool {
	return true
}

type jsonEmpty struct {
}

func (j jsonEmpty) Interface() interface{} {
	return nil
}

func (j jsonEmpty) Get(key string) MaybeJson {
	return jsonEmpty{}
}

func (j jsonEmpty) At(index int) MaybeJson {
	return jsonEmpty{}
}

func (j jsonEmpty) Iterate(f func(interface{}, MaybeJson)) {
}

func (j jsonEmpty) String(def string) string {
	return def
}

func (j jsonEmpty) AsString() (string, error) {
	return "", fmt.Errorf("This element is not a string")
}

func (j jsonEmpty) Int64(def int64) int64 {
	return def
}

func (j jsonEmpty) AsInt64() (int64, error) {
	return 0, fmt.Errorf("This element is not a integer")
}

func (j jsonEmpty) Float64(def float64) float64 {
	return def
}

func (j jsonEmpty) AsFloat64() (float64, error) {
	return 0, fmt.Errorf("This element is not a number")
}

func (j jsonEmpty) Bool(def bool) bool {
	return def
}

func (j jsonEmpty) AsBool() (bool, error) {
	return false, fmt.Errorf("This element is not a boolean")
}

func (j jsonEmpty) IsNull() bool {
	return false
}

func (j jsonEmpty) IsEmpty() bool {
	return true
}

func (j jsonEmpty) IsValid() bool {
	return false
}

func UnmarshalToMaybeJson(bytes []byte) (MaybeJson, error) {
	var body interface{}

	err := json.Unmarshal(bytes, &body)
	if err != nil {
		return nil, err
	}

	maybeJson := AsJson(body)

	return maybeJson, nil
}
