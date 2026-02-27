package tui

import (
	"net"
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

type step int

const (
	stepHost step = iota
	stepProdLoopbackConfirm
	stepPort
	stepConfirm
)

type model struct {
	options cli.Options

	step   step
	cursor int

	hostInput textinput.Model
	portInput textinput.Model

	errMsg    string
	done      bool
	cancelled bool
	width     int
	height    int
}

func newModel(options cli.Options) model {
	host, port := splitHostPortFromURL(options.PublicURL)
	if !options.HasExplicitPublicAddress() {
		host = ""
	} else {
		host = strings.TrimSpace(host)
	}
	if strings.TrimSpace(port) == "" {
		port = strings.TrimSpace(options.PublicPort)
	}

	hostInput := textinput.New()
	hostInput.Placeholder = "例如 localhost / 192.168.1.10"
	hostInput.SetValue(host)
	hostInput.CharLimit = 100
	hostInput.Width = 36
	hostInput.Prompt = ""

	portInput := textinput.New()
	portInput.Placeholder = "例如: 18443"
	portInput.SetValue(port)
	portInput.CharLimit = 5
	portInput.Width = 10
	portInput.Prompt = ""

	m := model{
		options:   options,
		step:      stepHost,
		cursor:    0,
		hostInput: hostInput,
		portInput: portInput,
	}
	m.syncInputFocus()
	return m
}

func (m model) Init() tea.Cmd {
	switch m.step {
	case stepHost:
		return m.hostInput.Focus()
	case stepPort:
		return m.portInput.Focus()
	default:
		return nil
	}
}

func (m *model) syncInputFocus() tea.Cmd {
	switch m.step {
	case stepHost:
		m.portInput.Blur()
		return m.hostInput.Focus()
	case stepPort:
		m.hostInput.Blur()
		return m.portInput.Focus()
	default:
		m.hostInput.Blur()
		m.portInput.Blur()
		return nil
	}
}

func (m *model) adjustInputWidths() {
	hostWidth := 36
	portWidth := 10

	if m.width > 0 {
		hostWidth = m.width - 30
		if hostWidth < 18 {
			hostWidth = 18
		}
		if hostWidth > 72 {
			hostWidth = 72
		}

		switch {
		case m.width < 42:
			portWidth = 6
		case m.width < 56:
			portWidth = 8
		}
	}

	m.hostInput.Width = hostWidth
	m.portInput.Width = portWidth
}

func batchWith(cmds []tea.Cmd, cmd tea.Cmd) tea.Cmd {
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch m.step {
	case stepHost:
		nextInput, cmd := m.hostInput.Update(msg)
		m.hostInput = nextInput
		cmds = append(cmds, cmd)
	case stepPort:
		nextInput, cmd := m.portInput.Update(msg)
		m.portInput = nextInput
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adjustInputWidths()
		return m, batchWith(cmds, nil)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		}

		var stepCmd tea.Cmd
		switch m.step {
		case stepHost:
			m, stepCmd = m.updateHostStep(msg)
		case stepProdLoopbackConfirm:
			m, stepCmd = m.updateProdLoopbackConfirmStep(msg)
		case stepPort:
			m, stepCmd = m.updatePortStep(msg)
		case stepConfirm:
			m, stepCmd = m.updateConfirmStep(msg)
		}
		return m, batchWith(cmds, stepCmd)
	}

	return m, batchWith(cmds, nil)
}

func (m model) updateHostStep(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyTab:
		host, err := cli.ParsePublicHostInput(m.hostInput.Value())
		if err != nil {
			m.errMsg = "主机不合法: " + err.Error()
			return m, nil
		}
		if host == "" {
			m.errMsg = "主机不合法: 不能为空"
			return m, nil
		}

		m.hostInput.SetValue(host)
		m.errMsg = ""

		if m.options.Mode == cli.ModeProd && cli.IsLoopbackHost(host) {
			m.step = stepProdLoopbackConfirm
			m.cursor = 0
			return m, m.syncInputFocus()
		}

		m.step = stepPort
		return m, m.syncInputFocus()
	}
	return m, nil
}

func (m model) updateProdLoopbackConfirmStep(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown, tea.KeyTab:
		if m.cursor < 1 {
			m.cursor++
		}
	case tea.KeyEnter:
		if m.cursor == 0 {
			m.step = stepHost
			m.errMsg = ""
			return m, m.syncInputFocus()
		}
		m.step = stepPort
		m.errMsg = ""
		return m, m.syncInputFocus()
	}
	return m, nil
}

func (m model) updatePortStep(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyTab:
		host, err := cli.ParsePublicHostInput(m.hostInput.Value())
		if err != nil {
			m.errMsg = "主机不合法: " + err.Error()
			return m, nil
		}
		if host == "" {
			m.errMsg = "主机不合法: 不能为空"
			return m, nil
		}

		port := strings.TrimSpace(m.portInput.Value())
		if port == "" {
			m.errMsg = "端口不合法: 不能为空"
			return m, nil
		}
		if err := cli.ValidatePublicPort(port); err != nil {
			m.errMsg = "端口不合法: " + err.Error()
			return m, nil
		}

		m.portInput.SetValue(port)
		m.options.PublicPort = port
		m.options.PublicURL = cli.BuildPublicURL(host, port)
		m.options.PublicAddressSource = cli.PublicAddressSourceHostPort
		m.errMsg = ""

		m.step = stepConfirm
		m.cursor = 0
		return m, m.syncInputFocus()
	}
	return m, nil
}

func (m model) updateConfirmStep(msg tea.KeyMsg) (model, tea.Cmd) {
	maxCursor := 2
	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown, tea.KeyTab:
		if m.cursor < maxCursor {
			m.cursor++
		}
	case tea.KeyEnter:
		switch m.cursor {
		case 0:
			m.done = true
			return m, tea.Quit
		case 1:
			m.step = stepHost
			m.errMsg = ""
			return m, m.syncInputFocus()
		case 2:
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) totalSteps() int {
	return 3 // 主机 + 端口 + 确认
}

func (m model) currentStepNumber() int {
	switch m.step {
	case stepHost, stepProdLoopbackConfirm:
		return 1
	case stepPort:
		return 2
	case stepConfirm:
		return m.totalSteps()
	default:
		return 1
	}
}

func (m model) stepLabels() []string {
	return []string{"主机", "端口", "确认"}
}

func (m model) renderCompletedSummary() string {
	var b strings.Builder
	current := m.currentStepNumber()

	if current > 1 {
		host, _, _ := m.resolvedHost()
		if host == "" {
			host = "-"
		}
		b.WriteString(renderCompletedItem("主机", host) + "\n")
	}
	if current > 2 {
		port, _, _ := m.resolvedPort()
		b.WriteString(renderCompletedItem("端口", port) + "\n")
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) resolvedHost() (string, bool, string) {
	host, err := cli.ParsePublicHostInput(m.hostInput.Value())
	if err != nil {
		return "", false, "主机错误: " + err.Error()
	}
	if host == "" {
		return "", false, "主机待输入"
	}
	return host, true, "主机合法"
}

func (m model) resolvedPort() (string, bool, string) {
	port := strings.TrimSpace(m.portInput.Value())
	if port == "" {
		return "", false, "端口待输入"
	}
	if err := cli.ValidatePublicPort(port); err != nil {
		return port, false, "端口错误: " + err.Error()
	}
	return port, true, "端口合法"
}

func (m model) hostType() string {
	host, ok, _ := m.resolvedHost()
	if !ok {
		return "-"
	}
	if cli.IsLoopbackHost(host) {
		return "localhost"
	}
	if net.ParseIP(host) != nil {
		return "IP"
	}
	return "未知"
}

func (m model) previewURL() string {
	host, hostOK, _ := m.resolvedHost()
	port, portOK, _ := m.resolvedPort()
	if !hostOK || !portOK {
		return "-"
	}
	return cli.BuildPublicURL(host, port)
}

func (m model) View() string {
	initStyles()

	if m.done {
		return m.renderDoneView()
	}
	if m.cancelled {
		return m.renderCancelledView()
	}

	current := m.currentStepNumber()

	var content strings.Builder
	content.WriteString(m.renderHeader(current))

	var body strings.Builder
	switch m.step {
	case stepHost:
		body.WriteString(m.renderHostStep())
	case stepProdLoopbackConfirm:
		body.WriteString(m.renderProdLoopbackConfirmStep())
	case stepPort:
		body.WriteString(m.renderPortStep())
	case stepConfirm:
		body.WriteString(m.renderConfirmStep())
	}

	if strings.TrimSpace(m.errMsg) != "" {
		body.WriteString("\n" + renderStatusErr(m.errMsg) + "\n")
	}

	content.WriteString(renderCard(body.String(), m.width) + "\n")
	content.WriteString(renderSeparator(m.width) + "\n")
	content.WriteString(helpStyle.Render(m.stepHelpText()))
	return content.String()
}

func (m model) renderHeader(currentStep int) string {
	var b strings.Builder
	b.WriteString("\n")
	titleStr := "  L U N A F O X   I N S T A L L E R      "
	modeStr := "[" + string(m.options.Mode) + "]"
	b.WriteString(titleStyle.Render(titleStr) + subtitleStyle.Render(modeStr) + "\n")
	b.WriteString(renderDoubleSeparator(m.width) + "\n")
	b.WriteString(renderProgressBar(m.stepLabels(), currentStep) + "\n")
	b.WriteString(renderSeparator(m.width) + "\n\n")
	return b.String()
}

func (m model) stepHelpText() string {
	switch m.step {
	case stepHost:
		return "Enter 下一步  │  Ctrl+C 取消"
	case stepProdLoopbackConfirm, stepConfirm:
		return "↑↓ 选择  │  Enter 继续  │  Ctrl+C 取消"
	case stepPort:
		return "Enter 下一步  │  Ctrl+C 取消"
	default:
		return "Enter 继续  │  Ctrl+C 取消"
	}
}

func (m model) renderHostStep() string {
	var b strings.Builder
	_, hostOK, hostStatus := m.resolvedHost()

	b.WriteString(renderInputPrefix("主机项") + m.hostInput.View() + "\n\n")
	if strings.TrimSpace(m.hostInput.Value()) != "" {
		if hostOK {
			b.WriteString(renderStatusOk(hostStatus) + "\n")
		} else {
			b.WriteString(renderStatusErr(hostStatus) + "\n")
		}
	}
	if m.options.Mode == cli.ModeProd {
		b.WriteString(renderHint("分布式功能必须填写公网IP（不要填 localhost）") + "\n")
	} else {
		b.WriteString(renderHint("输入主机(localhost/IPv4)") + "\n")
	}
	return b.String()
}

func (m model) renderProdLoopbackConfirmStep() string {
	var b string
	b += renderHint("你输入了 localhost/loopback，远程通常无法访问") + "\n\n"
	b += renderConfigItem("当前主机", strings.TrimSpace(m.hostInput.Value())) + "\n\n"
	b += renderMenuItem(m.cursor == 0, "返回修改主机") + "\n"
	b += renderMenuItem(m.cursor == 1, "确认继续使用 localhost") + "\n"
	return b
}

func (m model) renderPortStep() string {
	var b strings.Builder
	_, portOK, portStatus := m.resolvedPort()

	b.WriteString(m.renderCompletedSummary())
	b.WriteString(renderInputPrefix("公网端口") + m.portInput.View() + "\n\n")
	if portOK {
		b.WriteString(renderStatusOk(portStatus) + "\n")
	} else {
		b.WriteString(renderStatusErr(portStatus) + "\n")
	}
	b.WriteString(renderHint("输入范围 1-65535（必填）") + "\n")
	return b.String()
}

func (m model) renderConfirmStep() string {
	var b strings.Builder
	b.WriteString(renderConfigItem("运行模式", string(m.options.Mode)) + "\n")
	b.WriteString(renderConfigItem("主机类型", m.hostType()) + "\n")
	b.WriteString(renderConfigItem("访问地址", m.options.PublicURL) + "\n")

	b.WriteString("\n")
	b.WriteString(renderMenuItem(m.cursor == 0, "开始安装") + "\n")
	b.WriteString(renderMenuItem(m.cursor == 1, "返回修改") + "\n")
	b.WriteString(renderMenuItem(m.cursor == 2, "取消退出") + "\n")
	return b.String()
}

func (m model) renderDoneView() string {
	var b strings.Builder

	b.WriteString(m.renderHeader(m.totalSteps() + 1))

	b.WriteString(renderStatusOk(successMsgStyle.Render("配置完成")) + "\n\n")
	b.WriteString(renderConfigItem("模式", string(m.options.Mode)) + "\n")
	b.WriteString(renderConfigItem("主机类型", m.hostType()) + "\n")
	b.WriteString(renderConfigItem("访问地址", m.options.PublicURL) + "\n")

	b.WriteString("\n")
	b.WriteString(renderSeparator(m.width) + "\n")
	b.WriteString(renderHint("准备就绪，接下来将开始拉取镜像并启动服务...") + "\n")
	return b.String()
}

func (m model) renderCancelledView() string {
	return errorMsgStyle.Render("安装已取消。\n")
}

func splitHostPortFromURL(raw string) (string, string) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", ""
	}
	return strings.Trim(parsed.Hostname(), "[]"), parsed.Port()
}
