package rds

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/spiker/spiker-server/model"
)

func _prefixed(obj interface{}, tableName string, prefix string) []string {
	columns := []string{}

	v := reflect.ValueOf(obj)
	t := v.Type()

	var x time.Time
	var j model.JSON

	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		valueType := v.Field(i)
		if valueType.Kind() == reflect.Ptr {
			ptrType := t.Field(i).Type.Elem()
			if ptrType.Kind() != reflect.Struct ||
				ptrType == reflect.TypeOf(x) {
				columnName := strcase.ToSnake(name)
				columns = append(columns, fmt.Sprintf("%s.%s AS %s_%s", tableName, columnName, prefix, columnName))
			}
		} else if (valueType.Kind() != reflect.Struct && valueType.Kind() != reflect.Slice) || (valueType.Type() == reflect.TypeOf(x) || valueType.Type() == reflect.TypeOf(j)) {
			// 構造体以外 または 時間の構造体以外はカラムとみなす。
			columnName := strcase.ToSnake(name)
			columns = append(columns, fmt.Sprintf("%s.%s AS %s_%s", tableName, columnName, prefix, columnName))
		}
	}
	return columns
}

func prefixColumns(obj interface{}, tableName string, prefix string) string {
	return strings.Join(_prefixed(obj, tableName, prefix), ",")
}

func escapeLike(value string) string {
	escaped := ""

	for _, r := range value {
		switch r {
		case '\\':
			escaped += "\\\\\\\\"
		case '%':
			escaped += "\\%"
		case '_':
			escaped += "\\_"
		default:
			escaped += string(r)
		}
	}

	return escaped
}

type incrementalPlaceholder struct {
	Index int
}

func (holder *incrementalPlaceholder) Generate(cond string) string {
	holder.Index = holder.Index + 1
	return fmt.Sprintf("%s $%d", cond, holder.Index)
}

func (holder *incrementalPlaceholder) GetIndex() int {
	holder.Index = holder.Index + 1
	return holder.Index
}

func safeRowsIterator(rows *sql.Rows, scanFunc func(*sql.Rows)) {
	defer rows.Close()
	for rows.Next() {
		scanFunc(rows)
	}
}

type reflectField struct {
	Field reflect.StructField
	Value reflect.Value
}

func scanRows(db model.QueryExecutor, rows *sql.Rows, parameter interface{}, prefix string) error {
	fields := map[string]reflectField{}
	resetFields := map[int]reflectField{}

	v := reflect.ValueOf(parameter).Elem()

	// ターゲットとなるモデルのDBのカラム名とマッピングする値の作成。
	for i := 0; i < v.NumField(); i++ {
		valueField := v.Field(i)
		typeField := v.Type().Field(i)
		tagName := typeField.Tag.Get("db")
		tagNames := strings.Split(tagName, ",")
		if tagNames[0] != "-" {
			name := strings.TrimSpace(tagNames[0])
			if len(prefix) > 0 {
				name = prefix + "_" + tagNames[0]
			}
			fields[name] = reflectField{Field: typeField, Value: valueField}
		}
	}

	// 取得済みの列情報とScanする為に型を代入する。
	columns, _ := rows.Columns()
	var ignored interface{}
	var values = make([]interface{}, len(columns))
	for index, column := range columns {
		values[index] = &ignored
		if field, ok := fields[column]; ok {
			if field.Value.Kind() == reflect.Ptr {
				values[index] = field.Value.Addr().Interface()
			} else {
				reflectValue := reflect.New(reflect.PtrTo(field.Field.Type))
				values[index] = reflectValue.Interface()
				resetFields[index] = field
			}
		}
	}
	rows.Scan(values...)

	// 各要素に戻す。
	for index, field := range resetFields {
		if v := reflect.ValueOf(values[index]).Elem().Elem(); v.IsValid() {
			if field.Value.CanSet() {
				field.Value.Set(v)
			}
		}
	}

	return nil
}

// 条件節とプレースホルダパラメータから構成される条件インタフェース。
type Conditional interface {
	Clause() string
	Params() []interface{}
}

type interfaces struct {
	values []interface{}
}

func newInterfaces(values ...interface{}) *interfaces {
	return &interfaces{values}
}

func (i *interfaces) clone() *interfaces {
	c := &interfaces{[]interface{}{}}
	return c.add(i.values...)
}

func (i *interfaces) add(values ...interface{}) *interfaces {
	i.values = append(i.values, values...)
	return i
}

func (i *interfaces) prepend(values ...interface{}) *interfaces {
	i.values = append(values, i.values...)
	return i
}

type aCondition struct {
	clause string
	params []interface{}
}

func (c aCondition) Clause() string {
	return c.clause
}

func (c aCondition) Params() []interface{} {
	return c.params
}

type querying struct {
	conditions []Conditional
	allOrAny   bool
}

func andQuery() *querying {
	return &querying{[]Conditional{}, true}
}

func orQuery() *querying {
	return &querying{[]Conditional{}, false}
}

func (q querying) Clause() string {
	clauses := []string{}

	if len(q.conditions) == 1 {
		clauses = append(clauses, q.conditions[0].Clause())
	} else {
		for _, cnd := range q.conditions {
			clauses = append(clauses, fmt.Sprintf("(%s)", cnd.Clause()))
		}
	}

	op := ""
	if q.allOrAny {
		op = " AND "
	} else {
		op = " OR "
	}

	return strings.Join(clauses, op)
}

func (q querying) Params() []interface{} {
	params := []interface{}{}

	for _, cnd := range q.conditions {
		params = append(params, cnd.Params()...)
	}

	return params
}

func (q *querying) add(clause string, params ...interface{}) *querying {
	q.conditions = append(q.conditions, &aCondition{clause, params})
	return q
}

func (q *querying) addCondition(cond Conditional) *querying {
	return q.add(cond.Clause(), cond.Params()...)
}

func (q querying) where() (string, *interfaces) {
	clause := q.Clause()
	params := q.Params()

	if len(clause) > 0 {
		return fmt.Sprintf("WHERE %s", clause), newInterfaces(params...)
	} else {
		return "", newInterfaces(params...)
	}
}
