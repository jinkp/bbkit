package git

import (
	"os/exec"
	"regexp"
	"strings"
)

type RemoteInfo struct {
	Workspace string
	RepoSlug  string
}

var (
	httpsRemotePattern = regexp.MustCompile(`^https://bitbucket\.org/([^/]+)/([^/]+?)(?:\.git)?$`)
	sshRemotePattern   = regexp.MustCompile(`^git@bitbucket\.org:([^/]+)/([^/]+?)(?:\.git)?$`)
	execCommand        = exec.Command
)

func InferFromGitRemote() (*RemoteInfo, error) {
	output, err := execCommand("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return nil, nil
	}

	remote := strings.TrimSpace(string(output))
	if remote == "" {
		return nil, nil
	}

	for _, pattern := range []*regexp.Regexp{httpsRemotePattern, sshRemotePattern} {
		matches := pattern.FindStringSubmatch(remote)
		if len(matches) == 3 {
			return &RemoteInfo{Workspace: matches[1], RepoSlug: matches[2]}, nil
		}
	}

	return nil, nil
}
