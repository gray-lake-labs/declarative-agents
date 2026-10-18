package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// DefaultBaseURL is the instance when GITLAB_BASE_URL is unset.
const DefaultBaseURL = "https://gitlab.com/api/v4"

// Client is the Tracker backed by the official Go client.
type Client struct {
	api *gl.Client
}

var _ Tracker = (*Client)(nil)

// BaseURL is GITLAB_BASE_URL, else gitlab.com.
func BaseURL() string {
	if u := strings.TrimSpace(os.Getenv("GITLAB_BASE_URL")); u != "" {
		return u
	}
	return DefaultBaseURL
}

// tokenSource is the glab fallback; tests replace it.
var tokenSource = func(host string) string {
	out, _ := exec.Command("glab", "auth", "status", "--hostname", host, "--show-token").CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		for _, marker := range []string{"Token found: ", "keyring: ", "Token: "} {
			if _, t, ok := strings.Cut(line, marker); ok {
				if t = strings.TrimSpace(t); t != "" && !strings.Contains(t, " ") {
					return t
				}
			}
		}
	}
	return ""
}

// ResolveToken is GITLAB_TOKEN, else the token glab holds for the host of
// baseURL. Neither is a named error.
func ResolveToken(baseURL string) (string, error) {
	if t := strings.TrimSpace(os.Getenv("GITLAB_TOKEN")); t != "" {
		return t, nil
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("GITLAB_BASE_URL %q is not a URL", baseURL)
	}
	if t := tokenSource(u.Host); t != "" {
		return t, nil
	}
	return "", fmt.Errorf("no GitLab token for %s: set GITLAB_TOKEN or run glab auth login --hostname %s", u.Host, u.Host)
}

// New connects to the instance at BaseURL with the resolved token.
func New() (*Client, error) {
	base := BaseURL()
	token, err := ResolveToken(base)
	if err != nil {
		return nil, err
	}
	return newClient(token, base, 60*time.Second, 100*time.Millisecond, 5*time.Second)
}

// newClient builds the client with a per-attempt timeout and the retry
// policy retryCheck states, waiting between waitMin and waitMax.
func newClient(token, base string, timeout, waitMin, waitMax time.Duration) (*Client, error) {
	api, err := gl.NewClient(token, append(ClientOptions(base, timeout), gl.WithCustomRetryWaitMinMax(waitMin, waitMax))...)
	if err != nil {
		return nil, err
	}
	return &Client{api: api}, nil
}

// ClientOptions are the client-go options tpm and the regression rig's
// harness share: the instance, a per-attempt timeout, and retryCheck.
func ClientOptions(base string, timeout time.Duration) []gl.ClientOptionFunc {
	return []gl.ClientOptionFunc{gl.WithBaseURL(base), gl.WithHTTPClient(&http.Client{Timeout: timeout}),
		gl.WithCustomRetry(retryCheck), gl.WithCustomRetryMax(retryMax), gl.WithCustomRetryWaitMinMax(100*time.Millisecond, 5*time.Second)}
}

// retryMax bounds the attempts after the first.
const retryMax = 3

// retryCheck retries a GET whose response did not arrive in time, which
// gitlab.com's slow answers produce, and a 429 or 5xx response, as
// client-go does. A write that timed out is not retried: it may have been
// applied, and a second one could file a duplicate.
func retryCheck(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) && ue.Timeout() && ue.Op == "Get" {
			return true, nil
		}
		return false, err
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500, nil
}

func notFound(resp *gl.Response, err error) error {
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	return err
}

func (c *Client) GetFile(project, file, ref string) ([]byte, error) {
	b, resp, err := c.api.RepositoryFiles.GetRawFile(project, file, &gl.GetRawFileOptions{Ref: gl.Ptr(ref)})
	if err != nil {
		return nil, notFound(resp, err)
	}
	return b, nil
}

func (c *Client) ListDir(project, dir, ref string) ([]string, error) {
	opt := &gl.ListTreeOptions{Path: gl.Ptr(dir), Ref: gl.Ptr(ref), ListOptions: gl.ListOptions{PerPage: 100, Page: 1}}
	var out []string
	for {
		nodes, resp, err := c.api.Repositories.ListTree(project, opt)
		if err != nil {
			return nil, notFound(resp, err)
		}
		for _, n := range nodes {
			if n.Type == "blob" {
				out = append(out, n.Path)
			}
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opt.Page = resp.NextPage
	}
}

func (c *Client) TagExists(project, tag string) (bool, error) {
	_, resp, err := c.api.Tags.GetTag(project, tag)
	if err != nil {
		if errors.Is(notFound(resp, err), ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func toIssue(project string, i *gl.Issue) Issue {
	out := Issue{Project: project, IID: int(i.IID), Title: i.Title, Description: i.Description,
		Labels: []string(i.Labels), State: i.State, WebURL: i.WebURL}
	if i.DueDate != nil {
		out.DueDate = i.DueDate.String()
	}
	if i.Milestone != nil {
		out.Milestone = i.Milestone.Title
	}
	for _, a := range i.Assignees {
		out.Assignees = append(out.Assignees, a.Username)
	}
	if i.Author != nil {
		out.Author = i.Author.Username
	}
	if i.UpdatedAt != nil {
		out.UpdatedAt = i.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (c *Client) GetIssue(project string, iid int) (Issue, error) {
	i, resp, err := c.api.Issues.GetIssue(project, int64(iid))
	if err != nil {
		return Issue{}, notFound(resp, err)
	}
	return toIssue(project, i), nil
}

func (c *Client) ListIssues(project string, q IssueQuery) ([]Issue, error) {
	opt := &gl.ListProjectIssuesOptions{ListOptions: gl.ListOptions{PerPage: 100, Page: 1}}
	if q.State != "" {
		opt.State = gl.Ptr(q.State)
	}
	if len(q.Labels) > 0 {
		opt.Labels = (*gl.LabelOptions)(&q.Labels)
	}
	if q.Search != "" {
		opt.Search = gl.Ptr(q.Search)
	}
	if q.UpdatedAfter != "" {
		t, err := time.Parse("2006-01-02", q.UpdatedAfter)
		if err != nil {
			return nil, fmt.Errorf("since %q is not YYYY-MM-DD", q.UpdatedAfter)
		}
		opt.UpdatedAfter = gl.Ptr(t)
	}
	var out []Issue
	for {
		page, resp, err := c.api.Issues.ListProjectIssues(project, opt)
		if err != nil {
			return nil, notFound(resp, err)
		}
		for _, i := range page {
			out = append(out, toIssue(project, i))
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opt.Page = resp.NextPage
	}
}

func isoDate(s string) (*gl.ISOTime, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("due date %q is not YYYY-MM-DD", s)
	}
	iso := gl.ISOTime(t)
	return &iso, nil
}

func (c *Client) CreateIssue(project string, in NewIssue) (Issue, error) {
	opt := &gl.CreateIssueOptions{Title: gl.Ptr(in.Title), Description: gl.Ptr(in.Description)}
	if len(in.Labels) > 0 {
		labels := gl.LabelOptions(in.Labels)
		opt.Labels = &labels
	}
	if in.DueDate != "" {
		d, err := isoDate(in.DueDate)
		if err != nil {
			return Issue{}, err
		}
		opt.DueDate = d
	}
	if in.Milestone != "" {
		id, err := c.milestoneID(project, in.Milestone)
		if err != nil {
			return Issue{}, err
		}
		opt.MilestoneID = gl.Ptr(id)
	}
	i, resp, err := c.api.Issues.CreateIssue(project, opt)
	if err != nil {
		return Issue{}, notFound(resp, err)
	}
	return toIssue(project, i), nil
}

func (c *Client) UpdateIssue(project string, iid int, u IssueUpdate) (Issue, error) {
	opt := &gl.UpdateIssueOptions{Description: u.Description, StateEvent: u.State}
	if u.DueDate != nil {
		d, err := isoDate(*u.DueDate)
		if err != nil {
			return Issue{}, err
		}
		opt.DueDate = d
	}
	if len(u.AddLabels) > 0 {
		opt.AddLabels = (*gl.LabelOptions)(&u.AddLabels)
	}
	if len(u.RemoveLabels) > 0 {
		opt.RemoveLabels = (*gl.LabelOptions)(&u.RemoveLabels)
	}
	i, resp, err := c.api.Issues.UpdateIssue(project, int64(iid), opt)
	if err != nil {
		return Issue{}, notFound(resp, err)
	}
	return toIssue(project, i), nil
}

func (c *Client) ListNotes(project string, iid int) ([]Note, error) {
	opt := &gl.ListIssueNotesOptions{ListOptions: gl.ListOptions{PerPage: 100, Page: 1}}
	var out []Note
	for {
		page, resp, err := c.api.Notes.ListIssueNotes(project, int64(iid), opt)
		if err != nil {
			return nil, notFound(resp, err)
		}
		for _, n := range page {
			out = append(out, Note{ID: int(n.ID), Body: n.Body, Author: n.Author.Username, System: n.System})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opt.Page = resp.NextPage
	}
}

func (c *Client) CreateNote(project string, iid int, body string) error {
	_, resp, err := c.api.Notes.CreateIssueNote(project, int64(iid), &gl.CreateIssueNoteOptions{Body: gl.Ptr(body)})
	return notFound(resp, err)
}

func (c *Client) GroupLabels(group string) ([]Label, error) {
	opt := &gl.ListGroupLabelsOptions{OnlyGroupLabels: gl.Ptr(true), ListOptions: gl.ListOptions{PerPage: 100, Page: 1}}
	var out []Label
	for {
		page, resp, err := c.api.GroupLabels.ListGroupLabels(group, opt)
		if err != nil {
			return nil, notFound(resp, err)
		}
		for _, l := range page {
			out = append(out, Label{Name: l.Name, Color: l.Color, Description: l.Description})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opt.Page = resp.NextPage
	}
}

func (c *Client) CreateGroupLabel(group string, l Label) error {
	_, resp, err := c.api.GroupLabels.CreateGroupLabel(group, &gl.CreateGroupLabelOptions{Name: gl.Ptr(l.Name), Color: gl.Ptr(l.Color), Description: gl.Ptr(l.Description)})
	return notFound(resp, err)
}

func (c *Client) UpdateGroupLabel(group string, l Label) error {
	_, resp, err := c.api.GroupLabels.UpdateGroupLabel(group, l.Name, &gl.UpdateGroupLabelOptions{Color: gl.Ptr(l.Color), Description: gl.Ptr(l.Description)})
	return notFound(resp, err)
}

func (c *Client) CurrentUser() (User, error) {
	u, resp, err := c.api.Users.CurrentUser()
	if err != nil {
		return User{}, notFound(resp, err)
	}
	return User{ID: int(u.ID), Username: u.Username}, nil
}

func (c *Client) AssignIssue(project string, iid int, user User) error {
	ids := []int64{int64(user.ID)}
	_, resp, err := c.api.Issues.UpdateIssue(project, int64(iid), &gl.UpdateIssueOptions{AssigneeIDs: &ids})
	return notFound(resp, err)
}

func (c *Client) DefaultBranch(project string) (string, error) {
	p, resp, err := c.api.Projects.GetProject(project, nil)
	if err != nil {
		return "", notFound(resp, err)
	}
	return p.DefaultBranch, nil
}

func (c *Client) BranchExists(project, branch string) (bool, error) {
	_, resp, err := c.api.Branches.GetBranch(project, branch)
	if err != nil {
		if errors.Is(notFound(resp, err), ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Client) CreateBranch(project, branch, ref string) error {
	_, resp, err := c.api.Branches.CreateBranch(project, &gl.CreateBranchOptions{Branch: gl.Ptr(branch), Ref: gl.Ptr(ref)})
	return notFound(resp, err)
}

func (c *Client) Compare(project, base, head string) (int, []string, error) {
	cmp, resp, err := c.api.Repositories.Compare(project, &gl.CompareOptions{From: gl.Ptr(base), To: gl.Ptr(head)})
	if err != nil {
		return 0, nil, notFound(resp, err)
	}
	var paths []string
	for _, d := range cmp.Diffs {
		paths = append(paths, d.NewPath)
		if d.OldPath != d.NewPath {
			paths = append(paths, d.OldPath)
		}
	}
	return len(cmp.Commits), paths, nil
}

func (c *Client) OpenMergeRequest(project, branch string) (*MergeRequest, error) {
	mrs, resp, err := c.api.MergeRequests.ListProjectMergeRequests(project, &gl.ListProjectMergeRequestsOptions{SourceBranch: gl.Ptr(branch), State: gl.Ptr("opened")})
	if err != nil {
		return nil, notFound(resp, err)
	}
	if len(mrs) == 0 {
		return nil, nil
	}
	m := mrs[0]
	return &MergeRequest{IID: int(m.IID), Title: m.Title, Description: m.Description, SourceBranch: m.SourceBranch, TargetBranch: m.TargetBranch, State: m.State, WebURL: m.WebURL}, nil
}

func (c *Client) CreateMergeRequest(project string, mr MergeRequest) (MergeRequest, error) {
	m, resp, err := c.api.MergeRequests.CreateMergeRequest(project, &gl.CreateMergeRequestOptions{
		Title: gl.Ptr(mr.Title), Description: gl.Ptr(mr.Description),
		SourceBranch: gl.Ptr(mr.SourceBranch), TargetBranch: gl.Ptr(mr.TargetBranch), RemoveSourceBranch: gl.Ptr(true)})
	if err != nil {
		return MergeRequest{}, notFound(resp, err)
	}
	return MergeRequest{IID: int(m.IID), Title: m.Title, Description: m.Description, SourceBranch: m.SourceBranch, TargetBranch: m.TargetBranch, State: m.State, WebURL: m.WebURL}, nil
}

func (c *Client) FindMilestone(project, title string) (Milestone, error) {
	m, err := c.findMilestone(project, title)
	if err != nil {
		return Milestone{}, err
	}
	out := Milestone{Title: m.title}
	if m.due != nil {
		out.DueDate = m.due.String()
	}
	return out, nil
}

func (c *Client) EnsureMilestone(group string, m Milestone) (string, error) {
	date := func(s string) (*gl.ISOTime, error) {
		if s == "" {
			return nil, nil
		}
		t, err := gl.ParseISOTime(s)
		return &t, err
	}
	start, err := date(m.StartDate)
	if err != nil {
		return "", fmt.Errorf("milestone %s start date: %w", m.Title, err)
	}
	due, err := date(m.DueDate)
	if err != nil {
		return "", fmt.Errorf("milestone %s due date: %w", m.Title, err)
	}
	found, _, err := c.api.GroupMilestones.ListGroupMilestones(group, &gl.ListGroupMilestonesOptions{Title: gl.Ptr(m.Title)})
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		_, _, err := c.api.GroupMilestones.CreateGroupMilestone(group, &gl.CreateGroupMilestoneOptions{
			Title: gl.Ptr(m.Title), Description: gl.Ptr(m.Description), StartDate: start, DueDate: due})
		if err != nil {
			return "", err
		}
		return MilestoneCreated, nil
	}
	cur := found[0]
	same := func(a, b *gl.ISOTime) bool { return (a == nil) == (b == nil) && (a == nil || a.String() == b.String()) }
	if cur.Description == m.Description && same(cur.StartDate, start) && same(cur.DueDate, due) {
		return MilestoneUnchanged, nil
	}
	_, _, err = c.api.GroupMilestones.UpdateGroupMilestone(group, cur.ID, &gl.UpdateGroupMilestoneOptions{
		Description: gl.Ptr(m.Description), StartDate: start, DueDate: due})
	if err != nil {
		return "", err
	}
	return MilestoneUpdated, nil
}

func (c *Client) MergedMergeRequests(project, author, since string) ([]MergeRequest, error) {
	t, err := time.Parse("2006-01-02", since)
	if err != nil {
		return nil, fmt.Errorf("since %q is not YYYY-MM-DD", since)
	}
	opt := &gl.ListProjectMergeRequestsOptions{State: gl.Ptr("merged"), UpdatedAfter: gl.Ptr(t), ListOptions: gl.ListOptions{PerPage: 100}}
	if author != "" {
		opt.AuthorUsername = gl.Ptr(author)
	}
	ms, _, err := c.api.MergeRequests.ListProjectMergeRequests(project, opt)
	if err != nil {
		return nil, err
	}
	var out []MergeRequest
	for _, m := range ms {
		if m.MergedAt == nil || m.MergedAt.Before(t) {
			continue
		}
		by := author
		if m.Author != nil {
			by = m.Author.Username
		}
		out = append(out, MergeRequest{IID: int(m.IID), Title: m.Title, Description: m.Description, SourceBranch: m.SourceBranch, TargetBranch: m.TargetBranch,
			State: m.State, WebURL: m.WebURL, Author: by, MergedAt: m.MergedAt.Format("2006-01-02")})
	}
	return out, nil
}

func (c *Client) IssueAwards(project string, iid int) ([]Award, error) {
	as, _, err := c.api.AwardEmoji.ListIssueAwardEmoji(project, int64(iid), &gl.ListAwardEmojiOptions{ListOptions: gl.ListOptions{PerPage: 100}})
	if err != nil {
		return nil, err
	}
	out := make([]Award, 0, len(as))
	for _, a := range as {
		out = append(out, Award{Name: a.Name, User: a.User.Username})
	}
	return out, nil
}

func (c *Client) ActiveMilestones(project string) ([]Milestone, error) {
	var out []Milestone
	add := func(title string, start, due *gl.ISOTime) {
		m := Milestone{Title: title}
		if start != nil {
			m.StartDate = start.String()
		}
		if due != nil {
			m.DueDate = due.String()
		}
		out = append(out, m)
	}
	ms, _, err := c.api.Milestones.ListMilestones(project, &gl.ListMilestonesOptions{State: gl.Ptr("active")})
	if err != nil {
		return nil, err
	}
	for _, m := range ms {
		add(m.Title, m.StartDate, m.DueDate)
	}
	for group := path.Dir(project); group != "." && group != "/"; group = path.Dir(group) {
		gms, _, err := c.api.GroupMilestones.ListGroupMilestones(group, &gl.ListGroupMilestonesOptions{State: gl.Ptr("active")})
		if err != nil {
			return nil, err
		}
		for _, m := range gms {
			add(m.Title, m.StartDate, m.DueDate)
		}
	}
	return out, nil
}

func (c *Client) milestoneID(project, title string) (int64, error) {
	m, err := c.findMilestone(project, title)
	return m.id, err
}

type milestone struct {
	id    int64
	title string
	due   *gl.ISOTime
}

// findMilestone searches the project, then each ancestor group from the
// nearest up, since a sprint milestone usually lives on the program group.
func (c *Client) findMilestone(project, title string) (milestone, error) {
	ms, _, err := c.api.Milestones.ListMilestones(project, &gl.ListMilestonesOptions{Title: gl.Ptr(title)})
	if err == nil && len(ms) > 0 {
		return milestone{ms[0].ID, ms[0].Title, ms[0].DueDate}, nil
	}
	for group := path.Dir(project); group != "." && group != "/"; group = path.Dir(group) {
		gms, _, err := c.api.GroupMilestones.ListGroupMilestones(group, &gl.ListGroupMilestonesOptions{Title: gl.Ptr(title)})
		if err == nil && len(gms) > 0 {
			return milestone{gms[0].ID, gms[0].Title, gms[0].DueDate}, nil
		}
	}
	return milestone{}, fmt.Errorf("milestone %q in %s or its groups: %w", title, project, ErrNotFound)
}
