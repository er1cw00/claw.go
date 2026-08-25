package provider

const defaultDeepSeekBaseURL = "https://api.deepseek.com"

// NewDeepSeekProvider creates a DeepSeek-compatible provider.
// DeepSeek follows the OpenAI chat completions API shape.
func NewDeepSeekProvider(apiKey, baseURL string) LLMProvider {
	if baseURL == "" {
		baseURL = defaultDeepSeekBaseURL
	}
	return newOpenAICompatibleProvider("deepseek", apiKey, baseURL, "")
}
