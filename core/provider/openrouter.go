package provider

const defaultOpenRouterBaseURL = "https://openrouter.ai/api"

// NewOpenRouterProvider creates an OpenRouter-compatible provider.
// OpenRouter follows the OpenAI chat completions API shape.
func NewOpenRouterProvider(apiKey, baseURL string) LLMProvider {
	if baseURL == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	return newOpenAICompatibleProvider("openrouter", apiKey, baseURL, "")
}
