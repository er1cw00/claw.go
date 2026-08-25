package agent

import (
	"context"
	"sync"
	"time"
)

type TurnState string

const (
	TurnStateSetup      TurnState = "setup"
	TurnStateRunning    TurnState = "running"
	TurnStateTools      TurnState = "tools"
	TurnStateFinalizing TurnState = "finalizing"
	TurnStateCompleted  TurnState = "completed"
	TurnStateAborted    TurnState = "aborted"
)

type AgentTurn struct {
	mu         sync.RWMutex
	turnID     string
	agentID    string
	sessionKey string

	channel      string
	chatID       string
	iteration    int
	state        TurnState
	startedAt    time.Time
	finalContent string

	providerCancel context.CancelFunc
	turnCancel     context.CancelFunc
}

func newAgentTurn() *AgentTurn {
	return &AgentTurn{}
}

func (t *AgentTurn) setState(state TurnState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = state
}

func (t *AgentTurn) setIteration(iteration int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.iteration = iteration
}

func (t *AgentTurn) currentIteration() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.iteration
}
