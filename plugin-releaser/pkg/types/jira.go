package types

// JiraConfig holds Jira related configuration
// baseURL: Jira server URL
// username: Jira username
// password: Jira password
type JiraConfig struct {
	BaseURL  string `yaml:"baseURL"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
