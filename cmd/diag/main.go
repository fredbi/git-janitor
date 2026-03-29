package main

import (
	"context"
	"fmt"

	"github.com/fredbi/git-janitor/internal/engine/setup"
	"github.com/fredbi/git-janitor/internal/git"
)

func main() {
	r := git.NewRunner("/home/fred/src/github.com/go-swagger/examples")
	ctx := context.Background()

	branches, err := r.Branches(ctx)
	if err != nil {
		fmt.Println("ERROR branches:", err)

		return
	}

	fmt.Println("=== Raw Branches ===")

	for _, b := range branches {
		if !b.IsRemote {
			fmt.Printf("  %s: upstream=%q ahead=%d behind=%d\n",
				b.Name, b.Upstream, b.Ahead, b.Behind)
		}
	}

	fmt.Println("\n=== CollectRepoInfo ===")

	info := git.CollectRepoInfo(ctx, r, "/home/fred/src/github.com/go-swagger/examples")
	fmt.Printf("Kind=%s DefaultBranch=%s Branches=%d\n", info.Kind, info.DefaultBranch, len(info.Branches))

	for _, b := range info.Branches {
		if !b.IsRemote {
			fmt.Printf("  %s: upstream=%q behind=%d ahead=%d merged=%v\n",
				b.Name, b.Upstream, b.Behind, b.Ahead, b.Merged)
		}
	}

	fmt.Println("\n=== Alerts ===")

	e := setup.NewEngine()
	alerts := e.EvaluateRepo(ctx, &info, nil)

	for _, a := range alerts {
		if a.Severity > 0 {
			fmt.Printf("  [%s] %s: %s\n", a.Severity, a.CheckName, a.Summary)

			for _, s := range a.Suggestions {
				fmt.Printf("    -> %s (%s: %v)\n", s.ActionName, s.SubjectKind, s.Subjects)
			}
		}
	}
}
