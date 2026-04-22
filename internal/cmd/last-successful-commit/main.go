package main

import (
	"context"
	"fmt"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appext"
	"github.com/google/go-github/v72/github"
	"github.com/spf13/pflag"

	"github.com/bufbuild/plugins/internal/release"
)

func main() {
	appcmd.Main(context.Background(), newRootCommand("last-successful-commit"))
}

func newRootCommand(name string) *appcmd.Command {
	builder := appext.NewBuilder(name)
	f := &flags{}
	return &appcmd.Command{
		Use:   name,
		Short: "Prints the HEAD SHA of the last successful run of a GitHub Actions workflow.",
		Args:  appcmd.NoArgs,
		Run: builder.NewRunFunc(func(ctx context.Context, container appext.Container) error {
			return run(ctx, container, f)
		}),
		BindFlags:           f.Bind,
		BindPersistentFlags: builder.BindRoot,
	}
}

type flags struct {
	owner    string
	repo     string
	workflow string
	branch   string
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.owner, "owner", "bufbuild", "GitHub repository owner")
	flagSet.StringVar(&f.repo, "repo", "plugins", "GitHub repository name")
	flagSet.StringVar(&f.workflow, "workflow", "", "workflow filename (e.g. ci.yml)")
	flagSet.StringVar(&f.branch, "branch", "", "branch to query")
	_ = appcmd.MarkFlagRequired(flagSet, "workflow")
	_ = appcmd.MarkFlagRequired(flagSet, "branch")
}

func run(ctx context.Context, container appext.Container, f *flags) error {
	client := release.NewClient()
	runs, _, err := client.GitHub.Actions.ListWorkflowRunsByFileName(
		ctx,
		f.owner,
		f.repo,
		f.workflow,
		&github.ListWorkflowRunsOptions{
			Branch:      f.branch,
			Status:      "success",
			ListOptions: github.ListOptions{PerPage: 1},
		},
	)
	if err != nil {
		return fmt.Errorf("list workflow runs for %s on %s: %w", f.workflow, f.branch, err)
	}
	if len(runs.WorkflowRuns) == 0 {
		return fmt.Errorf("no successful runs found for workflow %q on branch %q", f.workflow, f.branch)
	}
	fmt.Fprintln(container.Stdout(), runs.WorkflowRuns[0].GetHeadSHA())
	return nil
}
