package gitlabrig

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// API is the slice of the GitLab REST API the rig needs.
type API interface {
	CurrentUserID() (int, error)
	// AccessLevel is the user's effective access level on the group (40 is
	// maintainer, 50 owner); 0 when the user is not a member.
	AccessLevel(group string, user int) (int, error)
	Subgroups(group string) ([]Subgroup, error)
	// GroupID is the id of the group at path, or 0 when there is none.
	GroupID(path string) (int, error)
	// DeleteGroup removes the group and everything in it permanently.
	DeleteGroup(id int) error
	// EnsureInitialCommit commits a README to main when the project has no
	// commits, and reports whether it did.
	EnsureInitialCommit(projectID string) (bool, error)
}

// Subgroup is one direct child group.
type Subgroup struct {
	ID        int       `json:"id"`
	Path      string    `json:"path"`
	FullPath  string    `json:"full_path"`
	CreatedAt time.Time `json:"created_at"`
}

// maintainer is the access level that may create subgroups.
const maintainer = 40

// RESTAPI talks to a GitLab instance with a personal access token.
type RESTAPI struct {
	Base   string
	Token  string
	Client *http.Client
}

func (a RESTAPI) do(method, path string, into any) error {
	req, err := http.NewRequest(method, a.Base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", a.Token)
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, body)
	}
	if into == nil {
		return nil
	}
	return json.Unmarshal(body, into)
}

var errNotFound = fmt.Errorf("not found")

func (a RESTAPI) CurrentUserID() (int, error) {
	var u struct {
		ID int `json:"id"`
	}
	err := a.do(http.MethodGet, "/user", &u)
	return u.ID, err
}

func (a RESTAPI) AccessLevel(group string, user int) (int, error) {
	var m struct {
		AccessLevel int `json:"access_level"`
	}
	err := a.do(http.MethodGet, "/groups/"+url.PathEscape(group)+"/members/all/"+strconv.Itoa(user), &m)
	if err == errNotFound {
		return 0, nil
	}
	return m.AccessLevel, err
}

func (a RESTAPI) Subgroups(group string) ([]Subgroup, error) {
	var all []Subgroup
	for page := 1; ; page++ {
		var batch []Subgroup
		path := fmt.Sprintf("/groups/%s/subgroups?per_page=100&page=%d", url.PathEscape(group), page)
		if err := a.do(http.MethodGet, path, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			return all, nil
		}
	}
}

// EnsureInitialCommit repairs a project GitLab created without the initial
// commit initialize_with_readme asks for, which it sometimes skips.
func (a RESTAPI) EnsureInitialCommit(projectID string) (bool, error) {
	var p struct {
		EmptyRepo bool `json:"empty_repo"`
	}
	if err := a.do(http.MethodGet, "/projects/"+url.PathEscape(projectID), &p); err != nil {
		return false, err
	}
	if !p.EmptyRepo {
		return false, nil
	}
	q := url.Values{"branch": {"main"}, "content": {"# Regression rig repository\n"}, "commit_message": {"Initial commit"}}
	if err := a.do(http.MethodPost, "/projects/"+url.PathEscape(projectID)+"/repository/files/README.md?"+q.Encode(), nil); err != nil {
		return false, fmt.Errorf("initial commit in project %s: %w", projectID, err)
	}
	return true, nil
}

func (a RESTAPI) GroupID(path string) (int, error) {
	var g struct {
		ID int `json:"id"`
	}
	err := a.do(http.MethodGet, "/groups/"+url.PathEscape(path), &g)
	if err == errNotFound {
		return 0, nil
	}
	return g.ID, err
}

// DeleteGroup marks the group for deletion, then removes it permanently.
// On tiers that delay deletion, marking renames the group to
// <path>-deletion_scheduled-<id>, and permanent removal must name that new
// full path, so the group is read again between the two calls. Where
// deletion is immediate, the first call removes it and the read finds none.
func (a RESTAPI) DeleteGroup(id int) error {
	base := "/groups/" + strconv.Itoa(id)
	if err := a.do(http.MethodDelete, base, nil); err != nil {
		if err == errNotFound {
			return nil
		}
		return err
	}
	var g struct {
		FullPath string `json:"full_path"`
	}
	if err := a.do(http.MethodGet, base, &g); err != nil {
		if err == errNotFound {
			return nil
		}
		return err
	}
	q := url.Values{"permanently_remove": {"true"}, "full_path": {g.FullPath}}
	if err := a.do(http.MethodDelete, base+"?"+q.Encode(), nil); err != nil && err != errNotFound {
		return err
	}
	return nil
}
