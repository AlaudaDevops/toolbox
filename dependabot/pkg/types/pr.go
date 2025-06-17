package types

// PRInfo represents a pull request information
type PRInfo struct {
	// Title is the pull request title
	Title string
	// SourceBranch is the branch containing changes
	SourceBranch string
	// TargetBranch is the branch to merge into (usually main/master)
	TargetBranch string
	// Number is the pull request number
	Number int `json:"number" yaml:"number"`
	// URL is the pull request URL
	URL string `json:"url" yaml:"url"`
}
