package provider

const defaultOpenAIBaseURL = "https://api.openai.com"

// NewOpenAIProvider creates an OpenAI-compatible provider using the official
// OpenAI chat completions endpoint.
func NewOpenAIProvider(apiKey, baseURL string) LLMProvider {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	return newOpenAICompatibleProvider("openai", apiKey, baseURL, "")
}
