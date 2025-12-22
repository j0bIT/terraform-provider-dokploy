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

	state.Name = types.StringValue(app.Name)
	if app.ProjectID != "" {
		state.ProjectID = types.StringValue(app.ProjectID)
	}
	if app.EnvironmentID != "" {
		state.EnvironmentID = types.StringValue(app.EnvironmentID)
	}
	if app.RepositoryURL != "" {
		state.RepositoryURL = types.StringValue(app.RepositoryURL)
	}
	if app.Branch != "" {
		state.Branch = types.StringValue(app.Branch)
	}
	if app.BuildType != "" {
		state.BuildType = types.StringValue(app.BuildType)
	}
	if app.DockerfilePath != "" {
		state.DockerfilePath = types.StringValue(app.DockerfilePath)
	}
	if app.DockerContextPath != "" {
		state.DockerContextPath = types.StringValue(app.DockerContextPath)
	}
	if app.DockerBuildStage != "" {
		state.DockerBuildStage = types.StringValue(app.DockerBuildStage)
	}

	// Map new fields
	// Note: API might return empty string for nulls, check if we need to preserve state
	if app.CustomGitUrl != "" {
		state.CustomGitUrl = types.StringValue(app.CustomGitUrl)
	}
	if app.CustomGitBranch != "" {
		state.CustomGitBranch = types.StringValue(app.CustomGitBranch)
	}
	if app.CustomGitSSHKeyId != "" {
		state.CustomGitSSHKeyID = types.StringValue(app.CustomGitSSHKeyId)
	}
	if app.CustomGitBuildPath != "" {
		state.CustomGitBuildPath = types.StringValue(app.CustomGitBuildPath)
	}
	if app.SourceType != "" {
		state.SourceType = types.StringValue(app.SourceType)
	}
	if app.Username != "" {
		state.Username = types.StringValue(app.Username)
	}

	state.AutoDeploy = types.BoolValue(app.AutoDeploy)
	// Don't read password back if not returned or hashed

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
