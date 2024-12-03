package mahogany

import (
	github "github.com/google/go-github/v67/github"
)

type GithubReleaseEvent struct {
	github.ReleaseEvent
}
