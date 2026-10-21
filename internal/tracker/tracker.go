// Package tracker is tpm's tracker boundary. Every package that reads or
// writes issues goes through the Tracker interface: production uses the
// beads tracker (internal/beads), and tests use it or the fake in
// internal/tracker/fake.
package tracker

import "errors"

// ErrNotFound is returned when a project, issue, file, tag, or milestone does
// not exist.
var ErrNotFound = errors.New("not found")

// Issue is the part of a tracker issue tpm reads.
type Issue struct {
	Project     string   `json:"project"`
	IID         int      `json:"iid"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	State       string   `json:"state"`
	WebURL      string   `json:"web_url"`
	DueDate     string   `json:"due_date,omitempty"`
	Milestone   string   `json:"milestone,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`
	Author      string   `json:"author,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

// HasLabel reports whether the issue carries the label.
func (i Issue) HasLabel(label string) bool {
	for _, l := range i.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// NewIssue is what CreateIssue files. It never names an assignee: only pop
// assigns, and only the person who runs it (srd041 R9.1).
type NewIssue struct {
	Title       string
	Description string
	Labels      []string
	DueDate     string // YYYY-MM-DD, or empty
	Milestone   string // milestone title, or empty
}

// IssueUpdate changes an issue; nil fields are left alone.
type IssueUpdate struct {
	// State is "close" or "reopen", or nil to leave the state alone.
	State       *string
	Description *string
	DueDate     *string
	// Milestone sets the issue's milestone, by title.
	Milestone    *string
	AddLabels    []string
	RemoveLabels []string
}

// IssueQuery filters ListIssues.
type IssueQuery struct {
	State  string // opened, closed, or all
	Labels []string
	Search string
	// UpdatedAfter is a YYYY-MM-DD date, or empty.
	UpdatedAfter string
}

// Note is one comment on an issue.
type Note struct {
	ID     int    `json:"id"`
	Body   string `json:"body"`
	Author string `json:"author"`
	System bool   `json:"system"`
}

// Milestone is a project or group milestone.
type Milestone struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

// Whether EnsureMilestone created, updated, or left a milestone as it was.
const (
	MilestoneCreated   = "created"
	MilestoneUpdated   = "updated"
	MilestoneUnchanged = "unchanged"
)

// Award is an award emoji on an issue and the person who gave it.
type Award struct {
	Name string `json:"name"`
	User string `json:"user"`
}

// Merge is one merged delivery in a project's history, as reports
// read it; on beads a merge commit (srd042 R2.3).
type Merge struct {
	IID          int    `json:"iid"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	State        string `json:"state"`
	WebURL       string `json:"web_url"`
	Author       string `json:"author,omitempty"`
	MergedAt     string `json:"merged_at,omitempty"`
}

// User is a tracker account.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// Label is a group label.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// Read names the tracker state a command read: the tracker repository
// commit, when it was pulled, and whether the command read the clone's cache
// because the repository was unreachable (srd042 R3.3).
type Read struct {
	Commit  string `json:"commit"`
	At      string `json:"at"`
	Offline bool   `json:"offline,omitempty"`
}

// Reader is a tracker that knows what it last read. The beads tracker is
// one; the fake, which reads nothing remote, is not.
type Reader interface {
	LastRead() (Read, bool)
}

// Triager is a tracker that holds admission as srd041 state transitions
// fired by lint, not as a bare label: Triage admits or flags one issue and
// reports whether its state changed. The fake is not one; there admission
// is the needs-triage label alone.
type Triager interface {
	Triage(project string, iid int, admitted bool) (bool, error)
}

// AdmittingCreator creates an issue and records its admission in the same
// tracker write: admit sees the issue as created and returns lint's reasons
// and the note to leave on a failing issue. A tracker whose writes are
// costly offers it, so filing is one write and never leaves an issue
// without its admission.
type AdmittingCreator interface {
	CreateAdmitted(project string, in NewIssue, admit func(Issue) (reasons []string, note string)) (Issue, error)
}

// Tracker is everything tpm asks of the tracker; a new need is a new method here.
type Tracker interface {
	GetFile(project, path, ref string) ([]byte, error)
	// ListDir returns the paths of the files directly under dir at ref.
	ListDir(project, dir, ref string) ([]string, error)
	TagExists(project, tag string) (bool, error)
	GetIssue(project string, iid int) (Issue, error)
	ListIssues(project string, q IssueQuery) ([]Issue, error)
	CreateIssue(project string, in NewIssue) (Issue, error)
	UpdateIssue(project string, iid int, u IssueUpdate) (Issue, error)
	ListNotes(project string, iid int) ([]Note, error)
	CreateNote(project string, iid int, body string) error
	// FindMilestone looks in the project, then in its ancestor groups.
	FindMilestone(project, title string) (Milestone, error)
	// ActiveMilestones lists the active milestones of the project and its
	// ancestor groups.
	ActiveMilestones(project string) ([]Milestone, error)
	// EnsureMilestone makes the group's milestone titled m.Title carry m's
	// description and dates: it creates the milestone when the group has
	// none by that title and updates it otherwise, so planning a release
	// twice leaves one milestone (srd001 R7.1, srd031 R4.2). It returns
	// MilestoneCreated, MilestoneUpdated, or MilestoneUnchanged.
	EnsureMilestone(group string, m Milestone) (string, error)
	// Merges lists the merges into the project since a YYYY-MM-DD day,
	// from its midnight UTC, or since an RFC 3339 instant, by author or,
	// with author empty, by anyone. A merge's MergedAt is its UTC day
	// (srd018 R3.2, R8.1).
	Merges(project, author, since string) ([]Merge, error)
	// IssueAwards lists the award emoji on an issue (srd015 R1.2).
	IssueAwards(project string, iid int) ([]Award, error)
	// GroupLabels lists the labels defined on the group itself.
	GroupLabels(group string) ([]Label, error)
	CreateGroupLabel(group string, l Label) error
	UpdateGroupLabel(group string, l Label) error
	// CurrentUser is the account the token belongs to.
	CurrentUser() (User, error)
	// AssignIssue makes user the issue's only assignee. Only pop calls it
	// (srd041 R9.1).
	AssignIssue(project string, iid int, user User) error
	DefaultBranch(project string) (string, error)
	BranchExists(project, branch string) (bool, error)
	CreateBranch(project, branch, ref string) error
}
