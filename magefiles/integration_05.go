package main

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gray-lake-labs/scrum-master/internal/program"
	"github.com/gray-lake-labs/scrum-master/internal/reports"
	"github.com/gray-lake-labs/scrum-master/internal/tracker"
)

func init() {
	integrationTargets = append(integrationTargets, Integration{}.WeeklyStatus, Integration{}.Reports)
}

// weeklyEnv is what both release 01.1 targets share: the system, reports,
// and po-1 repositories, the rig user, and a reports clone scaffolded by
// tpm reports init.
type weeklyEnv struct {
	*rigEnv
	system, reports, po1, user, clone string
}

func newWeeklyEnv(cloneName string) (*weeklyEnv, error) {
	r, err := newRigEnv()
	if err != nil {
		return nil, err
	}
	e := &weeklyEnv{rigEnv: r, user: r.user}
	for key, into := range map[string]*string{"system": &e.system, "reports": &e.reports, "product-owners/po-1": &e.po1} {
		if *into, err = r.project(key); err != nil {
			return nil, err
		}
	}
	po2, err := r.project("product-owners/po-2")
	if err != nil {
		return nil, err
	}
	if err := r.seedSystem(e.system, e.po1, po2); err != nil {
		return nil, err
	}
	if _, err := r.runTPM("issue", "labels", "--group", r.group); err != nil {
		return nil, err
	}
	if e.clone, err = r.clone(e.reports, cloneName); err != nil {
		return nil, err
	}
	_, err = r.runTPM("reports", "init", "--dir", e.clone, "--member", e.user+"="+e.po1, "--system", e.system)
	return e, err
}

// objectiveIssue files an open objective issue in project, as tasks
// would, and returns its reference.
func (e *weeklyEnv) objectiveIssue(project, title string) (string, int, error) {
	i, err := e.tr.CreateIssue(project, tracker.NewIssue{Title: title,
		Description: "An objective.\n\n<!-- tpm goals objective=obj001-reliable-agents:R1 release=" + rigRelease + " -->",
		DueDate:     time.Now().AddDate(0, 0, 21).Format(time.DateOnly)})
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("%s#%d", project, i.IID), i.IID, nil
}

// WeeklyStatus files a status issue for the rig user, works it into a validated report merged in the reports repository with tpm issue merge, and proves the merge closes it and the next week's run recurs it with the next due date (test-rel01.1-weekly-status).
func (Integration) WeeklyStatus() error {
	e, err := newWeeklyEnv("reports-weekly")
	if err != nil {
		return err
	}
	c := &check{}
	objective, _, err := e.objectiveIssue(e.po1, "Score one agent's trajectories")
	if err != nil {
		return err
	}

	var filed []reports.StatusResult
	if err := e.tpmJSON(&filed, "reports", "status-issues", "--dir", e.clone, "--reports", e.reports, "--ref", seedBranch); err != nil {
		return err
	}
	c.expect(len(filed) == 1 && filed[0].Action == "filed" && filed[0].Admitted, "status-issues: %+v", filed)
	if len(filed) != 1 {
		return c.err("weeklyStatus")
	}
	status := filed[0]
	issue, err := e.issue(e.po1, status.IID)
	if err != nil {
		return err
	}
	c.expect(issue.DueDate == status.DueDate, "due date %s, want %s", issue.DueDate, status.DueDate)
	c.expect(len(issue.Assignees) == 0 && strings.Contains(issue.Description, "@"+e.user), "assignees %d; mention present %v", len(issue.Assignees), strings.Contains(issue.Description, "@"+e.user))
	c.expect(strings.Contains(issue.Description, "## Recurrence") && slices.Contains(issue.Labels, "type::recurring"), "recurrence section or label missing")
	var again []reports.StatusResult
	if err := e.tpmJSON(&again, "reports", "status-issues", "--dir", e.clone, "--reports", e.reports, "--ref", seedBranch); err != nil {
		return err
	}
	c.expect(len(again) == 1 && again[0].Action == "exists", "second run: %+v", again)

	var week reports.WeekResult
	if err := e.tpmJSON(&week, "reports", "week", "--project", e.po1, "--user", e.user); err != nil {
		return err
	}
	found := false
	for _, o := range week.SprintGoals {
		found = found || o.Ref == objective
	}
	c.expect(found, "week objectives %+v lack %s", week.SprintGoals, objective)

	start, err := reports.ParseWeek(status.Week)
	if err != nil {
		return err
	}
	path := reports.ReportPath(start, e.user)
	branch := status.Week + "-" + e.user
	report := reports.Render(e.user, status.Week, []reports.Entry{{"objective": objective, "component": "po-1", "status": "at-risk", "complete": "no", "problems": "stale scoring results"}}, nil)
	if err := e.git(e.clone, "switch", "--quiet", "-c", branch); err != nil {
		return err
	}
	writeFileTo(filepath.Join(e.clone, path), strings.Replace(report, "- po-1", "- wrong", 1))
	_, err = e.runTPM("reports", "validate", "--dir", e.clone, path)
	c.expect(err != nil && strings.Contains(err.Error(), "break the report format"), "validate of a mismatched report: %v", err)
	writeFileTo(filepath.Join(e.clone, path), report)
	if _, err := e.runTPM("reports", "validate", "--dir", e.clone, path); err != nil {
		return err
	}
	if err := e.commitPush(e.clone, "Weekly status "+status.Week, path); err != nil {
		return err
	}
	if err := e.approve(e.reports, e.po1, status.IID, branch); err != nil {
		return err
	}
	var merged program.MergeResult
	if err := e.tpmJSON(&merged, "issue", "merge", "--project", e.reports, "--issue-project", e.po1,
		"--issue", fmt.Sprint(status.IID), "--branch", branch, "--dir", e.clone); err != nil {
		return err
	}
	c.expect(merged.Merged, "merge: %+v", merged)
	c.expect(e.closed(e.po1, status.IID), "status issue #%d still open after the report merged", status.IID)

	var next []reports.StatusResult
	nextDay := start.AddDate(0, 0, 7).Format("2006-01-02")
	if err := e.tpmJSON(&next, "reports", "status-issues", "--dir", e.clone, "--reports", e.reports, "--ref", seedBranch, "--date", nextDay); err != nil {
		return err
	}
	c.expect(len(next) == 1 && next[0].Action == "recurred", "next week: %+v", next)
	if len(next) == 1 {
		n, err := e.issue(e.po1, next[0].IID)
		if err != nil {
			return err
		}
		want := start.AddDate(0, 0, 7)
		c.expect(n.DueDate == next[0].DueDate && next[0].Week == reports.ISOWeek(want), "next instance due %s week %s", n.DueDate, next[0].Week)
		c.expect(strings.Contains(n.Description, fmt.Sprintf("%s#%d (due %s): closed by its merge.", e.po1, status.IID, status.DueDate)), "next instance lacks the run in Previous Runs")
	}
	return c.err("weeklyStatus")
}

// Reports aggregates two weeks of fixture reports against the rig's tracker and proves every signal, progress per objective, and that --push files an actionable signal once (test-rel01.1-weekly-status).
func (Integration) Reports() error {
	e, err := newWeeklyEnv("reports-aggregate")
	if err != nil {
		return err
	}
	c := &check{}
	po2, _ := e.project("product-owners/po-2")
	blocked, _, err := e.objectiveIssue(e.po1, "Grader for one agent")
	if err != nil {
		return err
	}
	claimed, _, err := e.objectiveIssue(e.po1, "Grader notes")
	if err != nil {
		return err
	}
	met, metIID, err := e.objectiveIssue(po2, "Fleet rollout")
	if err != nil {
		return err
	}
	if err := e.tr.CreateNote(po2, metIID, "Met.\n\n<!-- tpm program event=met objective=obj001 -->"); err != nil {
		return err
	}
	if err := e.mergedWork(e.po1, "Unreported cache fix", "Closes #999"); err != nil {
		return err
	}

	thisWeek := reports.WeekStart(time.Now())
	lastWeek := thisWeek.AddDate(0, 0, -7)
	since := lastWeek.AddDate(0, 0, -30).Format("2006-01-02")
	put := func(user string, week time.Time, work, blockers []reports.Entry) {
		writeFileTo(filepath.Join(e.clone, reports.ReportPath(week, user)), reports.Render(user, reports.ISOWeek(week), work, blockers))
	}
	w := func(obj, comp, status, complete string) reports.Entry {
		return reports.Entry{"objective": obj, "component": comp, "status": status, "complete": complete, "problems": "none"}
	}
	b := func(what, comp, waiting string) reports.Entry {
		return reports.Entry{"blocker": what, "component": comp, "since": since, "waiting-on": waiting}
	}
	put(e.user, lastWeek, []reports.Entry{w(blocked, "po-1", "at-risk", "no")}, []reports.Entry{b("stale scores", "po-1", "scoring-api")})
	put(e.user, thisWeek, []reports.Entry{w(blocked, "po-1", "blocked", "no"), w(claimed, "po-1", "done", "yes")}, []reports.Entry{b("stale scores", "po-1", "scoring-api")})
	put("fixture-two", thisWeek, []reports.Entry{w(met, "po-2", "done", "yes")}, []reports.Entry{b("no quota", "po-2", "scoring-api")})

	from, to := reports.ISOWeek(lastWeek), reports.ISOWeek(thisWeek)
	var a reports.Aggregation
	if err := e.tpmJSON(&a, "reports", "aggregate", "--dir", e.clone, "--from", from, "--to", to, "--ref", seedBranch); err != nil {
		return err
	}
	kinds := map[string]int{}
	for _, s := range a.Signals {
		kinds[s.Kind]++
		c.expect(!strings.Contains(s.Detail, e.user) && !strings.Contains(s.Detail, "fixture-two"), "signal names a person: %s", s.Detail)
	}
	for _, k := range []string{reports.SignalSharedBlocker, reports.SignalAgingBlocker, reports.SignalSlipped, reports.SignalContradiction, reports.SignalUnreportedWork} {
		c.expect(kinds[k] > 0, "no %s signal: %v", k, kinds)
	}
	progress := map[string]string{}
	for _, p := range a.Progress {
		progress[p.SprintGoal] = p.State
	}
	c.expect(progress[met] == "met" && progress[claimed] == "reported complete", "progress %v", progress)

	var pushed reports.Aggregation
	if err := e.tpmJSON(&pushed, "reports", "aggregate", "--dir", e.clone, "--from", from, "--to", to, "--ref", seedBranch, "--push"); err != nil {
		return err
	}
	filed := 0
	for _, p := range pushed.Pushed {
		if p.Action == "filed" {
			filed++
		}
	}
	c.expect(filed > 0, "--push filed nothing: %+v", pushed.Pushed)
	var repeat reports.Aggregation
	if err := e.tpmJSON(&repeat, "reports", "aggregate", "--dir", e.clone, "--from", from, "--to", to, "--ref", seedBranch, "--push"); err != nil {
		return err
	}
	for _, p := range repeat.Pushed {
		c.expect(p.Action == "exists", "a second --push filed %s again", p.Key)
	}
	return c.err("reports")
}

// mergedWork merges a one-file change into project's main with a merge
// commit whose message carries closes, as work merged without a status
// report would land.
func (e *weeklyEnv) mergedWork(project, title, closes string) error {
	clone, err := e.rigEnv.clone(project, path.Base(project)+"-work")
	if err != nil {
		return err
	}
	branch := "work-" + strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	writeFileTo(filepath.Join(clone, branch+".md"), title+"\n")
	for _, args := range [][]string{{"fetch", "-q", "origin"}, {"checkout", "-q", "-B", branch, "origin/main"}, {"add", "-A"}, {"commit", "-q", "-m", title},
		{"checkout", "-q", "-B", "main", "origin/main"}, {"merge", "-q", "--no-ff", branch, "-m", title + "\n\n" + closes}, {"push", "-q", "origin", "main"}} {
		if err := e.git(clone, args...); err != nil {
			return err
		}
	}
	return nil
}
