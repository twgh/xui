package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/twgh/xcgui/app"
	"github.com/twgh/xcgui/wapi"
	"github.com/twgh/xcgui/widget"
	"github.com/twgh/xcgui/window"
	"github.com/twgh/xcgui/xc"
	"github.com/twgh/xcgui/xcc"
	"github.com/twgh/xui"
	"github.com/twgh/xui/icons"
)

// Questionnaire 几何常量(逻辑像素), 对齐 shadcn base-nova 的 questionnaire.tsx:
//   - 根 / 每步之间 gap-4(16); 选项之间 gap-2(8);
//   - 选项行 rounded-lg border px-3 py-2.5, 最小高 44(min-h-11);
//   - 指示器 size-4(16), 与首行文字的间距 gap-2.5(10), 下移 2 对齐首行;
//   - 标题行 text-base(16)/medium/leading-snug(22); 说明/选项文字 text-sm(14)/20;
//   - 快捷键角标 size-5(20) rounded-md border, mono 10;
//   - 进度 text-xs(12)/medium/muted; 动作行 h-8 按钮、间距 8。
const (
	questionnaireGap          int32 = 16
	questionnaireChoicesGap   int32 = 8
	questionnaireTitleH       int32 = 22
	questionnaireTextH        int32 = 20
	questionnaireProgressH    int32 = 16
	questionnaireChoicePadX   int32 = 12
	questionnaireChoicePadY   int32 = 10
	questionnaireChoiceGap    int32 = 10
	questionnaireChoiceMinH   int32 = 44
	questionnaireIndicator    int32 = 16
	questionnaireIndicatorTop int32 = 2
	questionnaireIndicatorR   int32 = 4
	questionnaireShortcutSize int32 = 20
	questionnaireShortcutFont int32 = 10
	questionnaireActionsH     int32 = 33 // h-8 按钮含 1px 按压预留
	questionnaireErrorMt      int32 = 8  // Error 的 mt-2
	questionnaireStackGap     int32 = 2  // 标题与说明 / 选项标题与描述 gap-0.5
)

// QuestionnaireShortcuts 是答题快捷键模式, 对应 shadcn 的 shortcuts="letters"|"numbers"。
// 快捷键只在当前题目内生效, 选中后不前进; 焦点在自由输入框里时不触发。
type QuestionnaireShortcuts int

const (
	QuestionnaireShortcutsNone    QuestionnaireShortcuts = iota
	QuestionnaireShortcutsLetters                        // A-Z
	QuestionnaireShortcutsNumbers                        // 1-9
)

// QuestionnaireItemStatus 是一道题的作答状态, 对应上游的 onStatusChange。
type QuestionnaireItemStatus int

const (
	QuestionnaireUnanswered QuestionnaireItemStatus = iota
	QuestionnaireAnswered
	QuestionnaireSkipped
)

func (s QuestionnaireItemStatus) String() string {
	switch s {
	case QuestionnaireAnswered:
		return "answered"
	case QuestionnaireSkipped:
		return "skipped"
	default:
		return "unanswered"
	}
}

// QuestionnaireChoice 是固定选项(单选或多选条目)。
type QuestionnaireChoice struct {
	Value       string // 表单值
	Label       string // 选项标题
	Description string // 选项说明(可空)
	Disabled    bool
}

// QuestionnaireInput 是"另一个答案"自由输入框; 传 nil 表示该题没有自由输入。
type QuestionnaireInput struct {
	Placeholder string
}

// QuestionnaireItem 是一道题。Name 在同一个问卷内唯一, 也是 Answers() 的键。
type QuestionnaireItem struct {
	Name        string
	Title       string
	Description string
	Required    bool
	Multiple    bool
	Disabled    bool
	Choices     []QuestionnaireChoice
	Input       *QuestionnaireInput
	// DefaultChoices / DefaultInput 是初始答案(对应 defaultChecked / defaultValue), Reset 会恢复它们。
	DefaultChoices []string
	DefaultInput   string
	// ErrorText 覆盖默认校验文案; 空串时按 required 取内置文案。
	ErrorText string
}

// QuestionnaireOption 问卷配置项。
type QuestionnaireOption func(*Questionnaire)

// WithQuestionnaireShortcuts 设置答题快捷键(默认无)。
func WithQuestionnaireShortcuts(m QuestionnaireShortcuts) QuestionnaireOption {
	return func(q *Questionnaire) { q.shortcuts = m }
}

// WithQuestionnaireDefaultItem 设置初始题(空串 = 第一道可用题)。
func WithQuestionnaireDefaultItem(name string) QuestionnaireOption {
	return func(q *Questionnaire) { q.defaultItem = name }
}

// WithQuestionnaireActionLabels 覆盖动作按钮文案; 空串保留默认英文。
func WithQuestionnaireActionLabels(previous, skip, next, submit string) QuestionnaireOption {
	return func(q *Questionnaire) {
		q.labels = [4]string{previous, skip, next, submit}
	}
}

// WithQuestionnaireBackdrop 指定选项行铺的宿主底色(默认按"放在 Card 上"用 t.Card)。
// 放在 Dialog(Popover) 或页面底色上时跟随宿主, 否则 hover/选中底下的旧像素会透出来。
func WithQuestionnaireBackdrop(fn func(*xui.Theme) xui.Color) QuestionnaireOption {
	return func(q *Questionnaire) {
		if fn != nil {
			q.backdrop = fn
		}
	}
}

// Questionnaire 多步问卷: 一次显示一道题, 管理答案、进度、校验与前进/后退/跳过/提交。
//
// 对齐 shadcn questionnaire.tsx 的语义:
//   - 必答题要有答案; 可选题要有答案或显式跳过(Skip 只对可选题出现);
//   - Next 先校验当前题, 失败显示错误并聚焦答案; Submit 校验所有可用题,
//     定位到第一道未通过并聚焦; 全部通过才回调 OnSubmit;
//   - 禁用的题不参与进度、导航、校验与提交(条件题用 SetItemDisabled 动态切换);
//   - shortcuts 为每个选项分配字母/数字快捷键(禁用的选项跳过), 按键选中但不前进;
//   - ↑/↓ 在答案间移动(单选会顺带选中), ←/→ 切题; Ctrl+Enter 在任何位置校验并继续/提交;
//     Enter 在已选中的选项或已填写的自由输入上继续;
//   - 自由输入框获得焦点时不触发快捷键与方向键切题(键盘交给文本框)。
//
// xcgui 限制: 布局容器里的元素不在 Tab 焦点链上(见 DESIGN 第八节 157), 键盘操作需先
// 鼠标点击问卷内的控件获得焦点; Tab 无法进入选项/动作按钮。
//
// 用法:
//
//	q := ui.NewQuestionnaire(0, 0, 0, 0, cardContent.Handle, []ui.QuestionnaireItem{
//	    {Name: "scope", Title: "改哪些文件?", Required: true,
//	        Choices: []ui.QuestionnaireChoice{{Value: "ui", Label: "只改组件"}}},
//	    {Name: "note", Title: "补充说明", Input: &ui.QuestionnaireInput{Placeholder: "可选"}},
//	}, ui.WithQuestionnaireShortcuts(ui.QuestionnaireShortcutsLetters))
//	xui.ItemFill(q)
//	q.OnSubmit(func(answers map[string][]string) { log.Println(answers["scope"]) })
type Questionnaire struct {
	*widget.LayoutEle
	items         []*questionnaireItem
	byName        map[string]*questionnaireItem
	currentName   string
	defaultItem   string
	shortcuts     QuestionnaireShortcuts
	progress      *Label
	actions       *widget.LayoutEle
	prev          *Button
	skip          *Button
	next          *Button
	submit        *Button
	labels        [4]string
	backdrop      func(*xui.Theme) xui.Color
	format        func(current, total int) string
	onChange      func(item string)
	onSubmit      func(answers map[string][]string)
	onStatus      func(item string, status QuestionnaireItemStatus)
	hWindow       int
	keyID         int
	resetting     bool
	reflowPending bool
}

type questionnaireItem struct {
	q          *Questionnaire
	def        QuestionnaireItem
	root       *widget.LayoutEle
	title      *Label
	desc       *Label
	choices    *widget.LayoutEle
	rows       []*questionnaireChoiceRow
	input      *Input
	errorLabel *Label
	skipped    bool
	attempted  bool
	invalid    bool
	errorShown bool
	lastStatus QuestionnaireItemStatus
}

type questionnaireChoiceRow struct {
	*widget.LayoutEle
	q       *Questionnaire
	item    *questionnaireItem
	def     QuestionnaireChoice
	column  *widget.LayoutEle
	title   *Label
	desc    *Label
	checked bool
	short   string
	hover   bool
	focused bool
}

// NewQuestionnaire 创建问卷。宽铺满、高自撑; items 为题目定义。
func NewQuestionnaire(x, y, cx, cy int32, hParent int, items []QuestionnaireItem, opts ...QuestionnaireOption) *Questionnaire {
	root := widget.NewLayoutEle(x, y, cx, cy, hParent)
	if root == nil {
		return nil
	}
	q := &Questionnaire{
		LayoutEle: root,
		byName:    map[string]*questionnaireItem{},
		backdrop:  func(t *xui.Theme) xui.Color { return t.Card },
		labels:    [4]string{"Previous", "Skip", "Next", "Submit"},
	}
	for _, o := range opts {
		o(q)
	}
	root.EnableHorizon(false)
	root.EnableLayout(true)
	root.EnableBkTransparent(true)
	root.EnableAutoWrap(false)
	root.SetSpace(questionnaireGap)
	root.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	root.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.AttachWindow(root.Handle)

	// 进度: Question 1 of 3, text-xs / medium / muted。
	q.progress = NewLabel(0, 0, 0, questionnaireProgressH, "", root.Handle).SetMuted().SetMedium()
	q.progress.SetFontSize(12)
	xui.ItemFill(q.progress)
	q.progress.Show(false)

	for _, def := range items {
		it := q.newItem(def)
		if it == nil {
			continue
		}
		q.items = append(q.items, it)
		q.byName[def.Name] = it
	}

	// 空串回落默认文案(上游允许只改其中一两个按钮), 否则空文案按钮自动宽为 0。
	defaults := [4]string{"Previous", "Skip", "Next", "Submit"}
	for i := range q.labels {
		if q.labels[i] == "" {
			q.labels[i] = defaults[i]
		}
	}
	q.actions = xui.HStack(root.Handle, questionnaireActionsH, questionnaireChoicesGap)
	q.prev = NewButton(0, 0, 0, 0, q.labels[0], q.actions.Handle, WithVariant(ButtonOutline))
	spacer := widget.NewLayoutEle(0, 0, 0, 0, q.actions.Handle)
	spacer.EnableBkTransparent(true)
	spacer.EnableMouseThrough(true)
	spacer.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
	spacer.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 1)
	q.skip = NewButton(0, 0, 0, 0, q.labels[1], q.actions.Handle, WithVariant(ButtonOutline))
	q.next = NewButton(0, 0, 0, 0, q.labels[2], q.actions.Handle)
	q.submit = NewButton(0, 0, 0, 0, q.labels[3], q.actions.Handle)
	q.bindActions()
	q.resetState(false)
	q.installKeys()
	q.syncFlow(false)
	return q
}

func newQuestionnaireStack(parent int, gap int32) *widget.LayoutEle {
	l := widget.NewLayoutEle(0, 0, 0, 0, parent)
	l.EnableHorizon(false)
	l.EnableLayout(true)
	l.EnableBkTransparent(true)
	l.EnableAutoWrap(false)
	l.SetSpace(gap)
	l.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	l.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	return l
}

func (q *Questionnaire) newItem(def QuestionnaireItem) *questionnaireItem {
	it := &questionnaireItem{q: q, def: def, lastStatus: QuestionnaireUnanswered}
	it.root = newQuestionnaireStack(q.Handle, questionnaireGap)
	it.root.EnableFocus(true)
	it.root.EnableSwitchFocus(false) // 只做程序化焦点落点, 不进 Tab 链
	it.root.EnableDrawFocus(false)

	head := newQuestionnaireStack(it.root.Handle, questionnaireStackGap)
	it.title = NewLabel(0, 0, 0, questionnaireTitleH, def.Title, head.Handle).SetMedium().SetAutoWrap()
	it.title.SetFontSize(16)
	xui.ItemFill(it.title)
	if def.Description != "" {
		it.desc = NewLabel(0, 0, 0, questionnaireTextH, def.Description, head.Handle).SetMuted().SetAutoWrap()
		xui.ItemFill(it.desc)
	}

	it.choices = newQuestionnaireStack(it.root.Handle, questionnaireChoicesGap)
	for _, c := range def.Choices {
		if row := it.newChoiceRow(c); row != nil {
			it.rows = append(it.rows, row)
		}
	}
	if def.Input != nil {
		it.input = NewInput(0, 0, 0, 0, it.choices.Handle, WithPlaceholder(def.Input.Placeholder))
		xui.ItemFill(it.input)
		it.input.OnChange(func(string) {
			if it.skipped && strings.TrimSpace(it.input.Value()) != "" {
				it.skipped = false
			}
			q.answerChanged(it)
		})
	}

	it.errorLabel = NewLabel(0, 0, 0, questionnaireTextH, "", it.root.Handle)
	it.errorLabel.SetFontSize(14).SetAutoWrap()
	it.errorLabel.SetColorFunc(func(t *xui.Theme) xui.Color { return t.Destructive })
	xui.ItemFill(it.errorLabel)
	xui.ItemMarginTop(it.errorLabel, questionnaireErrorMt)
	it.errorLabel.Show(false)

	it.root.Show(false)
	return it
}

func (it *questionnaireItem) newChoiceRow(def QuestionnaireChoice) *questionnaireChoiceRow {
	ele := widget.NewLayoutEle(0, 0, 0, 0, it.choices.Handle)
	if ele == nil {
		return nil
	}
	row := &questionnaireChoiceRow{LayoutEle: ele, q: it.q, item: it, def: def}
	ele.EnableHorizon(true)
	ele.EnableLayout(true)
	ele.EnableBkTransparent(true)
	ele.EnableMouseThrough(false)
	ele.EnableAutoWrap(false)
	ele.SetAlignV(xcc.Layout_Align_Top)
	ele.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	ele.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	xui.ItemMinSize(ele, 0, questionnaireChoiceMinH)
	xc.XEle_SetPadding(ele.Handle, questionnaireChoicePadX, questionnaireChoicePadY,
		questionnaireChoicePadX, questionnaireChoicePadY)
	ele.EnableFocus(true)
	ele.EnableSwitchFocus(false)
	ele.EnableDrawFocus(false)
	ele.AddEvent_Paint(row.onPaint)
	ele.AddEvent_MouseStay(func(hEle int, pbHandled *bool) int {
		if !row.disabled() && !row.hover {
			row.hover = true
			row.Redraw(true)
		}
		return 0
	})
	ele.AddEvent_MouseLeave(func(hEle, hEleStay int, pbHandled *bool) int {
		if row.hover {
			row.hover = false
			row.Redraw(true)
		}
		return 0
	})
	ele.AddEvent_LButtonDown(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		xui.NotePointerPress() // 鼠标焦点不画 focus ring
		return 0
	})
	ele.AddEvent_LButtonUp(func(hEle int, nFlags int, pPt *xc.POINT, pbHandled *bool) int {
		if row.disabled() {
			return 0
		}
		xui.FocusElement(row.Handle)
		row.q.selectChoice(row)
		return 0
	})
	ele.AddEvent_SetFocus(func(hEle int, pbHandled *bool) int {
		row.focused = true
		row.Redraw(false)
		return 0
	})
	ele.AddEvent_KillFocus(func(hEle int, pbHandled *bool) int {
		row.focused = false
		row.Redraw(false)
		return 0
	})
	xui.AttachWindow(ele.Handle)

	// 内容列: 在横向行里按权重吃剩余宽度; xui.VStack 会在宽度变化时把 Fill 子项锁成 Fixed,
	// 否则嵌套布局里的 Fill 子项宽恒为 0(见 DESIGN 第八节 101/129)。
	row.column = xui.VStack(ele.Handle, 1, questionnaireStackGap)
	row.column.EnableMouseThrough(true)
	xui.ItemMarginLeft(row.column, questionnaireIndicator+questionnaireChoiceGap)
	row.setShortcutMargin("")

	row.title = NewLabel(0, 0, 0, questionnaireTextH, def.Label, row.column.Handle).SetMedium().SetAutoWrap()
	row.title.SetColorFunc(func(t *xui.Theme) xui.Color {
		if row.disabled() {
			return t.Foreground.WithAlpha(128)
		}
		return t.Foreground
	})
	row.title.EnableMouseThrough(true)
	xui.ItemFill(row.title)
	if def.Description != "" {
		row.desc = NewLabel(0, 0, 0, questionnaireTextH, def.Description, row.column.Handle).SetMuted().SetAutoWrap()
		row.desc.SetColorFunc(func(t *xui.Theme) xui.Color {
			if row.disabled() {
				return t.MutedForeground.WithAlpha(128)
			}
			return t.MutedForeground
		})
		row.desc.EnableMouseThrough(true)
		xui.ItemFill(row.desc)
	}
	if def.Disabled {
		ele.Enable(false)
	}
	return row
}

// setShortcutMargin 让内容列为快捷键角标预留位置; 没有快捷键时不占那 30px。
func (r *questionnaireChoiceRow) setShortcutMargin(short string) {
	r.short = short
	right := int32(0)
	if short != "" {
		right = questionnaireShortcutSize + questionnaireChoiceGap
	}
	xui.ItemMarginRight(r.column, right)
}

func (r *questionnaireChoiceRow) disabled() bool {
	return r.def.Disabled || r.item.def.Disabled
}

func (r *questionnaireChoiceRow) setChecked(v bool) {
	if r.checked == v {
		return
	}
	r.checked = v
	r.Redraw(false)
}

func (r *questionnaireChoiceRow) onPaint(hEle int, hDraw int, pbHandled *bool) int {
	p := xui.BeginPaint(hEle, hDraw)
	if p == nil {
		return 0
	}
	*pbHandled = true
	t := xui.Current()
	disabled := r.disabled()
	checked := r.checked && !r.item.skipped
	invalid := r.item.invalid
	focused := r.focused && !disabled && xui.FocusedByKeyboard()
	hv := xui.Hover(hEle, r.hover && !disabled)

	radius := t.RadiusLg()
	host := r.q.backdropColor(t)
	border := t.Input
	switch {
	case disabled:
		border = t.Input.ScaleAlpha(0.5)
	case invalid:
		border = t.Destructive
	case checked:
		border = t.Primary.ScaleAlpha(0.4)
	case focused:
		border = t.Ring
	}

	// 底: 先铺宿主色(透明元素重绘不擦底), 再叠 dark:bg-input/20 / checked:bg-muted /
	// hover:bg-muted/50(见第八节 28/61)。
	p.FillRound(&p.Rc, host, radius)
	if checked {
		p.FillRound(&p.Rc, t.Muted, radius)
	} else if !disabled {
		if t.IsDark() {
			p.FillRound(&p.Rc, t.Input.ScaleAlpha(0.20), radius)
		}
		if hv > 0 {
			p.FillRound(&p.Rc, t.Muted.ScaleAlpha(0.5*hv), radius)
		}
	}

	if focused {
		// focus-visible: border-ring + ring-3 ring-ring/50。光环画在元素内(同 Button),
		// 内芯先铺宿主底色再叠内容底, 避免半透明色互相叠深。
		p.FillRound(&p.Rc, t.Ring.WithAlpha(128), radius)
		const inset = int32(3)
		inner := xc.RECT{Left: p.Rc.Left + inset, Top: p.Rc.Top + inset, Right: p.Rc.Right - inset, Bottom: p.Rc.Bottom - inset}
		if inner.Right-inner.Left > 4 && inner.Bottom-inner.Top > 4 {
			p.FillRound(&inner, host, radius-inset)
			if checked {
				p.FillRound(&inner, t.Muted, radius-inset)
			} else if t.IsDark() {
				p.FillRound(&inner, t.Input.ScaleAlpha(0.20), radius-inset)
			}
			p.StrokeRound(&inner, border, radius-inset, 1)
		}
	} else {
		p.StrokeRound(&p.Rc, border, radius, 1)
	}

	r.paintIndicator(p, t, checked, disabled)
	r.paintShortcut(p, t, disabled)
	return 0
}

func (r *questionnaireChoiceRow) paintIndicator(p *xui.Painter, t *xui.Theme, checked, disabled bool) {
	top := p.Rc.Top + questionnaireChoicePadY + questionnaireIndicatorTop
	rc := xc.RECT{
		Left:   p.Rc.Left + questionnaireChoicePadX,
		Top:    top,
		Right:  p.Rc.Left + questionnaireChoicePadX + questionnaireIndicator,
		Bottom: top + questionnaireIndicator,
	}
	alpha := func(c xui.Color) xui.Color {
		if disabled {
			return c.WithAlpha(byte(float64(alphaOf(c)) * 0.5))
		}
		return c
	}
	radius := questionnaireIndicatorR
	if !r.item.def.Multiple {
		radius = questionnaireIndicator / 2
	}
	if checked {
		p.FillRound(&rc, alpha(t.Primary), radius)
		p.StrokeRound(&rc, alpha(t.Primary), radius, 1)
		if r.item.def.Multiple {
			check := rc
			check.Left++
			check.Top++
			check.Right--
			check.Bottom--
			p.Icon(icons.IconCheck, &check, alpha(t.PrimaryForeground), 2)
		} else {
			dot := questionnaireIndicator * 50 / 100
			center := xc.RECT{
				Left:   rc.Left + (questionnaireIndicator-dot)/2,
				Top:    rc.Top + (questionnaireIndicator-dot)/2,
				Right:  rc.Left + (questionnaireIndicator+dot)/2,
				Bottom: rc.Top + (questionnaireIndicator+dot)/2,
			}
			p.FillRound(&center, alpha(t.PrimaryForeground), dot/2)
		}
		return
	}
	if t.IsDark() {
		p.FillRound(&rc, alpha(t.Input.ScaleAlpha(0.30)), radius)
	}
	p.StrokeRound(&rc, alpha(t.Input), radius, 1)
}

func (r *questionnaireChoiceRow) paintShortcut(p *xui.Painter, t *xui.Theme, disabled bool) {
	if r.short == "" {
		return
	}
	top := p.Rc.Top + questionnaireChoicePadY + questionnaireIndicatorTop
	rc := xc.RECT{
		Left:   p.Rc.Right - questionnaireChoicePadX - questionnaireShortcutSize,
		Top:    top,
		Right:  p.Rc.Right - questionnaireChoicePadX,
		Bottom: top + questionnaireShortcutSize,
	}
	alpha := func(c xui.Color) xui.Color {
		if disabled {
			return c.WithAlpha(byte(float64(alphaOf(c)) * 0.5))
		}
		return c
	}
	p.FillRound(&rc, alpha(t.Background), t.RadiusMd())
	p.StrokeRound(&rc, alpha(t.Input), t.RadiusMd(), 1)
	if f := xui.FontHandle(xui.FontSpec{Family: t.FontMono.Family, Size: questionnaireShortcutFont}); f != 0 {
		p.Draw.SetFont(f)
	}
	p.Text(r.short, &rc, alpha(t.MutedForeground),
		xcc.TextAlignFlag_Center|xcc.TextAlignFlag_Vcenter|xcc.TextFormatFlag_NoWrap)
}

func alphaOf(c xui.Color) byte {
	_, _, _, a := c.RGBA()
	return a
}

func (q *Questionnaire) backdropColor(t *xui.Theme) xui.Color {
	if q.backdrop != nil {
		return q.backdrop(t)
	}
	return t.Card
}

// bindActions 挂动作按钮。HStack 的子项顺序是 prev / spacer / skip / next / submit,
// prev 贴左, spacer 撑开剩余宽度后其余按钮靠右。
func (q *Questionnaire) bindActions() {
	q.prev.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		q.goPrevious()
		return 0
	})
	q.skip.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		q.skipCurrent()
		return 0
	})
	q.next.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		q.goNext()
		return 0
	})
	q.submit.AddEvent_BnClick(func(hEle int, pbHandled *bool) int {
		q.submitAll()
		return 0
	})
}

// SetShortcuts 切换快捷键模式; 已显示的角标与保留宽度同步更新。
func (q *Questionnaire) SetShortcuts(m QuestionnaireShortcuts) *Questionnaire {
	if q == nil || q.shortcuts == m {
		return q
	}
	q.shortcuts = m
	q.syncShortcuts()
	return q
}

// SetBackdrop 指定选项行的宿主底色(见 WithQuestionnaireBackdrop)。
func (q *Questionnaire) SetBackdrop(fn func(*xui.Theme) xui.Color) *Questionnaire {
	if q == nil || fn == nil {
		return q
	}
	q.backdrop = fn
	for _, it := range q.items {
		for _, row := range it.rows {
			row.Redraw(false)
		}
	}
	return q
}

// SetProgressFormat 自定义进度文案(默认 "Question {current} of {total}")。
func (q *Questionnaire) SetProgressFormat(fn func(current, total int) string) *Questionnaire {
	if q == nil {
		return q
	}
	q.format = fn
	q.syncProgress()
	return q
}

// OnItemChange 注册当前题变化回调(翻页/跳转时调用)。
func (q *Questionnaire) OnItemChange(fn func(item string)) *Questionnaire {
	if q != nil {
		q.onChange = fn
	}
	return q
}

// OnSubmit 注册提交回调; answers 只包含有答案且未被跳过的题(多选题为多个值)。
func (q *Questionnaire) OnSubmit(fn func(answers map[string][]string)) *Questionnaire {
	if q != nil {
		q.onSubmit = fn
	}
	return q
}

// OnStatusChange 注册作答状态变化回调。
func (q *Questionnaire) OnStatusChange(fn func(item string, status QuestionnaireItemStatus)) *Questionnaire {
	if q != nil {
		q.onStatus = fn
	}
	return q
}

// Item 返回当前题名。
func (q *Questionnaire) Item() string {
	if q == nil {
		return ""
	}
	return q.currentName
}

// SetItem 跳到指定题(外部受控导航, 不校验当前题); 名称不存在或被禁用时忽略。
func (q *Questionnaire) SetItem(name string) *Questionnaire {
	if q == nil {
		return q
	}
	it := q.byName[name]
	if it == nil || it.def.Disabled {
		return q
	}
	if q.currentName == name {
		return q
	}
	q.setCurrent(it, false)
	return q
}

// SetItemDisabled 动态启用/禁用一道题(条件题); 禁用后不参与进度/导航/校验/提交。
func (q *Questionnaire) SetItemDisabled(name string, disabled bool) *Questionnaire {
	if q == nil {
		return q
	}
	it := q.byName[name]
	if it == nil || it.def.Disabled == disabled {
		return q
	}
	it.def.Disabled = disabled
	if disabled {
		it.root.Show(false)
	}
	// 当前题被禁用时回落到第一道可用题(同上游: 逻辑列表里找不到当前项就取第一项)。
	if disabled && q.currentName == name {
		q.currentName = ""
	}
	q.syncFlow(true)
	return q
}

// Reset 恢复初始题、默认答案、跳过与校验状态(同原生 form.reset)。
func (q *Questionnaire) Reset() *Questionnaire {
	if q == nil {
		return q
	}
	q.resetState(true)
	q.syncFlow(true)
	return q
}

// Answers 返回当前答案快照: 未作答/被跳过的题不出现; 多选按选项定义顺序, 自由输入追加在末尾。
func (q *Questionnaire) Answers() map[string][]string {
	out := make(map[string][]string, len(q.items))
	if q == nil {
		return out
	}
	for _, it := range q.items {
		if it.def.Disabled || it.skipped {
			continue
		}
		vals := make([]string, 0, len(it.rows)+1)
		for _, row := range it.rows {
			if row.checked && !row.disabled() {
				vals = append(vals, row.def.Value)
			}
		}
		if it.input != nil {
			if v := strings.TrimSpace(it.input.Value()); v != "" {
				vals = append(vals, v)
			}
		}
		if len(vals) > 0 {
			out[it.def.Name] = vals
		}
	}
	return out
}

// Status 返回指定题的作答状态。
func (q *Questionnaire) Status(name string) QuestionnaireItemStatus {
	if q == nil {
		return QuestionnaireUnanswered
	}
	if it := q.byName[name]; it != nil {
		return it.status()
	}
	return QuestionnaireUnanswered
}

func (q *Questionnaire) resetState(notify bool) {
	q.resetting = true
	for _, it := range q.items {
		it.skipped = false
		it.attempted = false
		it.invalid = false
		if it.errorShown {
			it.errorShown = false
			it.errorLabel.Show(false)
		}
		defaults := map[string]bool{}
		for _, v := range it.def.DefaultChoices {
			defaults[v] = true
		}
		done := false
		for _, row := range it.rows {
			on := defaults[row.def.Value] && !row.def.Disabled
			if on && !it.def.Multiple {
				on = !done
				done = true
			}
			row.checked = on
			row.Redraw(false)
		}
		if it.input != nil {
			it.input.SetValue(it.def.DefaultInput)
			it.input.SetInvalid(false)
		}
		if notify {
			it.notifyStatus()
		} else {
			it.lastStatus = it.status()
		}
	}
	q.resetting = false
	start := q.defaultItem
	if it := q.byName[start]; it == nil || it.def.Disabled {
		start = ""
	}
	q.currentName = start
}

func (q *Questionnaire) enabledItems() []*questionnaireItem {
	out := make([]*questionnaireItem, 0, len(q.items))
	for _, it := range q.items {
		if !it.def.Disabled {
			out = append(out, it)
		}
	}
	return out
}

func (q *Questionnaire) activeItem() *questionnaireItem {
	if q == nil {
		return nil
	}
	if it := q.byName[q.currentName]; it != nil && !it.def.Disabled {
		return it
	}
	return nil
}

func (q *Questionnaire) indexOf(name string) int {
	for i, it := range q.enabledItems() {
		if it.def.Name == name {
			return i
		}
	}
	return -1
}

// syncFlow 按当前题刷新显隐、进度与动作按钮。focus=true 时把焦点落到当前题容器
// (同上游导航后 focus 新 fieldset; 面板内的键盘操作从这里开始)。
func (q *Questionnaire) syncFlow(focus bool) {
	enabled := q.enabledItems()
	idx := q.indexOf(q.currentName)
	if idx < 0 {
		if len(enabled) == 0 {
			q.currentName = ""
		} else {
			q.currentName = enabled[0].def.Name
			idx = 0
		}
	}
	for _, it := range q.items {
		show := !it.def.Disabled && it.def.Name == q.currentName
		it.root.Show(show)
		if show {
			it.restoreSize()
			it.root.AdjustLayout()
			// 重新显示的原生控件(Edit)与自绘按钮不一定会跟着父级重排自动重绘, 显式失效。
			it.root.Redraw(false)
			if it.input != nil {
				it.input.Redraw(false)
			}
			for _, row := range it.rows {
				row.Redraw(false)
			}
		}
	}
	q.syncProgress()
	q.syncActions()
	q.syncShortcuts()
	q.reflow()
	if focus && idx >= 0 {
		if it := q.activeItem(); it != nil {
			xui.FocusElement(it.root.Handle)
		}
	}
}

func (q *Questionnaire) syncProgress() {
	enabled := q.enabledItems()
	idx := q.indexOf(q.currentName)
	show := idx >= 0 && len(enabled) > 1
	q.progress.Show(show)
	if !show {
		return
	}
	q.progress.SetHeight(questionnaireProgressH)
	text := ""
	if q.format != nil {
		text = q.format(idx+1, len(enabled))
	}
	if text == "" {
		text = fmt.Sprintf("Question %d of %d", idx+1, len(enabled))
	}
	q.progress.SetText(text)
	q.progress.Redraw(false)
}

func (q *Questionnaire) syncActions() {
	enabled := q.enabledItems()
	idx := q.indexOf(q.currentName)
	last := idx == len(enabled)-1
	it := q.activeItem()
	q.showAction(q.prev, idx > 0)
	q.showAction(q.skip, it != nil && !it.def.Required)
	q.showAction(q.next, idx >= 0 && !last)
	q.showAction(q.submit, idx >= 0 && last)
	q.actions.AdjustLayout()
	for _, b := range []*Button{q.prev, q.skip, q.next, q.submit} {
		if b.IsShow() {
			b.Redraw(false)
		}
	}
}

// showAction 在重新显示动作按钮时按文字恢复宽: 隐藏期间元素被排成 0 宽, 直接 Show(true)
// 会剩一个 0 宽按钮(实测); FitWidth 只对 cx=0 创建的自动宽按钮生效。
func (q *Questionnaire) showAction(b *Button, show bool) {
	b.Show(show)
	if show {
		// 隐藏会把布局项类型重置为 Layout_Size_Disable, 重新 Show 不恢复:
		// 元素矩形还是旧值, 但父级排版不再为它让位。按文字重设宽高并改回 Fixed。
		b.SetHeight(questionnaireActionsH)
		b.FitWidth()
		b.LayoutItem_SetWidth(xcc.Layout_Size_Fixed, b.GetWidth())
		b.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, questionnaireActionsH)
	}
}

func (q *Questionnaire) syncShortcuts() {
	for _, it := range q.items {
		shortcuts := questionnaireShortcutKeys(q.shortcuts, it.rows)
		for i, row := range it.rows {
			short := ""
			if i < len(shortcuts) {
				short = shortcuts[i]
			}
			if row.short != short {
				row.setShortcutMargin(short)
				row.Redraw(false)
			}
		}
		it.choices.AdjustLayout()
	}
	q.reflow()
}

// questionnaireShortcutKeys 按选项顺序分配快捷键; 禁用的选项跳过。
func questionnaireShortcutKeys(mode QuestionnaireShortcuts, rows []*questionnaireChoiceRow) []string {
	disabled := make([]bool, len(rows))
	for i, row := range rows {
		disabled[i] = row.def.Disabled
	}
	return questionnaireShortcutAssign(mode, disabled)
}

// questionnaireShortcutAssign 是 questionnaireShortcutKeys 的纯计算版本(便于测试)。
func questionnaireShortcutAssign(mode QuestionnaireShortcuts, disabled []bool) []string {
	out := make([]string, len(disabled))
	if mode == QuestionnaireShortcutsNone {
		return out
	}
	key := 0
	for i, off := range disabled {
		if off {
			continue
		}
		switch mode {
		case QuestionnaireShortcutsLetters:
			if key < 26 {
				out[i] = string(rune('A' + key))
			}
		case QuestionnaireShortcutsNumbers:
			if key < 9 {
				out[i] = string(rune('1' + key))
			}
		}
		key++
	}
	return out
}

func (q *Questionnaire) goPrevious() {
	enabled := q.enabledItems()
	idx := q.indexOf(q.currentName)
	if idx <= 0 {
		return
	}
	q.setCurrent(enabled[idx-1], true)
}

func (q *Questionnaire) goNext() {
	enabled := q.enabledItems()
	idx := q.indexOf(q.currentName)
	it := q.activeItem()
	if it == nil || idx < 0 || idx >= len(enabled)-1 {
		return
	}
	if !q.validate(it) {
		it.focusFirstAnswer()
		return
	}
	q.setCurrent(enabled[idx+1], true)
}

func (q *Questionnaire) skipCurrent() {
	it := q.activeItem()
	if it == nil || it.def.Required {
		return
	}
	it.skip()
	enabled := q.enabledItems()
	idx := q.indexOf(q.currentName)
	if idx == len(enabled)-1 {
		q.submitAll()
		return
	}
	if idx >= 0 && idx < len(enabled)-1 {
		q.setCurrent(enabled[idx+1], true)
	}
}

func (q *Questionnaire) submitAll() {
	for _, it := range q.enabledItems() {
		if !q.validate(it) {
			if q.currentName != it.def.Name {
				q.setCurrent(it, false)
			}
			it.focusFirstAnswer()
			return
		}
	}
	if q.onSubmit != nil {
		q.onSubmit(q.Answers())
	}
}

func (q *Questionnaire) setCurrent(it *questionnaireItem, focus bool) {
	if it == nil || it.def.Disabled {
		return
	}
	changed := q.currentName != it.def.Name
	q.currentName = it.def.Name
	q.syncFlow(focus)
	if changed && q.onChange != nil {
		q.onChange(it.def.Name)
	}
}

func (q *Questionnaire) validate(it *questionnaireItem) bool {
	it.attempted = true
	it.syncInvalid()
	return !it.invalid
}

func (q *Questionnaire) selectChoice(row *questionnaireChoiceRow) {
	it := row.item
	if it == nil || row.disabled() || it.def.Disabled {
		return
	}
	it.skipped = false
	if it.def.Multiple {
		row.setChecked(!row.checked)
	} else {
		for _, r := range it.rows {
			r.setChecked(r == row)
		}
	}
	q.answerChanged(it)
}

func (q *Questionnaire) answerChanged(it *questionnaireItem) {
	if q.resetting {
		return
	}
	if it.attempted {
		it.syncInvalid() // 错误提示显隐会改高度, 内部自己重排
	}
	it.notifyStatus()
}

func (it *questionnaireItem) answered() bool {
	for _, row := range it.rows {
		if row.checked && !row.disabled() {
			return true
		}
	}
	if it.input != nil {
		return strings.TrimSpace(it.input.Value()) != ""
	}
	return false
}

func (it *questionnaireItem) status() QuestionnaireItemStatus {
	if it.skipped && !it.def.Required {
		return QuestionnaireSkipped
	}
	if it.answered() {
		return QuestionnaireAnswered
	}
	return QuestionnaireUnanswered
}

func (it *questionnaireItem) valid() bool {
	if it.def.Disabled {
		return true
	}
	if it.skipped && !it.def.Required {
		return true
	}
	return it.answered()
}

func (it *questionnaireItem) errorText() string {
	if it.def.ErrorText != "" {
		return it.def.ErrorText
	}
	if it.def.Required {
		return "Choose an answer to continue."
	}
	return "Choose an answer or skip this question."
}

func (it *questionnaireItem) syncInvalid() {
	invalid := it.attempted && !it.valid()
	it.invalid = invalid
	for _, row := range it.rows {
		row.Redraw(false)
	}
	if it.input != nil {
		it.input.SetInvalid(invalid)
	}
	if it.errorShown != invalid {
		it.errorShown = invalid
		it.errorLabel.Show(invalid)
		if invalid {
			// 错误标签创建时是隐藏的, 布局项停在 Disable; 显示时和切题一样要重写声明,
			// 否则它不占高度、动作行会被顶回旧位置(实测 Next 整块消失, 见第八节 197)。
			it.errorLabel.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
			it.errorLabel.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
			it.errorLabel.SetText(it.errorText())
		}
		it.root.AdjustLayout()
		it.q.reflow()
	}
	if invalid {
		it.errorLabel.Redraw(false)
	}
}

// reflow 逐级通知祖先重排: q 自己的 Auto 高 / 卡片 Auto 高 / 页面滚动内容都要跟着变,
// 只对 q 调 AdjustLayout 时父级仍按旧高裁剪, 表现为重新显示的自由输入框与动作行整块不见。
func (q *Questionnaire) reflow() {
	reflowAfterTextHeight(q.Handle)
	q.redrawHost()
	if q.reflowPending {
		return
	}
	// 同步重排后绘制位置已经变了, 但控件命中区还停在旧位置(点新位置没反应,
	// 点旧位置才触发, 见第八节 118); 下一轮消息里再排一次, 让框架重建命中映射
	// (与 demo 切页后的延迟重排同一口径)。
	q.reflowPending = true
	app.AfterFunc(2*16*time.Millisecond, func() {
		q.reflowPending = false
		if !xc.XC_IsHXCGUI(q.Handle, xcc.XC_ELE) {
			return
		}
		reflowAfterTextHeight(q.Handle)
		q.redrawHost()
	})
}

func (q *Questionnaire) redrawHost() {
	// 重排只改坐标, 不擦旧像素: 动作行上移/下移会留下按钮重影(实测 Next 残影),
	// 布局趟结束后整窗重绘一次(同 Field 错误区的重排口径, 见第八节 102/90)。
	if hwnd := xc.XWidget_GetHWND(q.Handle); hwnd != 0 {
		if hw := xc.XC_hWindowFromHWnd(hwnd); hw != 0 {
			xc.XWnd_Redraw(hw, false)
		}
	}
}

func (it *questionnaireItem) skip() {
	it.skipped = true
	it.attempted = false
	it.invalid = false
	for _, row := range it.rows {
		row.setChecked(false)
	}
	if it.errorShown {
		it.errorShown = false
		it.errorLabel.Show(false)
	}
	if it.input != nil {
		it.input.SetInvalid(false)
	}
	it.notifyStatus()
	it.root.AdjustLayout()
	it.q.reflow()
}

func (it *questionnaireItem) notifyStatus() {
	st := it.status()
	if st == it.lastStatus {
		return
	}
	it.lastStatus = st
	if it.q.onStatus != nil {
		it.q.onStatus(it.def.Name, st)
	}
}

// restoreSize 重写被 Show(false) 清掉的布局声明。隐藏会把布局项类型改成
// Layout_Size_Disable(实测 GetWidth/GetHeight 返回 (5,0)), Show(true) 只改显隐、
// 不会恢复声明: 输入框的 Fixed 高、错误标签的 Auto 高与行的 Fill/Auto 要按原值写回。
func (it *questionnaireItem) restoreSize() {
	it.root.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	it.root.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	it.choices.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	it.choices.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	if it.title != nil {
		it.title.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	}
	if it.desc != nil {
		it.desc.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
	}
	if it.input != nil {
		it.input.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		it.input.LayoutItem_SetHeight(xcc.Layout_Size_Fixed, 32)
		it.input.SetHeight(32)
	}
	if it.errorLabel != nil {
		it.errorLabel.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		it.errorLabel.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	}
	for _, row := range it.rows {
		row.LayoutItem_SetWidth(xcc.Layout_Size_Fill, 0)
		row.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
		row.column.LayoutItem_SetWidth(xcc.Layout_Size_Weight, 1)
		row.column.LayoutItem_SetHeight(xcc.Layout_Size_Auto, 0)
	}
}

func (it *questionnaireItem) focusFirstAnswer() {
	for _, row := range it.rows {
		if !row.disabled() {
			xui.NoteKeyboard()
			xui.FocusElement(row.Handle)
			return
		}
	}
	if it.input != nil {
		xui.NoteKeyboard()
		xui.FocusElement(it.input.Handle)
		return
	}
	xui.NoteKeyboard()
	xui.FocusElement(it.root.Handle)
}

// installKeys 挂窗口级键盘处理: 焦点在问卷内才响应。挂窗口级的原因见 Tabs 的同名注释
// (xcgui 会先做自己的焦点切换, 元素级 KeyDown 收不到)。
func (q *Questionnaire) installKeys() {
	q.hWindow = xui.HostWindowOf(q.Handle)
	if q.hWindow == 0 {
		return
	}
	w := window.NewByHandle(q.hWindow)
	if w == nil {
		return
	}
	q.keyID = w.AddEvent_KeyDown(func(hWin int, wParam, lParam uintptr, pbHandled *bool) int {
		focus := xc.XWnd_GetFocusEle(hWin)
		if !q.contains(focus) {
			return 0
		}
		return q.onKeyDown(focus, wParam, pbHandled)
	})
	q.AddEvent_Destroy(func(hEle int, pbHandled *bool) int {
		if q.keyID > 0 {
			if xc.XC_IsHXCGUI(q.hWindow, xcc.XC_WINDOW) {
				w.RemoveEvent(xcc.WM_KEYDOWN, q.keyID)
			}
			q.keyID = 0
		}
		return 0
	})
}

func (q *Questionnaire) contains(hEle int) bool {
	for h := hEle; h != 0; h = xc.XWidget_GetParentEle(h) {
		if h == q.Handle {
			return true
		}
	}
	return false
}

const (
	vkReturn  = 0x0D
	vkSpace   = 0x20
	vkLeft    = 0x25
	vkUp      = 0x26
	vkRight   = 0x27
	vkDown    = 0x28
	vkControl = 0x11
)

func (q *Questionnaire) onKeyDown(focus int, wParam uintptr, pbHandled *bool) int {
	it := q.activeItem()
	if it == nil {
		return 0
	}
	ctrl := wapi.GetAsyncKeyState(vkControl) < 0

	if wParam == vkReturn && ctrl {
		*pbHandled = true
		if q.enabledItemsLast(it) {
			q.submitAll()
		} else {
			q.goNext()
		}
		return 0
	}
	if ctrl {
		return 0
	}

	// 自由输入框里: 方向键与打字都交给原生 Edit; Enter 有内容时继续。
	if focus == it.inputHandle() {
		switch wParam {
		case vkReturn:
			if strings.TrimSpace(it.input.Value()) != "" {
				*pbHandled = true
				q.confirm(it)
			}
		case vkUp, vkDown:
			if strings.TrimSpace(it.input.Value()) == "" {
				*pbHandled = true
				dir := 1
				if wParam == vkUp {
					dir = -1
				}
				it.moveFocusFrom(nil, dir)
			}
		}
		return 0
	}

	switch wParam {
	case vkReturn:
		// 只在"有答案的答案控件"上继续; 按钮上的 Enter 交给原生按钮, 避免触发两次。
		if row := it.rowByHandle(focus); row != nil {
			if row.checked {
				*pbHandled = true
				q.confirm(it)
			}
		}
		return 0
	case vkSpace:
		if row := it.rowByHandle(focus); row != nil && !row.disabled() {
			*pbHandled = true
			q.selectChoice(row)
			return 0
		}
	case vkUp, vkDown:
		dir := 1
		if wParam == vkUp {
			dir = -1
		}
		if it.moveFocusFrom(it.rowByHandle(focus), dir) {
			*pbHandled = true
		}
		return 0
	case vkLeft, vkRight:
		if wParam == vkLeft {
			*pbHandled = true
			q.goPrevious()
			return 0
		}
		if it.status() != QuestionnaireUnanswered {
			*pbHandled = true
			q.goNext()
		}
		return 0
	}

	// 字母/数字快捷键: 分配时已跳过禁用项; 焦点在文本输入框时上面已提前返回。
	if q.shortcuts != QuestionnaireShortcutsNone {
		if shortcut := questionnaireKeyOf(q.shortcuts, wParam); shortcut != "" {
			if row := it.rowByShortcut(shortcut); row != nil && !row.disabled() {
				*pbHandled = true
				xui.NoteKeyboard()
				xui.FocusElement(row.Handle)
				q.selectChoice(row)
			}
		}
	}
	return 0
}

// confirm 对应 Enter / Ctrl+Enter: 校验当前题后前进, 最后一题则提交。
func (q *Questionnaire) confirm(it *questionnaireItem) {
	if q.enabledItemsLast(it) {
		q.submitAll()
		return
	}
	q.goNext()
}

func (q *Questionnaire) enabledItemsLast(it *questionnaireItem) bool {
	enabled := q.enabledItems()
	return len(enabled) > 0 && enabled[len(enabled)-1].def.Name == it.def.Name
}

func questionnaireKeyOf(mode QuestionnaireShortcuts, wParam uintptr) string {
	switch mode {
	case QuestionnaireShortcutsLetters:
		if wParam >= 'A' && wParam <= 'Z' {
			return string(rune(wParam))
		}
	case QuestionnaireShortcutsNumbers:
		if wParam >= '1' && wParam <= '9' {
			return string(rune(wParam))
		}
	}
	return ""
}

func (it *questionnaireItem) inputHandle() int {
	if it.input == nil {
		return 0
	}
	return it.input.Handle
}

func (it *questionnaireItem) rowByHandle(hEle int) *questionnaireChoiceRow {
	for _, row := range it.rows {
		if row.Handle == hEle {
			return row
		}
	}
	return nil
}

func (it *questionnaireItem) rowByShortcut(short string) *questionnaireChoiceRow {
	for _, row := range it.rows {
		if row.short == short {
			return row
		}
	}
	return nil
}

// moveFocusFrom 在答案之间移动焦点: from=nil 表示从题目容器出发(先找已填写的答案);
// 方向 1/-1 循环。单选会顺带选中(同原生 radio 的方向键)。
func (it *questionnaireItem) moveFocusFrom(from *questionnaireChoiceRow, dir int) bool {
	type answer struct {
		row *questionnaireChoiceRow
		in  bool
	}
	answers := make([]answer, 0, len(it.rows)+1)
	for _, row := range it.rows {
		if !row.disabled() {
			answers = append(answers, answer{row: row})
		}
	}
	if it.input != nil {
		answers = append(answers, answer{in: true})
	}
	if len(answers) == 0 {
		return false
	}
	idx := -1
	for i, a := range answers {
		if from != nil && a.row == from {
			idx = i
			break
		}
	}
	var target answer
	if idx < 0 {
		// 题目容器出发: 优先落已选中的选项, 否则向下取首个 / 向上取末个。
		found := false
		for _, a := range answers {
			if a.row != nil && a.row.checked {
				target, found = a, true
				break
			}
		}
		if !found {
			if dir > 0 {
				target = answers[0]
			} else {
				target = answers[len(answers)-1]
			}
		}
	} else {
		target = answers[(idx+dir+len(answers))%len(answers)]
	}
	if target.in {
		xui.NoteKeyboard()
		xui.FocusElement(it.input.Handle)
		return true
	}
	if target.row == nil || target.row == from {
		return false
	}
	xui.NoteKeyboard()
	xui.FocusElement(target.row.Handle)
	if !it.def.Multiple {
		it.q.selectChoice(target.row)
	}
	return true
}
