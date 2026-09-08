package provider

const defaultOpenCodeBaseURL = "https://api.opencode.ai"

// NewOpenCodeProvider creates an OpenCode-compatible provider.
// OpenCode follows the OpenAI chat completions API shape and requires an
// x-opencode-session header on every request. The session ID is supplied per
// request via CompletionRequest.SessionID and added in Complete.
func NewOpenCodeProvider(apiKey, baseURL string) LLMProvider {
	if baseURL == "" {
		baseURL = defaultOpenCodeBaseURL
	}
	return newOpenAICompatibleProvider("opencode", apiKey, baseURL, "")
}
