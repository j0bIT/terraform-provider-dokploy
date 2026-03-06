resource "dokploy_project" "example" {
  name        = "My Project"
  description = "A project managed by Terraform"
}

resource "dokploy_project_environment_variables" "example" {
  project_id = dokploy_project.example.id

  variables = {
    DATABASE_URL = "postgres://db:5432/app"
    LOG_LEVEL    = "info"
  }
}
