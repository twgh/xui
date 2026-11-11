package ui

import "testing"

func TestQuestionnaireShortcutAssign(t *testing.T) {
	cases := []struct {
		name     string
		mode     QuestionnaireShortcuts
		disabled []bool
		want     []string
	}{
		{"none", QuestionnaireShortcutsNone, []bool{false, false}, []string{"", ""}},
		{"letters", QuestionnaireShortcutsLetters, []bool{false, false, false}, []string{"A", "B", "C"}},
		{"letters skip disabled", QuestionnaireShortcutsLetters, []bool{true, false, false}, []string{"", "A", "B"}},
		{"numbers", QuestionnaireShortcutsNumbers, []bool{false, false}, []string{"1", "2"}},
		{"numbers skip disabled", QuestionnaireShortcutsNumbers, []bool{false, true, false}, []string{"1", "", "2"}},
		{"numbers cap at 9", QuestionnaireShortcutsNumbers, make([]bool, 11), []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "", ""}},
	}
	for _, c := range cases {
		got := questionnaireShortcutAssign(c.mode, c.disabled)
		if len(got) != len(c.want) {
			t.Fatalf("%s: len = %d, want %d", c.name, len(got), len(c.want))
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: [%d] = %q, want %q", c.name, i, got[i], c.want[i])
			}
		}
	}
}

func TestQuestionnaireValidationRules(t *testing.T) {
	required := &questionnaireItem{def: QuestionnaireItem{Required: true}}
	if required.valid() {
		t.Error("required unanswered item should be invalid")
	}
	required.skipped = true
	if required.valid() {
		t.Error("required item cannot be skipped")
	}
	optional := &questionnaireItem{def: QuestionnaireItem{}}
	if optional.valid() {
		t.Error("optional unanswered item should be invalid until skipped")
	}
	optional.skipped = true
	if !optional.valid() {
		t.Error("optional skipped item should be valid")
	}
	disabled := &questionnaireItem{def: QuestionnaireItem{Disabled: true, Required: true}}
	if !disabled.valid() {
		t.Error("disabled item should be ignored by validation")
	}
	if got := required.errorText(); got != "Choose an answer to continue." {
		t.Errorf("required error = %q", got)
	}
	if got := optional.errorText(); got != "Choose an answer or skip this question." {
		t.Errorf("optional error = %q", got)
	}
}

func TestQuestionnaireKeyOf(t *testing.T) {
	if got := questionnaireKeyOf(QuestionnaireShortcutsLetters, 'A'); got != "A" {
		t.Errorf("letters A = %q", got)
	}
	if got := questionnaireKeyOf(QuestionnaireShortcutsLetters, '1'); got != "" {
		t.Errorf("letters 1 = %q", got)
	}
	if got := questionnaireKeyOf(QuestionnaireShortcutsNumbers, '9'); got != "9" {
		t.Errorf("numbers 9 = %q", got)
	}
	if got := questionnaireKeyOf(QuestionnaireShortcutsNumbers, '0'); got != "" {
		t.Errorf("numbers 0 = %q", got)
	}
	if got := questionnaireKeyOf(QuestionnaireShortcutsNone, 'A'); got != "" {
		t.Errorf("none A = %q", got)
	}
}
