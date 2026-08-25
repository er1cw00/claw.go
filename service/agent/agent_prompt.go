package agent

import ()

const SYSTEM_PROMPT = "You are a helpful AI assistant with access to tools.\n" +
	"Use the tools to help the user with file operations and shell commands.\n" +
	"Always read a file before editing it.\n" +
	"When using edit_file, the old_string must match EXACTLY (including whitespace)."

func (agent *Agent) BuildPrompt() string {
	return ""
}
