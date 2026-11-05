package ui

import (
	"reflect"
	"testing"
)

// 标签的解析 / 增删是纯计算, 不需要 GUI, 直接测。
func TestTagInputDrainSeparators(t *testing.T) {
	ti := &TagInput{separators: ",，\n"}
	cases := []struct {
		in       string
		want     []string
		wantRest string
	}{
		{"", nil, ""},
		{"abc", nil, "abc"},
		{"a,b,c", []string{"a", "b"}, "c"},
		{"a，b，c", []string{"a", "b"}, "c"},
		{"a\nb\nc", []string{"a", "b"}, "c"}, // 粘贴多行
		{"a,", []string{"a"}, ""},
		{",a", []string{""}, "a"},
		{"a,,b", []string{"a", ""}, "b"},
	}
	for _, c := range cases {
		parts, rest := ti.drainSeparators(c.in)
		if rest != c.wantRest {
			t.Errorf("drainSeparators(%q) rest = %q, want %q", c.in, rest, c.wantRest)
		}
		if len(parts) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(parts, c.want) {
			t.Errorf("drainSeparators(%q) parts = %q, want %q", c.in, parts, c.want)
		}
	}

	// 多字节分隔符必须按 rune 宽度挪, 否则会在 rest 里留下半个汉字
	if parts, rest := ti.drainSeparators("标签，剩余"); len(parts) != 1 || rest != "剩余" {
		t.Errorf("中文逗号切分 = %q / %q", parts, rest)
	}

	closed := &TagInput{separators: ""}
	if parts, rest := closed.drainSeparators("a,b"); len(parts) != 0 || rest != "a,b" {
		t.Errorf("关闭分隔符后不该切分: %q / %q", parts, rest)
	}
}

func TestTagInputAdd(t *testing.T) {
	ti := &TagInput{}
	added, rejected := ti.add("  设计  ")
	if !added || rejected || !reflect.DeepEqual(ti.tags, []string{"设计"}) {
		t.Fatalf("add 应 trim 后加入: %v %v %q", added, rejected, ti.tags)
	}
	if added, rejected := ti.add("   "); added || rejected {
		t.Error("空白串应静默忽略, 不算拒绝")
	}
	if added, rejected := ti.add("设计"); added || rejected {
		t.Error("默认应精确匹配去重, 且不算拒绝")
	}
	if added, _ := ti.add("设计 "); added {
		t.Error("trim 后重复也应被去重")
	}

	dup := &TagInput{allowDup: true}
	dup.add("a")
	if added, _ := dup.add("a"); !added {
		t.Error("WithTagInputAllowDuplicates 后应能重复")
	}

	limited := &TagInput{max: 2}
	limited.add("a")
	limited.add("b")
	if added, rejected := limited.add("c"); added || !rejected {
		t.Error("达到上限应拒绝并标记 rejected")
	}
	if len(limited.tags) != 2 {
		t.Errorf("上限外不应追加: %q", limited.tags)
	}

	validated := &TagInput{validator: func(s string) bool { return len(s) >= 2 }}
	if added, rejected := validated.add("x"); added || !rejected {
		t.Error("校验器拒绝应标记 rejected")
	}
	if added, _ := validated.add("xx"); !added {
		t.Error("校验器通过应加入")
	}
}

func TestTagInputTagsAndRemove(t *testing.T) {
	ti := &TagInput{}
	ti.add("a")
	ti.add("b")
	ti.add("c")
	got := ti.Tags()
	got[0] = "改过了"
	if ti.tags[0] != "a" {
		t.Error("Tags() 必须返回副本")
	}
	if !ti.remove(1) || !reflect.DeepEqual(ti.tags, []string{"a", "c"}) {
		t.Errorf("remove(1) = %q", ti.tags)
	}
	if ti.remove(9) || ti.remove(-1) {
		t.Error("越界索引应返回 false")
	}
}
