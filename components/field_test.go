package ui

import (
	"errors"
	"reflect"
	"testing"
)

// 校验消息收集是纯逻辑, 不需要 GUI; 单独测必填/空白/校验器顺序与跳过规则。
func TestFieldValidationMessages(t *testing.T) {
	min3 := func(v string) error {
		if len([]rune(v)) < 3 {
			return errors.New("至少 3 个字符")
		}
		return nil
	}

	cases := []struct {
		name       string
		value      string
		required   bool
		requiredMs string
		validators []FieldValidator
		want       []string
	}{
		{"必填为空", "", true, "", nil, []string{fieldRequiredMessage}},
		{"必填仅空白", "   ", true, "", nil, []string{fieldRequiredMessage}},
		{"必填自定义文案", "", true, "请填写", nil, []string{"请填写"}},
		{"非必填空值通过", "", false, "", []FieldValidator{min3}, []string{"至少 3 个字符"}},
		{"必填与校验器同时失败", "", true, "请填写", []FieldValidator{min3}, []string{"请填写", "至少 3 个字符"}},
		{"通过", "twgh", true, "", []FieldValidator{min3, nil}, nil},
		{"空 error 文案跳过", "twgh", false, "", []FieldValidator{func(string) error {
			return errors.New("")
		}}, nil},
	}
	for _, c := range cases {
		got := fieldValidationMessages(c.value, c.required, c.requiredMs, c.validators)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: fieldValidationMessages() = %v, want %v", c.name, got, c.want)
		}
	}
}