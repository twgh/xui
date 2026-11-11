package main

import (
	"sort"
	"strings"

	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xui"
	ui "github.com/twgh/xui/components"
)

// ===== 页 9: 表单6(Questionnaire 多步问卷) =====

func buildForm6Page(body *widget.LayoutEle) {
	r := row(body.Handle, 0, gap)
	colL := xui.VStack(r.Handle, 1, gap)
	colR := xui.VStack(r.Handle, 1, gap)

	buildQuestionnaireMain(colL.Handle)
	buildQuestionnaireSkip(colR.Handle)
}

// questionnaireStatus 建一个随内容折行的状态标签(提交/状态变化时写结果)。
func questionnaireNote(parent int, text string) *ui.Label {
	l := ui.NewLabel(0, 0, 0, 20, text, parent).SetMuted().SetAutoWrap()
	xui.ItemFill(l)
	return l
}

func questionnaireStatus(parent int) *ui.Label {
	l := ui.NewLabel(0, 0, 0, 20, "答案会显示在这里。", parent).SetMuted().SetAutoWrap()
	xui.ItemFill(l)
	return l
}

func joinQuestionnaireAnswers(answers map[string][]string) string {
	if len(answers) == 0 {
		return "未作答"
	}
	keys := make([]string, 0, len(answers))
	for k := range answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+strings.Join(answers[k], ", "))
	}
	return strings.Join(parts, " · ")
}

// buildQuestionnaireMain 多步问卷: 单选(带说明) / 多选 / 自由输入 / 校验 / 快捷键 / 进度。
func buildQuestionnaireMain(parent int) {
	_, v := autoCard(parent, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "多步问卷(Questionnaire)", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"多步 / 单选 / 多选 / 自由输入 / 校验 / 字母快捷键", v.Handle))

	items := []ui.QuestionnaireItem{
		{
			Name:        "direction",
			Title:       "What should the agent build next?",
			Description: "Choose a direction or describe another task.",
			Required:    true,
			Choices: []ui.QuestionnaireChoice{
				{Value: "tool-calls", Label: "Tool call timeline", Description: "Show what the agent ran and what came back."},
				{Value: "approvals", Label: "Approval checkpoints", Description: "Ask before sensitive or destructive actions."},
				{Value: "handoffs", Label: "Sub-agent handoffs", Description: "Make delegated work and results easier to follow."},
			},
			Input: &ui.QuestionnaireInput{Placeholder: "Describe another feature…"},
		},
		{
			Name:        "signals",
			Title:       "What should every progress update include?",
			Description: "Select all that apply, or skip this question.",
			Multiple:    true,
			Choices: []ui.QuestionnaireChoice{
				{Value: "progress", Label: "Progress"},
				{Value: "decisions", Label: "Decisions"},
				{Value: "risks", Label: "Risks"},
				{Value: "next-step", Label: "Next step"},
			},
		},
		{
			Name:        "timing",
			Title:       "When should work begin?",
			Description: "Choose when the agent should begin the work.",
			Required:    true,
			Choices: []ui.QuestionnaireChoice{
				{Value: "now", Label: "Start now"},
				{Value: "next-cycle", Label: "Next development cycle"},
				{Value: "backlog", Label: "Add it to the backlog"},
			},
		},
	}

	q := ui.NewQuestionnaire(0, 0, 0, 0, v.Handle, items,
		ui.WithQuestionnaireShortcuts(ui.QuestionnaireShortcutsLetters),
		ui.WithQuestionnaireDefaultItem("direction"),
		ui.WithQuestionnaireActionLabels("", "", "Next", "Save plan"))
	xui.ItemFill(q)
	status := questionnaireStatus(v.Handle)
	q.OnSubmit(func(answers map[string][]string) {
		status.SetText("已提交 · " + joinQuestionnaireAnswers(answers))
		status.Redraw(false)
	})
	q.OnStatusChange(func(item string, st ui.QuestionnaireItemStatus) {
		status.SetText("状态 · " + item + " = " + st.String())
		status.Redraw(false)
	})

	reset := ui.NewButton(0, 0, 0, 0, "重置", v.Handle, ui.WithVariant(ui.ButtonOutline))
	reset.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		q.Reset()
		status.SetText("已恢复默认: 当前题 direction, 答案与校验状态清空。")
		status.Redraw(false)
		return 0
	})
	xui.ItemFill(questionnaireNote(v.Handle,
		"快捷键 A/B/C 选中但不前进; ↑/↓ 在答案间移动(单选顺带选中), ←/→ 切题; Enter 在已选/已填写时继续, Ctrl+Enter 随时校验并继续/提交。"))
}

// buildQuestionnaireSkip 可选题跳过 + 条件题(开关启用第二题) + 数字快捷键。
func buildQuestionnaireSkip(parent int) {
	_, v := autoCard(parent, 10)
	xui.ItemFill(ui.NewCardTitle(0, 0, 0, "跳过与条件题", v.Handle))
	xui.ItemFill(ui.NewCardDescription(0, 0, 0,
		"可选题显式 Skip; 开关启用第二题(条件题)", v.Handle))

	swRow := xui.HStack(v.Handle, 24, 8)
	swLabel := ui.NewLabel(0, 0, 0, 20, "启用复查题(条件题)", swRow.Handle)
	xui.ItemWidth(swLabel, 180)
	sw := ui.NewSwitch(0, 0, swRow.Handle)

	items := []ui.QuestionnaireItem{
		{
			Name:        "constraints",
			Title:       "Are there any implementation constraints?",
			Description: "Answer if needed, or intentionally skip this question.",
			Multiple:    true,
			Choices: []ui.QuestionnaireChoice{
				{Value: "no-dependencies", Label: "Do not add dependencies"},
				{Value: "no-migrations", Label: "Do not change the database"},
				{Value: "preserve-api", Label: "Preserve the public API"},
			},
			Input: &ui.QuestionnaireInput{Placeholder: "Describe another constraint…"},
		},
		{
			Name:        "review",
			Title:       "How should the work be reviewed?",
			Description: "Choose the checks the agent should complete before handoff.",
			Required:    true,
			Disabled:    true,
			Choices: []ui.QuestionnaireChoice{
				{Value: "tests", Label: "Run the test suite"},
				{Value: "diff", Label: "Review the final diff"},
				{Value: "both", Label: "Tests and diff review"},
			},
		},
	}

	q := ui.NewQuestionnaire(0, 0, 0, 0, v.Handle, items,
		ui.WithQuestionnaireShortcuts(ui.QuestionnaireShortcutsNumbers),
		ui.WithQuestionnaireActionLabels("", "Skip", "Next", "Submit brief"))
	xui.ItemFill(q)
	sw.OnChange(func(checked bool) {
		q.SetItemDisabled("review", !checked)
	})
	status := questionnaireStatus(v.Handle)
	q.OnSubmit(func(answers map[string][]string) {
		status.SetText("已提交 · " + joinQuestionnaireAnswers(answers))
		status.Redraw(false)
	})
	q.OnStatusChange(func(item string, st ui.QuestionnaireItemStatus) {
		status.SetText("状态 · " + item + " = " + st.String())
		status.Redraw(false)
	})

	xui.ItemFill(questionnaireNote(v.Handle,
		"数字 1/2/3 选择; 跳过会清空选项并把状态记为 skipped(自由输入的文字保留, 再输入即恢复 answered)。"))
}
