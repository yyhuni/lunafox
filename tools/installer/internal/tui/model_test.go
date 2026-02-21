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
		Mode:       cli.ModeDev,
		PublicURL:  cli.DefaultPublicURL,
		PublicPort: cli.DefaultPublicPort,
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

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepGoProxy {
		t.Fatalf("expected goproxy step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if !m.useGoProxy {
		t.Fatalf("expected goproxy enabled")
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepConfirm {
		t.Fatalf("expected confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.done {
		t.Fatalf("expected done")
	}
	if !m.options.UseGoProxyCN {
		t.Fatalf("expected options goproxy enabled")
	}
	if m.options.PublicURL != "https://localhost:8083" {
		t.Fatalf("unexpected public url: %s", m.options.PublicURL)
	}
}

func TestModelDevPublicFlow(t *testing.T) {
	m := newModel(cli.Options{
		Mode:       cli.ModeDev,
		PublicURL:  cli.DefaultPublicURL,
		PublicPort: cli.DefaultPublicPort,
	})

	m.hostInput.SetValue("10.10.10.10")
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepPort {
		t.Fatalf("expected port step, got %v", m.step)
	}
	if !m.portInput.Focused() {
		t.Fatalf("expected port input focused on port step")
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepGoProxy {
		t.Fatalf("expected goproxy step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepConfirm {
		t.Fatalf("expected confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.done {
		t.Fatalf("expected done")
	}
	if m.options.PublicURL != "https://10.10.10.10:8083" {
		t.Fatalf("unexpected public url: %s", m.options.PublicURL)
	}
}

func TestModelProdLoopbackNeedsDoubleConfirm(t *testing.T) {
	m := newModel(cli.Options{
		Mode:       cli.ModeProd,
		PublicURL:  cli.DefaultPublicURL,
		PublicPort: cli.DefaultPublicPort,
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

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepConfirm {
		t.Fatalf("expected confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.done {
		t.Fatalf("expected done")
	}
	if m.options.PublicURL != "https://localhost:8083" {
		t.Fatalf("unexpected public url: %s", m.options.PublicURL)
	}
}

func TestModelCancelByCtrlC(t *testing.T) {
	m := newModel(cli.Options{Mode: cli.ModeDev, PublicURL: cli.DefaultPublicURL, PublicPort: cli.DefaultPublicPort})
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
		Mode:       cli.ModeDev,
		PublicURL:  cli.DefaultPublicURL,
		PublicPort: cli.DefaultPublicPort,
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
		Mode:       cli.ModeProd,
		PublicURL:  cli.DefaultPublicURL,
		PublicPort: cli.DefaultPublicPort,
	})

	view := m.View()
	if !strings.Contains(view, "分布式功能必须填写公网IP") {
		t.Fatalf("expected prod host step to show distributed public IP hint")
	}
}
