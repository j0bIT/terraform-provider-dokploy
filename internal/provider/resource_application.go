package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/j0bit/terraform-provider-dokploy/internal/client"
)

var _ resource.Resource = &ApplicationResource{}
var _ resource.ResourceWithImportState = &ApplicationResource{}

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

type ApplicationResource struct {
	client *client.DokployClient
}

type ApplicationResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	EnvironmentID      types.String `tfsdk:"environment_id"`
	Name               types.String `tfsdk:"name"`
	RepositoryURL      types.String `tfsdk:"repository_url"`
	Branch             types.String `tfsdk:"branch"`
	BuildType          types.String `tfsdk:"build_type"`
	DockerfilePath     types.String `tfsdk:"dockerfile_path"`
	DockerContextPath  types.String `tfsdk:"docker_context_path"`
	DockerBuildStage   types.String `tfsdk:"docker_build_stage"`
	CustomGitUrl       types.String `tfsdk:"custom_git_url"`
	CustomGitBranch    types.String `tfsdk:"custom_git_branch"`
	CustomGitSSHKeyID  types.String `tfsdk:"custom_git_ssh_key_id"`
	CustomGitBuildPath types.String `tfsdk:"custom_git_build_path"`
	SourceType         types.String `tfsdk:"source_type"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	AutoDeploy         types.Bool   `tfsdk:"auto_deploy"`
	DeployOnCreate     types.Bool   `tfsdk:"deploy_on_create"`
	// GitHub Provider fields
	GithubRepository types.String `tfsdk:"github_repository"`
	GithubOwner      types.String `tfsdk:"github_owner"`
	GithubBranch     types.String `tfsdk:"github_branch"`
	GithubBuildPath  types.String `tfsdk:"github_build_path"`
	GithubID         types.String `tfsdk:"github_id"`
	GithubWatchPaths types.List   `tfsdk:"github_watch_paths"`
	EnableSubmodules types.Bool   `tfsdk:"enable_submodules"`
	TriggerType      types.String `tfsdk:"trigger_type"`
}

func (r *ApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"project_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"repository_url": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"branch": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  nil,
			},
			"build_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"dockerfile_path": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"docker_context_path": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"docker_build_stage": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"custom_git_url": schema.StringAttribute{
				Optional: true,
			},
			"custom_git_branch": schema.StringAttribute{
				Optional: true,
			},
			"custom_git_ssh_key_id": schema.StringAttribute{
				Optional: true,
			},
			"custom_git_build_path": schema.StringAttribute{
				Optional: true,
			},
			"source_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"username": schema.StringAttribute{
				Optional: true,
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"auto_deploy": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"deploy_on_create": schema.BoolAttribute{
				Optional: true,
			},
			"github_repository": schema.StringAttribute{
				Optional: true,
			},
			"github_owner": schema.StringAttribute{
				Optional: true,
			},
			"github_branch": schema.StringAttribute{
				Optional: true,
			},
			"github_build_path": schema.StringAttribute{
				Optional: true,
			},
			"github_id": schema.StringAttribute{
				Optional: true,
			},
			"github_watch_paths": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"enable_submodules": schema.BoolAttribute{
				Optional: true,
			},
			"trigger_type": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *ApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.DokployClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Type", fmt.Sprintf("Expected *client.DokployClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Branch.IsUnknown() || plan.Branch.IsNull() {
		plan.Branch = types.StringValue("main")
	}
	if plan.BuildType.IsUnknown() || plan.BuildType.IsNull() {
		plan.BuildType = types.StringValue("nixpacks")
	}
	if plan.DockerfilePath.IsUnknown() || plan.DockerfilePath.IsNull() {
		plan.DockerfilePath = types.StringValue("./Dockerfile")
	}
	if plan.DockerContextPath.IsUnknown() || plan.DockerContextPath.IsNull() {
		plan.DockerContextPath = types.StringValue("/")
	}
	if plan.DockerBuildStage.IsUnknown() || plan.DockerBuildStage.IsNull() {
		plan.DockerBuildStage = types.StringValue("")
	}

	// Default SourceType logic
	if plan.SourceType.IsUnknown() || plan.SourceType.IsNull() {
		if !plan.CustomGitUrl.IsNull() && !plan.CustomGitUrl.IsUnknown() && plan.CustomGitUrl.ValueString() != "" {
			plan.SourceType = types.StringValue("git")
		} else {
			plan.SourceType = types.StringValue("github")
		}
	}

	app := client.Application{
		Name:               plan.Name.ValueString(),
		ProjectID:          plan.ProjectID.ValueString(),
		EnvironmentID:      plan.EnvironmentID.ValueString(),
		RepositoryURL:      plan.RepositoryURL.ValueString(),
		Branch:             plan.Branch.ValueString(),
		BuildType:          plan.BuildType.ValueString(),
		DockerfilePath:     plan.DockerfilePath.ValueString(),
		DockerContextPath:  plan.DockerContextPath.ValueString(),
		DockerBuildStage:   plan.DockerBuildStage.ValueString(),
		CustomGitUrl:       plan.CustomGitUrl.ValueString(),
		CustomGitBranch:    plan.CustomGitBranch.ValueString(),
		CustomGitSSHKeyId:  plan.CustomGitSSHKeyID.ValueString(),
		CustomGitBuildPath: plan.CustomGitBuildPath.ValueString(),
		SourceType:         plan.SourceType.ValueString(),
		Username:           plan.Username.ValueString(),
		Password:           plan.Password.ValueString(),
		AutoDeploy:         plan.AutoDeploy.ValueBool(),
	}

	createdApp, err := r.client.CreateApplication(app)
	if err != nil {
		resp.Diagnostics.AddError("Error creating application", err.Error())
		return
	}

	plan.ID = types.StringValue(createdApp.ID)
	// Update computed fields
	if createdApp.EnvironmentID != "" {
		plan.EnvironmentID = types.StringValue(createdApp.EnvironmentID)
	}
	if createdApp.RepositoryURL != "" {
		plan.RepositoryURL = types.StringValue(createdApp.RepositoryURL)
	} else {
		plan.RepositoryURL = types.StringNull()
	}
	if createdApp.Branch != "" {
		plan.Branch = types.StringValue(createdApp.Branch)
	}
	if createdApp.BuildType != "" {
		plan.BuildType = types.StringValue(createdApp.BuildType)
	}
	if createdApp.SourceType != "" {
		plan.SourceType = types.StringValue(createdApp.SourceType)
	}
	if createdApp.DockerfilePath != "" {
		plan.DockerfilePath = types.StringValue(createdApp.DockerfilePath)
	}
	if createdApp.DockerContextPath != "" {
		plan.DockerContextPath = types.StringValue(createdApp.DockerContextPath)
	}
	if createdApp.DockerBuildStage != "" {
		plan.DockerBuildStage = types.StringValue(createdApp.DockerBuildStage)
	}

	plan.AutoDeploy = types.BoolValue(createdApp.AutoDeploy)

	// Save GitHub provider if GitHub fields are provided
	if !plan.GithubID.IsNull() && !plan.GithubID.IsUnknown() && plan.GithubID.ValueString() != "" {
		githubConfig := map[string]interface{}{
			"enableSubmodules": plan.EnableSubmodules.ValueBool(),
		}

		if !plan.GithubRepository.IsNull() && !plan.GithubRepository.IsUnknown() {
			githubConfig["repository"] = plan.GithubRepository.ValueString()
		}
		if !plan.GithubBranch.IsNull() && !plan.GithubBranch.IsUnknown() {
			githubConfig["branch"] = plan.GithubBranch.ValueString()
		}
		if !plan.GithubOwner.IsNull() && !plan.GithubOwner.IsUnknown() {
			githubConfig["owner"] = plan.GithubOwner.ValueString()
		}
		if !plan.GithubBuildPath.IsNull() && !plan.GithubBuildPath.IsUnknown() {
			githubConfig["buildPath"] = plan.GithubBuildPath.ValueString()
		}
		if !plan.GithubID.IsNull() && !plan.GithubID.IsUnknown() {
			githubConfig["githubId"] = plan.GithubID.ValueString()
		}
		if !plan.TriggerType.IsNull() && !plan.TriggerType.IsUnknown() {
			githubConfig["triggerType"] = plan.TriggerType.ValueString()
		} else {
			githubConfig["triggerType"] = "push"
		}

		// Handle watchPaths list
		if !plan.GithubWatchPaths.IsNull() && !plan.GithubWatchPaths.IsUnknown() {
			var watchPaths []string
			diags := plan.GithubWatchPaths.ElementsAs(ctx, &watchPaths, false)
			if !diags.HasError() && len(watchPaths) > 0 {
				githubConfig["watchPaths"] = watchPaths
			}
		}

		err := r.client.SaveGithubProvider(createdApp.ID, githubConfig)
		if err != nil {
			resp.Diagnostics.AddWarning("GitHub Provider Setup Failed",
				fmt.Sprintf("Application created but GitHub provider configuration failed: %s", err.Error()))
		}
	}

	if !plan.DeployOnCreate.IsNull() && plan.DeployOnCreate.ValueBool() {
		err := r.client.DeployApplication(createdApp.ID)
		if err != nil {
			resp.Diagnostics.AddWarning("Deployment Trigger Failed", fmt.Sprintf("Application created but deployment failed to trigger: %s", err.Error()))
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application", err.Error())
		return
	}

	// Required fields
	state.Name = types.StringValue(app.Name)
	// ProjectID is required but might not be in API response, preserve from state if missing
	if app.ProjectID != "" {
		state.ProjectID = types.StringValue(app.ProjectID)
	}
	// If ProjectID is empty in API response, keep existing state value (don't overwrite)

	// Optional fields
	if app.EnvironmentID != "" {
		state.EnvironmentID = types.StringValue(app.EnvironmentID)
	} else {
		state.EnvironmentID = types.StringNull()
	}

	// Computed fields - always set them, but preserve state if API returns empty
	if app.RepositoryURL != "" {
		state.RepositoryURL = types.StringValue(app.RepositoryURL)
	} else if state.RepositoryURL.IsNull() {
		state.RepositoryURL = types.StringValue("")
	}
	// else keep existing state value

	if app.Branch != "" {
		state.Branch = types.StringValue(app.Branch)
	} else if state.Branch.IsNull() {
		state.Branch = types.StringValue("")
	}

	if app.BuildType != "" {
		state.BuildType = types.StringValue(app.BuildType)
	} else if state.BuildType.IsNull() {
		state.BuildType = types.StringValue("")
	}

	if app.DockerfilePath != "" {
		state.DockerfilePath = types.StringValue(app.DockerfilePath)
	} else if state.DockerfilePath.IsNull() {
		state.DockerfilePath = types.StringValue("")
	}

	if app.DockerContextPath != "" {
		state.DockerContextPath = types.StringValue(app.DockerContextPath)
	} else if state.DockerContextPath.IsNull() {
		state.DockerContextPath = types.StringValue("")
	}

	if app.DockerBuildStage != "" {
		state.DockerBuildStage = types.StringValue(app.DockerBuildStage)
	} else if state.DockerBuildStage.IsNull() {
		state.DockerBuildStage = types.StringValue("")
	}

	if app.SourceType != "" {
		state.SourceType = types.StringValue(app.SourceType)
	} else if state.SourceType.IsNull() {
		state.SourceType = types.StringValue("")
	}

	// AutoDeploy is Computed boolean - always set from API
	state.AutoDeploy = types.BoolValue(app.AutoDeploy)

	// Optional custom git fields
	if app.CustomGitUrl != "" {
		state.CustomGitUrl = types.StringValue(app.CustomGitUrl)
	} else if !state.CustomGitUrl.IsNull() {
		state.CustomGitUrl = types.StringNull()
	}
	if app.CustomGitBranch != "" {
		state.CustomGitBranch = types.StringValue(app.CustomGitBranch)
	} else if !state.CustomGitBranch.IsNull() {
		state.CustomGitBranch = types.StringNull()
	}
	if app.CustomGitSSHKeyId != "" {
		state.CustomGitSSHKeyID = types.StringValue(app.CustomGitSSHKeyId)
	} else if !state.CustomGitSSHKeyID.IsNull() {
		state.CustomGitSSHKeyID = types.StringNull()
	}
	if app.CustomGitBuildPath != "" {
		state.CustomGitBuildPath = types.StringValue(app.CustomGitBuildPath)
	} else if !state.CustomGitBuildPath.IsNull() {
		state.CustomGitBuildPath = types.StringNull()
	}
	if app.Username != "" {
		state.Username = types.StringValue(app.Username)
	} else if !state.Username.IsNull() {
		state.Username = types.StringNull()
	}
	// Don't read password back

	// Optional GitHub Provider fields - only update if they were set in config
	// If state has a value (was configured), update it based on API response
	// If state was null (not configured), keep it null
	if !state.GithubRepository.IsNull() {
		if app.GithubRepository != "" {
			state.GithubRepository = types.StringValue(app.GithubRepository)
		} else {
			state.GithubRepository = types.StringNull()
		}
	}
	if !state.GithubOwner.IsNull() {
		if app.GithubOwner != "" {
			state.GithubOwner = types.StringValue(app.GithubOwner)
		} else {
			state.GithubOwner = types.StringNull()
		}
	}
	if !state.GithubBranch.IsNull() {
		if app.GithubBranch != "" {
			state.GithubBranch = types.StringValue(app.GithubBranch)
		} else {
			state.GithubBranch = types.StringNull()
		}
	}
	if !state.GithubBuildPath.IsNull() {
		if app.GithubBuildPath != "" {
			state.GithubBuildPath = types.StringValue(app.GithubBuildPath)
		} else {
			state.GithubBuildPath = types.StringNull()
		}
	}
	if !state.GithubID.IsNull() {
		if app.GithubID != "" {
			state.GithubID = types.StringValue(app.GithubID)
		} else {
			state.GithubID = types.StringNull()
		}
	}
	if !state.TriggerType.IsNull() {
		if app.TriggerType != "" {
			state.TriggerType = types.StringValue(app.TriggerType)
		} else {
			state.TriggerType = types.StringNull()
		}
	}
	if !state.GithubWatchPaths.IsNull() {
		if len(app.GithubWatchPaths) > 0 {
			watchPathsList, diags := types.ListValueFrom(ctx, types.StringType, app.GithubWatchPaths)
			if !diags.HasError() {
				state.GithubWatchPaths = watchPathsList
			}
		} else {
			state.GithubWatchPaths = types.ListNull(types.StringType)
		}
	}

	// EnableSubmodules - optional boolean, only update if it was set in config
	if !state.EnableSubmodules.IsNull() {
		state.EnableSubmodules = types.BoolValue(app.EnableSubmodules)
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Branch.IsUnknown() {
		plan.Branch = types.StringValue("main")
	}
	if plan.BuildType.IsUnknown() {
		plan.BuildType = types.StringValue("nixpacks")
	}
	if plan.DockerfilePath.IsUnknown() || plan.DockerfilePath.IsNull() {
		plan.DockerfilePath = types.StringValue("./Dockerfile")
	}
	if plan.DockerContextPath.IsUnknown() || plan.DockerContextPath.IsNull() {
		plan.DockerContextPath = types.StringValue("/")
	}
	if plan.DockerBuildStage.IsUnknown() || plan.DockerBuildStage.IsNull() {
		plan.DockerBuildStage = types.StringValue("")
	}

	app := client.Application{
		ID:                 plan.ID.ValueString(),
		Name:               plan.Name.ValueString(),
		ProjectID:          plan.ProjectID.ValueString(),
		EnvironmentID:      plan.EnvironmentID.ValueString(),
		RepositoryURL:      plan.RepositoryURL.ValueString(),
		Branch:             plan.Branch.ValueString(),
		BuildType:          plan.BuildType.ValueString(),
		DockerfilePath:     plan.DockerfilePath.ValueString(),
		DockerContextPath:  plan.DockerContextPath.ValueString(),
		DockerBuildStage:   plan.DockerBuildStage.ValueString(),
		CustomGitUrl:       plan.CustomGitUrl.ValueString(),
		CustomGitBranch:    plan.CustomGitBranch.ValueString(),
		CustomGitSSHKeyId:  plan.CustomGitSSHKeyID.ValueString(),
		CustomGitBuildPath: plan.CustomGitBuildPath.ValueString(),
		SourceType:         plan.SourceType.ValueString(),
		Username:           plan.Username.ValueString(),
		Password:           plan.Password.ValueString(),
		AutoDeploy:         plan.AutoDeploy.ValueBool(),
	}

	updatedApp, err := r.client.UpdateApplication(app)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application", err.Error())
		return
	}

	plan.Name = types.StringValue(updatedApp.Name)
	plan.EnvironmentID = types.StringValue(updatedApp.EnvironmentID)
	plan.AutoDeploy = types.BoolValue(updatedApp.AutoDeploy)

	// Update GitHub provider if GitHub fields are provided
	if !plan.GithubID.IsNull() && !plan.GithubID.IsUnknown() && plan.GithubID.ValueString() != "" {
		githubConfig := map[string]interface{}{
			"enableSubmodules": plan.EnableSubmodules.ValueBool(),
		}

		if !plan.GithubRepository.IsNull() && !plan.GithubRepository.IsUnknown() {
			githubConfig["repository"] = plan.GithubRepository.ValueString()
		}
		if !plan.GithubBranch.IsNull() && !plan.GithubBranch.IsUnknown() {
			githubConfig["branch"] = plan.GithubBranch.ValueString()
		}
		if !plan.GithubOwner.IsNull() && !plan.GithubOwner.IsUnknown() {
			githubConfig["owner"] = plan.GithubOwner.ValueString()
		}
		if !plan.GithubBuildPath.IsNull() && !plan.GithubBuildPath.IsUnknown() {
			githubConfig["buildPath"] = plan.GithubBuildPath.ValueString()
		}
		if !plan.GithubID.IsNull() && !plan.GithubID.IsUnknown() {
			githubConfig["githubId"] = plan.GithubID.ValueString()
		}
		if !plan.TriggerType.IsNull() && !plan.TriggerType.IsUnknown() {
			githubConfig["triggerType"] = plan.TriggerType.ValueString()
		} else {
			githubConfig["triggerType"] = "push"
		}

		// Handle watchPaths list
		if !plan.GithubWatchPaths.IsNull() && !plan.GithubWatchPaths.IsUnknown() {
			var watchPaths []string
			diags := plan.GithubWatchPaths.ElementsAs(ctx, &watchPaths, false)
			if !diags.HasError() && len(watchPaths) > 0 {
				githubConfig["watchPaths"] = watchPaths
			}
		}

		err := r.client.SaveGithubProvider(updatedApp.ID, githubConfig)
		if err != nil {
			resp.Diagnostics.AddWarning("GitHub Provider Update Failed",
				fmt.Sprintf("Application updated but GitHub provider configuration failed: %s", err.Error()))
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteApplication(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting application", err.Error())
		return
	}
}

func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
