package git_repo

import (
	"path/filepath"

	"github.com/werf/werf/pkg/werf"
)

const GitWorkTreeCacheVersion = "8"

func GetWorkTreeCacheDir() string {
	return filepath.Join(werf.GetLocalCacheDir(), "git_worktrees", GitWorkTreeCacheVersion)
}
