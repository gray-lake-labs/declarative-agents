package beads

import (
	"fmt"
	"strings"

	"github.com/gray-lake-labs/scrum-master/internal/cli"
)

func init() {
	cli.Default.Register(&cli.Command{
		Group: "tracker", Name: "reconcile",
		Summary: "push writes queued offline and confirm or lose provisional pops (beads tracker)",
		Run: func(ctx *cli.Context) error {
			t, err := ctx.Tracker()
			if err != nil {
				return err
			}
			b, ok := t.(*Tracker)
			if !ok {
				return fmt.Errorf("tpm tracker reconcile serves the beads tracker; set TPM_TRACKER")
			}
			if ctx.DryRun {
				return fmt.Errorf("%w: reconcile pushes queued writes and has no dry run", cli.ErrUsage)
			}
			r, err := b.Store.Reconcile()
			if err != nil {
				return err
			}
			lines := []string{fmt.Sprintf("%d provisional pops reconciled", len(r))}
			for _, x := range r {
				lines = append(lines, x.ID+": "+x.Outcome)
			}
			return ctx.Emit(struct {
				Reconciled []Reconciled `json:"reconciled"`
			}{r}, strings.Join(lines, "\n"))
		},
	})
}
