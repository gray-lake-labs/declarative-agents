// Package gitlab is tpm's access to the tracker. Every package that reads or
// writes issues goes through the Tracker interface, so tests use the fake in
// gitlabtest and production uses the official Go client.
package gitlab

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
	State        *string
	Description  *string
	DueDate      *string
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

// MergeRequest is the part of a merge request tpm reads.
type MergeRequest struct {
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
	// MergedMergeRequests lists the merge requests merged into the project
	// since a YYYY-MM-DD date, by author or, with author empty, by anyone
	// (srd018 R3.2, R8.1).
	MergedMergeRequests(project, author, since string) ([]MergeRequest, error)
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
	// Compare returns how many commits head has that base lacks, and the
	// paths those commits change.
	Compare(project, base, head string) (ahead int, paths []string, err error)
	// OpenMergeRequest returns the open merge request from branch, if any.
	OpenMergeRequest(project, branch string) (*MergeRequest, error)
	CreateMergeRequest(project string, mr MergeRequest) (MergeRequest, error)
}
