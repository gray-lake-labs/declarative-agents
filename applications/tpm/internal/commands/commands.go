// Copyright (c) 2026 Gray Lake Labs
// SPDX-License-Identifier: MIT-0
// Package commands registers every tpm subcommand group in cli.Default, so
// cmd/tpm and the gates that read the dispatch table see the same groups.
package commands

import (
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/backlog"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/goals"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/help"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/install"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/plan"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/program"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/release"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/reports"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/scaffold"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/sprintend"
	_ "github.com/gray-lake-labs/declarative-agents/applications/tpm/internal/standup"
)
