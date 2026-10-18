package standup

import (
	"flag"
	"fmt"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
)

func init() {
	var args struct {
		project, due, title string
		iid                 int
	}
	cli.Default.Register(&cli.Command{
		Group: "standup", Name: "recur", Summary: "open the next instance of a recurring issue",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&args.project, "project", "", "project path (required)")
			fs.IntVar(&args.iid, "issue", 0, "the recurring issue's IID (required)")
			fs.StringVar(&args.due, "due", "", "the next due date YYYY-MM-DD; empty adds seven days")
			fs.StringVar(&args.title, "title", "", "the next title; empty keeps the issue's")
		},
		Run: func(ctx *cli.Context) error {
			if args.project == "" || args.iid == 0 {
				return fmt.Errorf("%w: recur needs --project and --issue", cli.ErrUsage)
			}
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			r, err := Recur(t, args.project, args.iid, RecurOptions{Title: args.title, DueDate: args.due, Now: time.Now(), DryRun: ctx.DryRun})
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%s#%d recurs as #%d, due %s", r.Project, r.Previous, r.Next, r.DueDate)
			if r.Existing {
				human = fmt.Sprintf("%s#%d already recurs as #%d", r.Project, r.Previous, r.Next)
			}
			return ctx.Emit(r, human)
		},
	})
}
