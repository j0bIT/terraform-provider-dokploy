package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseResource(t *testing.T) {
	host := os.Getenv("DOKPLOY_HOST")
	apiKey := os.Getenv("DOKPLOY_API_KEY")

	if host == "" || apiKey == "" {
		t.Skip("DOKPLOY_HOST and DOKPLOY_API_KEY must be set for acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - Postgres
			{
				Config: testAccPostgresDatabaseConfig("test-postgres-db", "15"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokploy_database.postgres", "name", "test-postgres-db"),
					resource.TestCheckResourceAttr("dokploy_database.postgres", "type", "postgres"),
					resource.TestCheckResourceAttr("dokploy_database.postgres", "database_name", "testdb"),
					resource.TestCheckResourceAttr("dokploy_database.postgres", "database_user", "postgres"),
					resource.TestCheckResourceAttrSet("dokploy_database.postgres", "id"),
					resource.TestCheckResourceAttrSet("dokploy_database.postgres", "app_name"),
				),
			},
			// Update testing
			{
				Config: testAccPostgresDatabaseConfig("test-postgres-db-updated", "15"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokploy_database.postgres", "name", "test-postgres-db-updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "dokploy_database.postgres",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create MySQL database
			{
				Config: testAccMySQLDatabaseConfig("test-mysql-db", "8"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokploy_database.mysql", "name", "test-mysql-db"),
					resource.TestCheckResourceAttr("dokploy_database.mysql", "type", "mysql"),
					resource.TestCheckResourceAttr("dokploy_database.mysql", "database_name", "testdb"),
					resource.TestCheckResourceAttr("dokploy_database.mysql", "database_user", "root"),
					resource.TestCheckResourceAttrSet("dokploy_database.mysql", "id"),
				),
			},
			// Create Redis database
			{
				Config: testAccRedisDatabaseConfig("test-redis-db", "8"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokploy_database.redis", "name", "test-redis-db"),
					resource.TestCheckResourceAttr("dokploy_database.redis", "type", "redis"),
					resource.TestCheckResourceAttrSet("dokploy_database.redis", "id"),
				),
			},
			// Create MongoDB database with replica sets
			{
				Config: testAccMongoDatabaseConfig("test-mongo-db", "15", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokploy_database.mongo", "name", "test-mongo-db"),
					resource.TestCheckResourceAttr("dokploy_database.mongo", "type", "mongo"),
					resource.TestCheckResourceAttr("dokploy_database.mongo", "replica_sets", "true"),
					resource.TestCheckResourceAttrSet("dokploy_database.mongo", "id"),
				),
			},
			// Test with args
			{
				Config: testAccPostgresDatabaseWithArgsConfig("test-postgres-args", "15"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokploy_database.postgres", "name", "test-postgres-args"),
					resource.TestCheckResourceAttr("dokploy_database.postgres", "type", "postgres"),
					resource.TestCheckResourceAttr("dokploy_database.postgres", "args.#", "1"),
				),
			},
			// Test with redeploy_on_update
			{
				Config: testAccPostgresDatabaseWithRedeployConfig("test-postgres-redeploy", "15"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("dokploy_database.postgres", "redeploy_on_update", "true"),
				),
			},
		},
	})
}

func testAccPostgresDatabaseConfig(name, version string) string {
	return fmt.Sprintf(`
provider "dokploy" {
  host    = "%s"
  api_key = "%s"
}

resource "dokploy_project" "test" {
  name = "test-db-project"
}

resource "dokploy_environment" "test" {
  project_id = dokploy_project.test.id
  name       = "test-env"
}

resource "dokploy_database" "postgres" {
  environment_id         = dokploy_environment.test.id
  name                  = "%s"
  type                  = "postgres"
  app_name              = "%s"
  description           = "Test PostgreSQL database"
  database_name         = "testdb"
  database_user         = "postgres"
  database_password     = "testpassword123"
  docker_image          = "postgres:%s"
  args                 = ["--max-connections=100"]
}
`, os.Getenv("DOKPLOY_HOST"), os.Getenv("DOKPLOY_API_KEY"), name, name, version)
}

func testAccPostgresDatabaseWithArgsConfig(name, version string) string {
	return fmt.Sprintf(`
provider "dokploy" {
  host    = "%s"
  api_key = "%s"
}

resource "dokploy_project" "test" {
  name = "test-db-args-project"
}

resource "dokploy_environment" "test" {
  project_id = dokploy_project.test.id
  name       = "test-env"
}

resource "dokploy_database" "postgres" {
  environment_id         = dokploy_environment.test.id
  name                  = "%s"
  type                  = "postgres"
  app_name              = "%s"
  database_name         = "testdb"
  database_user         = "postgres"
  database_password     = "testpassword123"
  docker_image          = "postgres:%s"
  args                 = ["--max-connections=100", "--shared-buffers=256MB"]
}
`, os.Getenv("DOKPLOY_HOST"), os.Getenv("DOKPLOY_API_KEY"), name, name, version)
}

func testAccPostgresDatabaseWithRedeployConfig(name, version string) string {
	return fmt.Sprintf(`
provider "dokploy" {
  host    = "%s"
  api_key = "%s"
}

resource "dokploy_project" "test" {
  name = "test-db-redeploy-project"
}

resource "dokploy_environment" "test" {
  project_id = dokploy_project.test.id
  name       = "test-env"
}

resource "dokploy_database" "postgres" {
  environment_id         = dokploy_environment.test.id
  name                  = "%s"
  type                  = "postgres"
  app_name              = "%s"
  database_name         = "testdb"
  database_user         = "postgres"
  database_password     = "testpassword123"
  docker_image          = "postgres:%s"
  redeploy_on_update    = true
}
`, os.Getenv("DOKPLOY_HOST"), os.Getenv("DOKPLOY_API_KEY"), name, name, version)
}

func testAccMySQLDatabaseConfig(name, version string) string {
	return fmt.Sprintf(`
provider "dokploy" {
  host    = "%s"
  api_key = "%s"
}

resource "dokploy_project" "test" {
  name = "test-mysql-project"
}

resource "dokploy_environment" "test" {
  project_id = dokploy_project.test.id
  name       = "test-env"
}

resource "dokploy_database" "mysql" {
  environment_id           = dokploy_environment.test.id
  name                    = "%s"
  type                    = "mysql"
  app_name                = "%s"
  description             = "Test MySQL database"
  database_name           = "testdb"
  database_user           = "root"
  database_password       = "testpassword123"
  database_root_password = "rootpassword123"
  docker_image            = "mysql:%s"
}
`, os.Getenv("DOKPLOY_HOST"), os.Getenv("DOKPLOY_API_KEY"), name, name, version)
}

func testAccRedisDatabaseConfig(name, version string) string {
	return fmt.Sprintf(`
provider "dokploy" {
  host    = "%s"
  api_key = "%s"
}

resource "dokploy_project" "test" {
  name = "test-redis-project"
}

resource "dokploy_environment" "test" {
  project_id = dokploy_project.test.id
  name       = "test-env"
}

resource "dokploy_database" "redis" {
  environment_id         = dokploy_environment.test.id
  name                  = "%s"
  type                  = "redis"
  app_name              = "%s"
  description           = "Test Redis database"
  database_password     = "testpassword123"
  docker_image          = "redis:%s"
}
`, os.Getenv("DOKPLOY_HOST"), os.Getenv("DOKPLOY_API_KEY"), name, name, version)
}

func testAccMongoDatabaseConfig(name, version string, replicaSets bool) string {
	return fmt.Sprintf(`
provider "dokploy" {
  host    = "%s"
  api_key = "%s"
}

resource "dokploy_project" "test" {
  name = "test-mongo-project"
}

resource "dokploy_environment" "test" {
  project_id = dokploy_project.test.id
  name       = "test-env"
}

resource "dokploy_database" "mongo" {
  environment_id         = dokploy_environment.test.id
  name                  = "%s"
  type                  = "mongo"
  app_name              = "%s"
  description           = "Test MongoDB database"
  database_user         = "mongo"
  database_password     = "testpassword123"
  docker_image          = "mongo:%s"
  replica_sets          = %t
}
`, os.Getenv("DOKPLOY_HOST"), os.Getenv("DOKPLOY_API_KEY"), name, name, version, replicaSets)
}
