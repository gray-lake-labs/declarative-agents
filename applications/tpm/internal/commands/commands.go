// Package commands registers every tpm subcommand group in cli.Default, so
// cmd/tpm and the gates that read the dispatch table see the same groups.
package commands

import (
	_ "github.com/gray-lake-labs/scrum-master/internal/backlog"
	_ "github.com/gray-lake-labs/scrum-master/internal/goals"
	_ "github.com/gray-lake-labs/scrum-master/internal/help"
	_ "github.com/gray-lake-labs/scrum-master/internal/install"
	_ "github.com/gray-lake-labs/scrum-master/internal/plan"
	_ "github.com/gray-lake-labs/scrum-master/internal/program"
	_ "github.com/gray-lake-labs/scrum-master/internal/release"
	_ "github.com/gray-lake-labs/scrum-master/internal/reports"
	_ "github.com/gray-lake-labs/scrum-master/internal/scaffold"
	_ "github.com/gray-lake-labs/scrum-master/internal/sprintend"
	_ "github.com/gray-lake-labs/scrum-master/internal/standup"
)
