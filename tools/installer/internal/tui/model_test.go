package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

func nextModel(t *testing.T, current model, msg tea.KeyMsg) model {
	t.Helper()
	updated, _ := current.Update(msg)
	out, ok := updated.(model)
	if !ok {
		t.Fatalf("unexpected model type: %T", updated)
	}
	return out
}

func TestModelDevLocalFlow(t *testing.T) {
	m := newModel(cli.Options{
		Mode: cli.ModeDev,
	})

	if m.step != stepHost {
		t.Fatalf("expected host step, got %v", m.step)
	}
	if !m.hostInput.Focused() {
		t.Fatalf("expected host input focused on host step")
	}

	m.hostInput.SetValue("localhost")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepPort {
		t.Fatalf("expected port step, got %v", m.step)
	}

	m.portInput.SetValue("18443")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepConfirm {
		t.Fatalf("expected confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.done {
		t.Fatalf("expected done")
	}
	if m.options.PublicURL != "https://localhost:18443" {
		t.Fatalf("unexpected public url: %s", m.options.PublicURL)
	}
}

func TestModelDevPublicFlow(t *testing.T) {
	m := newModel(cli.Options{
		Mode: cli.ModeDev,
	})

	m.hostInput.SetValue("10.10.10.10")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepPort {
		t.Fatalf("expected port step, got %v", m.step)
	}
	if !m.portInput.Focused() {
		t.Fatalf("expected port input focused on port step")
	}

	m.portInput.SetValue("18443")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepConfirm {
		t.Fatalf("expected confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.done {
		t.Fatalf("expected done")
	}
	if m.options.PublicURL != "https://10.10.10.10:18443" {
		t.Fatalf("unexpected public url: %s", m.options.PublicURL)
	}
}

func TestModelProdLoopbackNeedsDoubleConfirm(t *testing.T) {
	m := newModel(cli.Options{
		Mode: cli.ModeProd,
	})

	m.hostInput.SetValue("localhost")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepProdLoopbackConfirm {
		t.Fatalf("expected prod loopback confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepHost {
		t.Fatalf("expected host step after choosing back, got %v", m.step)
	}

	m.hostInput.SetValue("localhost")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepProdLoopbackConfirm {
		t.Fatalf("expected prod loopback confirm step again, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepPort {
		t.Fatalf("expected port step, got %v", m.step)
	}

	m.portInput.SetValue("18443")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepConfirm {
		t.Fatalf("expected confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.done {
		t.Fatalf("expected done")
	}
	if m.options.PublicURL != "https://localhost:18443" {
		t.Fatalf("unexpected public url: %s", m.options.PublicURL)
	}
}

func TestModelCancelByCtrlC(t *testing.T) {
	m := newModel(cli.Options{Mode: cli.ModeDev})
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	out := updated.(model)
	if !out.cancelled {
		t.Fatalf("expected cancelled")
	}
	if cmd == nil {
		t.Fatalf("expected quit command")
	}
}

func TestModelFocusFlowHostToPort(t *testing.T) {
	m := newModel(cli.Options{
		Mode: cli.ModeDev,
	})

	if !m.hostInput.Focused() {
		t.Fatalf("expected host input focused on first step")
	}
	if m.portInput.Focused() {
		t.Fatalf("expected port input blurred on first step")
	}

	m.hostInput.SetValue("localhost")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepPort {
		t.Fatalf("expected port step, got %v", m.step)
	}
	if !m.portInput.Focused() {
		t.Fatalf("expected port input focused on port step")
	}
	if m.hostInput.Focused() {
		t.Fatalf("expected host input blurred on port step")
	}
}

func TestModelProdHostStepShowsPublicIPHint(t *testing.T) {
	m := newModel(cli.Options{
		Mode: cli.ModeProd,
	})

	view := m.View()
	if !strings.Contains(view, "分布式功能必须填写公网IP") {
		t.Fatalf("expected prod host step to show distributed public IP hint")
	}
}

func TestModelRejectsEmptyPort(t *testing.T) {
	m := newModel(cli.Options{Mode: cli.ModeDev})
	m.hostInput.SetValue("10.10.10.10")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepPort {
		t.Fatalf("expected port step, got %v", m.step)
	}

	m.portInput.SetValue("")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepPort {
		t.Fatalf("expected to stay on port step, got %v", m.step)
	}
	if !strings.Contains(m.errMsg, "端口不合法") {
		t.Fatalf("unexpected error message: %s", m.errMsg)
	}
}
