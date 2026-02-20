package tui

import (
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

	if m.step != stepSelectDeployment {
		t.Fatalf("unexpected step: %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepAddress {
		t.Fatalf("expected address step, got %v", m.step)
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

func TestModelProdLocalNeedsDoubleConfirm(t *testing.T) {
	m := newModel(cli.Options{
		Mode:       cli.ModeProd,
		PublicURL:  cli.DefaultPublicURL,
		PublicPort: cli.DefaultPublicPort,
	})

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyUp})
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepProdLocalConfirm {
		t.Fatalf("expected prod local confirm step, got %v", m.step)
	}

	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = nextModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != stepAddress {
		t.Fatalf("expected address step, got %v", m.step)
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
