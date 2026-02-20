package tui

import (
	"fmt"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

type step int

const (
	stepSelectDeployment step = iota
	stepProdLocalConfirm
	stepAddress
	stepGoProxy
	stepConfirm
)

type deploymentType int

const (
	deploymentLocal deploymentType = iota
	deploymentPublic
)

type model struct {
	options cli.Options

	step       step
	deployment deploymentType
	cursor     int
	focus      int

	host string
	port string

	hints      []cli.NetworkCandidate
	activeHint string
	useGoProxy bool
	errMsg     string
	done       bool
	cancelled  bool
}

func newModel(options cli.Options) model {
	host, port := splitHostPortFromURL(options.PublicURL)
	if port == "" {
		port = options.PublicPort
	}
	if strings.TrimSpace(port) == "" {
		port = cli.DefaultPublicPort
	}
	if strings.TrimSpace(host) == "" {
		host = "localhost"
	}

	deployment := deploymentLocal
	cursor := 0
	if options.Mode == cli.ModeProd {
		deployment = deploymentPublic
		cursor = 1
	}

	hints, _ := cli.ListNetworkCandidates()
	activeHint := ""
	if len(hints) > 0 {
		activeHint = hints[0].IP
	}

	if deployment == deploymentPublic && cli.IsLoopbackHost(host) && activeHint != "" {
		host = activeHint
	}
	if deployment == deploymentLocal {
		host = "localhost"
	}

	return model{
		options:    options,
		step:       stepSelectDeployment,
		deployment: deployment,
		cursor:     cursor,
		focus:      0,
		host:       host,
		port:       port,
		hints:      hints,
		activeHint: activeHint,
		useGoProxy: options.UseGoProxyCN,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.step {
		case stepSelectDeployment:
			return m.updateDeploymentStep(msg)
		case stepProdLocalConfirm:
			return m.updateProdLocalConfirmStep(msg)
		case stepAddress:
			return m.updateAddressStep(msg)
		case stepGoProxy:
			return m.updateGoProxyStep(msg)
		case stepConfirm:
			return m.updateConfirmStep(msg)
		}
	}
	return m, nil
}

func (m model) updateDeploymentStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.deployment = deploymentLocal
			m.host = "localhost"
			m.focus = 1
			if m.options.Mode == cli.ModeProd {
				m.step = stepProdLocalConfirm
				m.cursor = 0
				return m, nil
			}
		} else {
			m.deployment = deploymentPublic
			m.focus = 0
			if cli.IsLoopbackHost(m.host) {
				if m.activeHint != "" {
					m.host = m.activeHint
				} else {
					m.host = ""
				}
			}
		}
		m.step = stepAddress
		m.errMsg = ""
	}
	return m, nil
}

func (m model) updateProdLocalConfirmStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.step = stepSelectDeployment
			m.cursor = 1
			return m, nil
		}
		m.step = stepAddress
		m.focus = 1
		m.errMsg = ""
	}
	return m, nil
}

func (m model) updateAddressStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	hostEditable := m.deployment == deploymentPublic

	switch msg.Type {
	case tea.KeyUp:
		if hostEditable {
			m.focus = 0
		}
	case tea.KeyDown:
		m.focus = 1
	case tea.KeyTab, tea.KeyShiftTab:
		if hostEditable {
			if m.focus == 0 {
				m.focus = 1
			} else {
				m.focus = 0
			}
		}
	case tea.KeyBackspace, tea.KeyDelete:
		if m.focus == 0 && hostEditable {
			if len(m.host) > 0 {
				m.host = m.host[:len(m.host)-1]
			}
		} else if m.focus == 1 {
			if len(m.port) > 0 {
				m.port = m.port[:len(m.port)-1]
			}
		}
		m.errMsg = ""
	case tea.KeyEnter:
		host := m.host
		if m.deployment == deploymentLocal {
			host = "localhost"
		}
		publicURL, publicPort, err := cli.NormalizePublicHostPort(host, m.port)
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.options.PublicURL = publicURL
		m.options.PublicPort = publicPort
		m.options.PublicAddressSource = cli.PublicAddressSourceHostPort
		if m.options.Mode == cli.ModeDev {
			m.step = stepGoProxy
			m.cursor = 0
		} else {
			m.step = stepConfirm
			m.cursor = 0
		}
		m.errMsg = ""
	case tea.KeyRunes:
		text := string(msg.Runes)
		if m.focus == 0 && hostEditable {
			m.host += text
		} else if m.focus == 1 {
			m.port += text
		}
		m.errMsg = ""
	}

	return m, nil
}

func (m model) updateGoProxyStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyLeft, tea.KeyRight, tea.KeyTab, tea.KeyShiftTab, tea.KeySpace:
		m.useGoProxy = !m.useGoProxy
	case tea.KeyEnter:
		m.step = stepConfirm
		m.cursor = 0
	}
	return m, nil
}

func (m model) updateConfirmStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.options.UseGoProxyCN = m.useGoProxy
			if m.useGoProxy {
				m.options.GoProxy = "https://goproxy.cn,direct"
			} else {
				m.options.GoProxy = "https://proxy.golang.org,direct"
			}
			m.done = true
			return m, tea.Quit
		case 1:
			m.step = stepAddress
			if m.deployment == deploymentPublic {
				m.focus = 0
			} else {
				m.focus = 1
			}
		case 2:
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.done {
		return "配置已确认，开始安装...\n"
	}
	if m.cancelled {
		return "安装已取消。\n"
	}

	var builder strings.Builder
	builder.WriteString("LunaFox 终端安装向导\n")
	builder.WriteString("按 Ctrl+C 或 Esc 取消。\n\n")

	switch m.step {
	case stepSelectDeployment:
		builder.WriteString("步骤 1/4: 选择部署方式\n")
		builder.WriteString(menuItem(m.cursor == 0, "本机部署（localhost）") + "\n")
		builder.WriteString(menuItem(m.cursor == 1, "公网部署（IP/域名）") + "\n")
		if m.activeHint != "" {
			builder.WriteString(fmt.Sprintf("\n检测到网卡地址: %s\n", m.activeHint))
		}
		builder.WriteString("\n方向键选择，Enter 确认。\n")
	case stepProdLocalConfirm:
		builder.WriteString("步骤 1.5/4: 生产模式确认\n")
		builder.WriteString("你选择了本机部署（localhost）。这通常会导致远端访问失败。\n")
		builder.WriteString(menuItem(m.cursor == 0, "返回并改为公网部署") + "\n")
		builder.WriteString(menuItem(m.cursor == 1, "我确认继续本机部署") + "\n")
	case stepAddress:
		builder.WriteString("步骤 2/4: 填写访问地址\n")
		if m.deployment == deploymentLocal {
			builder.WriteString("主机: localhost（固定）\n")
		} else {
			builder.WriteString(inputLine("主机", m.host, m.focus == 0) + "\n")
		}
		builder.WriteString(inputLine("端口", m.port, m.focus == 1) + "\n")
		builder.WriteString("\nTab 切换输入项，Enter 下一步。\n")
	case stepGoProxy:
		builder.WriteString("步骤 3/4: 开发模式选项\n")
		status := "关闭"
		if m.useGoProxy {
			status = "开启"
		}
		builder.WriteString(fmt.Sprintf("goproxy.cn: %s\n", status))
		builder.WriteString("空格切换，Enter 下一步。\n")
	case stepConfirm:
		builder.WriteString("步骤 4/4: 确认安装\n")
		deployText := "本机部署"
		if m.deployment == deploymentPublic {
			deployText = "公网部署"
		}
		builder.WriteString(fmt.Sprintf("模式: %s\n", m.options.Mode))
		builder.WriteString(fmt.Sprintf("部署: %s\n", deployText))
		builder.WriteString(fmt.Sprintf("访问地址: %s\n", m.options.PublicURL))
		if m.options.Mode == cli.ModeDev {
			builder.WriteString(fmt.Sprintf("goproxy.cn: %t\n", m.useGoProxy))
		}
		builder.WriteString("\n")
		builder.WriteString(menuItem(m.cursor == 0, "确认并开始安装") + "\n")
		builder.WriteString(menuItem(m.cursor == 1, "返回修改地址") + "\n")
		builder.WriteString(menuItem(m.cursor == 2, "取消安装") + "\n")
	}

	if strings.TrimSpace(m.errMsg) != "" {
		builder.WriteString("\n错误: " + m.errMsg + "\n")
	}

	return builder.String()
}

func menuItem(active bool, label string) string {
	if active {
		return "> " + label
	}
	return "  " + label
}

func inputLine(label string, value string, focused bool) string {
	if focused {
		return fmt.Sprintf("%s: [%s]", label, value)
	}
	return fmt.Sprintf("%s: %s", label, value)
}

func splitHostPortFromURL(raw string) (string, string) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", ""
	}
	return strings.Trim(parsed.Hostname(), "[]"), parsed.Port()
}
