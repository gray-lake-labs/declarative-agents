// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
// Package commands registers every scrum-master subcommand group in cli.Default, so
// cmd/tpm and the gates that read the dispatch table see the same groups.
package commands

import (
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/backlog"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/goals"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/install"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/plan"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/program"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/release"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/reports"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/scaffold"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/sprintend"
	_ "github.com/gray-lake-labs/declarative-agents/applications/scrum-master/internal/standup"
)
