package gitlabrig

import (
	"strings"
	"time"
)

// Stale returns the rig-* subgroups of group created before now minus age.
// Nothing else under the group is ever selected.
func Stale(api API, group string, age time.Duration, now time.Time) ([]Subgroup, error) {
	subs, err := api.Subgroups(group)
	if err != nil {
		return nil, err
	}
	var stale []Subgroup
	for _, s := range subs {
		if strings.HasPrefix(s.Path, "rig-") && s.CreatedAt.Before(now.Add(-age)) {
			stale = append(stale, s)
		}
	}
	return stale, nil
}

// Sweep deletes the stale rig subgroups and returns their full paths.
func Sweep(api API, group string, age time.Duration, now time.Time) ([]string, error) {
	stale, err := Stale(api, group, age, now)
	if err != nil {
		return nil, err
	}
	var deleted []string
	for _, s := range stale {
		if err := api.DeleteGroup(s.ID); err != nil {
			return deleted, err
		}
		deleted = append(deleted, s.FullPath)
	}
	return deleted, nil
}
