package gitlabrig

import (
	"fmt"
	"os"
	"strings"
)

// Token returns the API token: GITLAB_TOKEN, else the token glab holds for
// the host. Neither is a named error.
func Token(r Runner, host string) (string, error) {
	if t := strings.TrimSpace(os.Getenv("GITLAB_TOKEN")); t != "" {
		return t, nil
	}
	if r.LookPath("glab") == nil {
		// glab prints its status, token included, on stderr; merge it.
		out, _ := r.Run("", nil, "sh", "-c", "glab auth status --hostname "+host+" --show-token 2>&1")
		for _, line := range strings.Split(out, "\n") {
			for _, marker := range []string{"Token found: ", "keyring: ", "Token: "} {
				if _, t, ok := strings.Cut(line, marker); ok {
					if t = strings.TrimSpace(t); t != "" && !strings.Contains(t, " ") {
						return t, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("no GitLab token for %s: set GITLAB_TOKEN or run glab auth login --hostname %s", host, host)
}

// Preflight checks, in order, the configuration, terraform, the token, and
// the token's right to create subgroups in the group. Each failure names
// what is missing; nothing is skipped.
func Preflight(cfg Config, r Runner, newAPI func(token string) API) (API, string, error) {
	if err := cfg.Validate(); err != nil {
		return nil, "", err
	}
	if err := r.LookPath("terraform"); err != nil {
		return nil, "", fmt.Errorf("terraform is not on PATH; install it from https://developer.hashicorp.com/terraform/install")
	}
	token, err := Token(r, cfg.Host())
	if err != nil {
		return nil, "", err
	}
	api := newAPI(token)
	user, err := api.CurrentUserID()
	if err != nil {
		return nil, "", fmt.Errorf("the token for %s is not accepted: %w", cfg.Host(), err)
	}
	level, err := api.AccessLevel(cfg.Group, user)
	if err != nil {
		return nil, "", fmt.Errorf("read access to group %s: %w", cfg.Group, err)
	}
	if level < maintainer {
		return nil, "", fmt.Errorf("the token's user cannot create subgroups in %s (access level %d, need maintainer or owner)", cfg.Group, level)
	}
	return api, token, nil
}
