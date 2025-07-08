package configutil

import (
	"bytes"
	"fmt"
	"github.com/gookit/config/v2"
	"reflect"
	"strings"
)

func LoadConfig(filename string, v interface{}) error {
	err := config.LoadFiles(filename)
	if err != nil {
		return err
	}
	err = config.Decode(v)
	if err != nil {
		return err
	}
	return nil
}

func InitDefaultConfig(cfg interface{}, indentLevel int) []byte {
	var yamlStr bytes.Buffer
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem() // 解引用指针
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		configTag, ok := fieldType.Tag.Lookup("config")
		if !ok {
			continue // 忽略没有config标签的字段
		}

		defaultTag := fieldType.Tag.Get("default")
		descriptionTag := fieldType.Tag.Get("description")

		// 添加注释
		if descriptionTag != "" {
			yamlStr.WriteString(fmt.Sprintf("%s# %s\n", strings.Repeat(" ", indentLevel*2), descriptionTag))
		}

		// 准备值
		valueStr := zeroValue(fieldType.Type.Kind(), defaultTag)
		// 根据字段类型进行处理
		switch field.Kind() {
		case reflect.Struct:
			yamlStr.WriteString(fmt.Sprintf("%s%s:\n%s", strings.Repeat(" ", indentLevel*2), configTag, InitDefaultConfig(field.Addr().Interface(), indentLevel+1)))
		case reflect.Ptr:
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			yamlStr.WriteString(fmt.Sprintf("%s%s:\n%s", strings.Repeat(" ", indentLevel*2), configTag, InitDefaultConfig(field.Interface(), indentLevel+1)))
		default:
			// 直接生成键值对
			yamlStr.WriteString(fmt.Sprintf("%s%s: %s\n", strings.Repeat(" ", indentLevel*2), configTag, valueStr))
		}
	}

	return yamlStr.Bytes()
}

// zeroValue 根据字段类型和default标签的值，准备最终的值字符串
func zeroValue(kind reflect.Kind, defaultVal string) string {
	if defaultVal != "" {
		return defaultVal
	}
	// 根据类型返回默认空值
	switch kind {
	case reflect.Bool:
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "0"
	case reflect.Float32, reflect.Float64:
		return "0.0"
	case reflect.Slice, reflect.Array:
		return "[]"
	case reflect.String:
		return `""`
	case reflect.Struct, reflect.Map:
		return "{}"
	default:
		return `""`
	}
}
