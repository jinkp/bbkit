package git

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInferFromGitRemote(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		err     error
		want    *RemoteInfo
		wantErr bool
	}{
		{name: "https", output: "https://bitbucket.org/workspace/repo.git\n", want: &RemoteInfo{Workspace: "workspace", RepoSlug: "repo"}},
		{name: "ssh", output: "git@bitbucket.org:workspace/repo.git\n", want: &RemoteInfo{Workspace: "workspace", RepoSlug: "repo"}},
		{name: "non bitbucket", output: "https://github.com/user/repo.git\n", want: nil},
		{name: "git command fails", err: errors.New("not a git repo"), want: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := execCommand
			t.Cleanup(func() { execCommand = original })
			execCommand = func(name string, args ...string) *exec.Cmd {
				return helperCommand(t, tt.output, tt.err != nil)
			}

			got, err := InferFromGitRemote()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func helperCommand(t *testing.T, output string, fail bool) *exec.Cmd {
	t.Helper()
	args := []string{"-test.run=TestHelperProcess", "--", output}
	if fail {
		args = append(args, "fail")
	}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	sep := 0
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if len(args) <= sep+1 {
		os.Exit(0)
	}
	if len(args) > sep+2 && args[sep+2] == "fail" {
		os.Exit(1)
	}
	_, _ = os.Stdout.WriteString(args[sep+1])
	os.Exit(0)
}
