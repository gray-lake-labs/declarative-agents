package beads

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// The srd041 states.
const (
	Open        = "open"
	NeedsTriage = "needs-triage"
	Ready       = "ready"
	Taken       = "taken"
	InProgress  = "in-progress"
	Closed      = "closed"
)

// The srd041 overlays, held as labels. A row's To is "+overlay" to set one
// and "-overlay" to clear it.
const (
	Waiting  = "waiting"
	Blocked  = "blocked"
	NeedsRun = "needs-run"
)

// The srd041 actors.
const (
	ActorPerson     = "person"
	ActorPop        = "pop"
	ActorMerge      = "merge"
	ActorTrackGoals = "track-goals"
	ActorHandoff    = "handoff"
	ActorLint       = "lint"
)

// The srd041 assignee effects.
const (
	EffectNone  = "none"
	EffectSet   = "set"
	EffectClear = "clear"
)

// KindObjective restricts a row to objective issues.
const KindObjective = "objective"

// objectiveMarker opens the marker tpm goals writes into every objective
// issue it files (srd022 R2.3).
const objectiveMarker = "<!-- tpm goals "

// ErrRefused is wrapped by every transition the table does not list.
var ErrRefused = errors.New("refused")

// Row is one row of the srd041 transitions table.
type Row struct {
	From     []string
	To       string
	Actor    string
	Assignee string
	// Requires is a label the issue must carry for the row to apply.
	Requires string
	// Kind restricts the row to a kind of issue.
	Kind string
}

var anyOpen = []string{Open, NeedsTriage, Ready, Taken, InProgress}

// Table is the complete srd041 state machine (R8.1): every transition the
// tracker makes is one of these rows, fired by the row's one actor.
var Table = []Row{
	{From: []string{Open}, To: NeedsTriage, Actor: ActorLint, Assignee: EffectNone},
	{From: []string{Open}, To: Ready, Actor: ActorLint, Assignee: EffectNone},
	{From: []string{Open}, To: Taken, Actor: ActorPop, Assignee: EffectSet},
	{From: []string{NeedsTriage}, To: Ready, Actor: ActorLint, Assignee: EffectNone},
	{From: []string{NeedsTriage}, To: Taken, Actor: ActorLint, Assignee: EffectNone},
	{From: []string{Ready}, To: Taken, Actor: ActorPop, Assignee: EffectSet},
	{From: []string{Taken}, To: Ready, Actor: ActorPerson, Assignee: EffectClear},
	{From: []string{Taken}, To: InProgress, Actor: ActorPerson, Assignee: EffectNone},
	{From: []string{InProgress}, To: Taken, Actor: ActorPerson, Assignee: EffectNone},
	{From: anyOpen, To: "+" + Waiting, Actor: ActorTrackGoals, Assignee: EffectNone},
	{From: anyOpen, To: "-" + Waiting, Actor: ActorTrackGoals, Assignee: EffectNone},
	{From: []string{Taken, InProgress}, To: "+" + Blocked, Actor: ActorPerson, Assignee: EffectNone},
	{From: []string{Taken, InProgress}, To: "-" + Blocked, Actor: ActorPerson, Assignee: EffectNone},
	{From: anyOpen, To: "+" + NeedsRun, Actor: ActorHandoff, Assignee: EffectNone, Requires: "type::recurring"},
	{From: []string{InProgress}, To: Closed, Actor: ActorMerge, Assignee: EffectNone},
	{From: anyOpen, To: Closed, Actor: ActorTrackGoals, Assignee: EffectNone, Kind: KindObjective},
	{From: anyOpen, To: Closed, Actor: ActorPerson, Assignee: EffectNone},
	{From: []string{Closed}, To: Open, Actor: ActorPerson, Assignee: EffectNone},
}

func isObjective(i Issue) bool {
	return i.HasLabel("plane::management") && strings.Contains(i.Description, objectiveMarker)
}

// row finds the table row that lets actor move i to to.
func row(i Issue, actor, to string) (Row, bool) {
	for _, r := range Table {
		if r.Actor != actor || r.To != to || !slices.Contains(r.From, i.State) {
			continue
		}
		if r.Requires != "" && !i.HasLabel(r.Requires) {
			continue
		}
		if r.Kind == KindObjective && !isObjective(i) {
			continue
		}
		if overlay, ok := strings.CutPrefix(to, "-"); ok && !i.HasLabel(overlay) {
			continue
		}
		return r, true
	}
	return Row{}, false
}

// transition fires actor's move of i to to on behalf of caller, or refuses
// with the attempted transition named. Only pop sets an assignee, and only
// the caller (R9.1); a person's moves on a taken issue are the assignee's
// alone (R9.2).
func transition(i *Issue, actor, to, caller string) error {
	refuse := func(why string) error {
		return fmt.Errorf("%w: %s %s to %s by %s%s (srd041)", ErrRefused, i.ID, i.State, to, actor, why)
	}
	r, ok := row(*i, actor, to)
	if !ok {
		return refuse("")
	}
	if actor == ActorPerson && to != Closed && to != Open && i.Assignee != "" && i.Assignee != caller {
		return refuse(", which only the assignee " + i.Assignee + " may fire")
	}
	if r.Assignee == EffectSet && i.Assignee != "" && i.Assignee != caller {
		return refuse(", taken by " + i.Assignee)
	}

	if overlay, ok := strings.CutPrefix(to, "+"); ok {
		if !i.HasLabel(overlay) {
			i.Labels = append(i.Labels, overlay)
		}
		return nil
	}
	if overlay, ok := strings.CutPrefix(to, "-"); ok {
		i.Labels = slices.DeleteFunc(i.Labels, func(l string) bool { return l == overlay })
		return nil
	}

	switch r.Assignee {
	case EffectSet:
		i.Assignee = caller
	case EffectClear:
		i.Assignee = ""
	}
	i.Labels = slices.DeleteFunc(i.Labels, func(l string) bool { return l == NeedsTriage })
	if to == NeedsTriage {
		i.Labels = append(i.Labels, NeedsTriage)
	}
	i.State, i.StateActor = to, actor
	switch to {
	case Closed:
		i.Status = StatusClosed
	case InProgress:
		i.Status = StatusInProgress
	default:
		i.Status = StatusOpen
		i.ClosedAt, i.CloseReason = "", ""
	}
	return nil
}
