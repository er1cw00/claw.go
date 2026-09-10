package cmd

import (
	"github.com/er1cw00/claw.go/base/logger"
	ss "github.com/er1cw00/claw.go/service/session"
)

type CommandNew struct {
	CommandBase
}

func NewCommandNew() *CommandNew {
	return &CommandNew{
		CommandBase: CommandBase{
			name:        "/new",
			title:       "New chat",
			description: "Reset this chat and start a fresh conversation.",
		},
	}
}
func (c *CommandNew) Name() string {
	return c.name
}
func (c *CommandNew) Title() string {
	return c.title
}
func (c *CommandNew) Description() string {
	return c.description
}
func (c *CommandNew) Execute(ctx *CommandContext) (string, bool) {
	logger.Debugf("[Command] /new from channel(%s); chat(%s)", ctx.From.Channel, ctx.From.ChatID)
	session := ctx.Session
	session.Reset()
	ss.GetService().Save(session)
	return "🐒 New session started.", true
}
