package cmd

type CommandBase struct {
	name        string
	title       string
	description string
}

func (c *CommandBase) Name() string {
	return c.name
}
func (c *CommandBase) Title() string {
	return c.title
}
func (c *CommandBase) Description() string {
	return c.description
}
func (c *CommandBase) Execute(cmdCtx *CommandContext) (string, bool) {
	return "", false
}
