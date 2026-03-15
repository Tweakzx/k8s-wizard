package prompts

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"k8s-wizard/pkg/tools"
)

//go:embed templates/*.yaml
var promptFiles embed.FS

// Prompt represents a loaded prompt template.
type Prompt struct {
	Name        string
	Version     string
	Description string
	System      string
	User        string
}

// Loader manages prompt templates.
type Loader struct {
	prompts    map[string]*Prompt
	tools      map[string]tools.ToolDescription
	categories map[string][]tools.ToolDescription
}

// NewLoader creates a new prompt loader.
func NewLoader() (*Loader, error) {
	loader := &Loader{
		prompts:    make(map[string]*Prompt),
		tools:      make(map[string]tools.ToolDescription),
		categories: make(map[string][]tools.ToolDescription),
	}

	if err := loader.loadEmbedded(); err != nil {
		return nil, err
	}

	return loader, nil
}

// loadEmbedded loads prompts from embedded files.
func (l *Loader) loadEmbedded() error {
	// Load intent prompt
	intentData, err := promptFiles.ReadFile("templates/intent.yaml")
	if err != nil {
		return fmt.Errorf("failed to load intent prompt: %w", err)
	}

	var intentPrompt struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
		System      string `yaml:"system_prompt"`
		User        string `yaml:"user_prompt"`
	}

	if err := yaml.Unmarshal(intentData, &intentPrompt); err != nil {
		return fmt.Errorf("failed to parse intent prompt: %w", err)
	}

	l.prompts["intent"] = &Prompt{
		Name:        intentPrompt.Name,
		Version:     intentPrompt.Version,
		Description: intentPrompt.Description,
		System:      intentPrompt.System,
		User:        intentPrompt.User,
	}

	// Load tools descriptions (optional, for reference)
	toolsData, err := promptFiles.ReadFile("templates/tools.yaml")
	if err == nil {
		var toolsConfig struct {
			Tools []struct {
				Category    string                  `yaml:"category"`
				Description string                  `yaml:"description"`
				Tools       []tools.ToolDescription `yaml:"tools"`
			} `yaml:"tools"`
		}

		if err := yaml.Unmarshal(toolsData, &toolsConfig); err == nil {
			for _, category := range toolsConfig.Tools {
				for _, tool := range category.Tools {
					tool.Category = category.Category
					l.tools[tool.Name] = tool
					l.categories[category.Category] = append(l.categories[category.Category], tool)
				}
			}
		}
	}

	return nil
}

// GetIntentPrompt returns the formatted intent prompt for a user message.
func (l *Loader) GetIntentPrompt(userMessage string, toolRegistry *tools.Registry) (string, error) {
	prompt, ok := l.prompts["intent"]
	if !ok {
		return "", fmt.Errorf("intent prompt not found")
	}

	data := make(map[string]interface{})
	data["UserMessage"] = userMessage

	// Use tool descriptions from registry if provided, otherwise use static descriptions
	if toolRegistry != nil {
		data["ToolDescriptions"] = toolRegistry.GetLLMDescriptions()
	} else {
		data["ToolDescriptions"] = l.formatToolDescriptions()
	}

	// Render user prompt with data
	tmpl, err := template.New("intent").Parse(prompt.User)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Prepend system prompt to user prompt
	fullPrompt := prompt.System + "\n\n" + buf.String()
	return fullPrompt, nil
}

// formatToolDescriptions formats tool descriptions for LLM prompting.
func (l *Loader) formatToolDescriptions() string {
	if len(l.categories) == 0 {
		return "No tools available"
	}

	var descriptions []string
	for category, tools := range l.categories {
		descriptions = append(descriptions, fmt.Sprintf("\n## %s\n", category))
		for _, tool := range tools {
			descriptions = append(descriptions, fmt.Sprintf("- **%s**: %s", tool.Name, tool.Description))
			if len(tool.Parameters) > 0 {
				descriptions = append(descriptions, "  Parameters:")
				for _, param := range tool.Parameters {
					required := ""
					if param.Required {
						required = " (required)"
					}
					if param.Default != nil {
						required = fmt.Sprintf(" (default: %v)", param.Default)
					}
					descriptions = append(descriptions, fmt.Sprintf("    - %s: %s%s", param.Name, param.Description, required))
				}
			}
		}
	}

	return strings.Join(descriptions, "\n")
}

// GetToolDescriptions returns formatted tool descriptions by category.
func (l *Loader) GetToolDescriptions(category string) []tools.ToolDescription {
	if category == "" {
		// Return all tools
		var all []tools.ToolDescription
		for _, tools := range l.categories {
			all = append(all, tools...)
		}
		return all
	}

	return l.categories[category]
}

// UpdateFromRegistry updates tool descriptions from the tool registry.
func (l *Loader) UpdateFromRegistry(registry *tools.Registry) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	// Clear existing static tool descriptions
	l.tools = make(map[string]tools.ToolDescription)
	l.categories = make(map[string][]tools.ToolDescription)

	// The registry will be used directly in GetIntentPrompt via GetLLMDescriptions
	// This method prepares the loader to use dynamic tool descriptions
	return nil
}

// GetPrompt returns a prompt by name.
func (l *Loader) GetPrompt(name string) (*Prompt, bool) {
	prompt, ok := l.prompts[name]
	return prompt, ok
}
