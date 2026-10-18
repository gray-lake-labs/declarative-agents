terraform {
  # gitlabrig points the path at the run directory: one state per run.
  backend "local" {}

  required_providers {
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = "~> 17.0"
    }
  }
}

provider "gitlab" {
  # Token from GITLAB_TOKEN; instance from GITLAB_BASE_URL (defaults to gitlab.com).
}

# Disposable program layout for one regression run
# (docs/engineering/eng01-gitlab-regression-rig.yaml). gitlabrig applies it
# with local state per run and destroys it afterward.

data "gitlab_group" "parent" {
  full_path = var.group
}

resource "gitlab_group" "rig" {
  name             = "rig-${var.run_id}"
  path             = "rig-${var.run_id}"
  parent_id        = data.gitlab_group.parent.group_id
  visibility_level = "private"
}

module "program" {
  source = "../modules/program"

  program_group = gitlab_group.rig.full_path
  scrums = {
    product-owners = [for i in range(1, var.po_repos + 1) : "po-${i}"]
  }
  root_projects = concat(["system"], var.reports ? ["reports"] : [])
  labels = {
    "plane::management" = "#1f77b4"
    "plane::control"    = "#d62728"
    "plane::data"       = "#2ca02c"
    "type::epic"        = "#9467bd"
    "type::recurring"   = "#8c564b"
    "needs-triage"      = "#ff7f0e"
  }
  service_accounts_enabled = var.service_accounts_enabled
  require_pipeline         = false
  protect_projects         = false
}
