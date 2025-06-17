package git

// GitOperator defines the interface for Git operations
type GitOperator interface {
	// CreateBranch creates a new branch from the current branch
	CreateBranch(branchName string) error

	// CommitChanges commits all changes with the given message
	CommitChanges(commitMessage string) error

	// PushBranch pushes the current branch to remote origin
	PushBranch() error

	// GetCurrentBranch returns the current branch name
	GetCurrentBranch() (string, error)

	// HasChanges returns true if there are uncommitted changes
	HasChanges() (bool, error)
}
