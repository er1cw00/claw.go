package cmd

import (
	"strings"
	"sync"

	"github.com/er1cw00/claw.go/model"
	ss "github.com/er1cw00/claw.go/service/session"
)

type CommandContext struct {
	From    *model.Participant
	Session *ss.Session
	Message string
}

type Command interface {
	Name() string
	Title() string
	Description() string
	Execute(cmdCtx *CommandContext) (string, bool)
}

type Registry struct {
	mu       sync.RWMutex
	commands map[string]Command
}

func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command, 0),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(c Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[c.Name()] = c
}

func (r *Registry) Get(cmd string) (Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.commands[cmd]
	return tool, ok
}
func (r *Registry) HandleCommand(sender *model.Participant, msg string, session *ss.Session) (string, bool) {
	for name, cmd := range r.commands {
		if msg[0] == '/' && strings.HasPrefix(msg, name) {
			ctx := &CommandContext{
				From:    sender,
				Message: msg,
				Session: session,
			}

			return cmd.Execute(ctx)
		}
	}
	return "", false
}

// func (bc *BuiltCommand) hasCommandPrefix(msg string) bool {

// }
// func (bc *BuiltCommand) HandleCommand(ctx context.Context, inboundMsg *model.InboundMessage ) {
//     if !bc.hasCommandPrefix(inboundMsg.Content) {
// 		return "", false
// 	}

// }
