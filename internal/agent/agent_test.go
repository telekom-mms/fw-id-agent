package agent

import (
	"testing"

	"github.com/telekom-mms/fw-id-agent/pkg/config"
)

// TestAgentStartStop tests Start and Stop of Agent
func TestAgentStartStop(_ *testing.T) {
	config := &config.Config{}
	agent := NewAgent(config)
	agent.Start()
	agent.Stop()
}

// TestNewAgent tests NewAgent
func TestNewAgent(t *testing.T) {
	config := &config.Config{}
	agent := NewAgent(config)
	if agent == nil ||
		agent.tnd == nil ||
		agent.done == nil ||
		agent.closed == nil {

		t.Errorf("got nil, want != nil")
	}
	if config != agent.config {
		t.Errorf("got %p, want %p", agent.config, config)
	}
}
