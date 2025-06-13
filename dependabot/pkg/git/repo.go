package git

import (
	"net/url"
	"strings"
)

// Repository represents a Git repository with its group and name.
type Repository struct {
	Group string
	Repo  string
}

// String returns the repository in the format "group/repo".
func (r *Repository) String() string {
	return strings.Trim(r.Group+"/"+r.Repo, "/")
}

// UrlEncode returns the repository string encoded for use in URLs.
func (r *Repository) UrlEncode() string {
	return url.PathEscape(r.String())
}
