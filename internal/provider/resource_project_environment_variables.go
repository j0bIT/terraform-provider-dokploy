package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/j0bit/terraform-provider-dokploy/internal/client"
)

var _ resource.Resource = &ProjectEnvironmentVariablesResource{}
var _ resource.ResourceWithImportState = &ProjectEnvironmentVariablesResource{}

func NewProjectEnvironmentVariablesResource() resource.Resource {
	return &ProjectEnvironmentVariablesResource{}
}

type ProjectEnvironmentVariablesResource struct {
	client *client.DokployClient
}

type ProjectEnvironmentVariablesResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	Variables types.Map    `tfsdk:"variables"`
}

func (r *ProjectEnvironmentVariablesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment_variables"
}

func (r *ProjectEnvironmentVariablesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages all project-level environment variables as a single resource.",
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
			"variables": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Sensitive:   true,
			},
		},
	}
}

func (r *ProjectEnvironmentVariablesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectEnvironmentVariablesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectEnvironmentVariablesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envMap := make(map[string]string)
	resp.Diagnostics.Append(plan.Variables.ElementsAs(ctx, &envMap, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateProjectEnv(plan.ProjectID.ValueString(), func(m map[string]string) {
		for k, v := range envMap {
			m[k] = v
		}
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating project environment variables", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ProjectID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ProjectEnvironmentVariablesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectEnvironmentVariablesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.GetProject(state.ProjectID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project", err.Error())
		return
	}

	envMap := client.ParseEnv(project.Env)
	state.ID = types.StringValue(state.ProjectID.ValueString())
	var diags diag.Diagnostics
	state.Variables, diags = types.MapValueFrom(ctx, types.StringType, envMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ProjectEnvironmentVariablesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectEnvironmentVariablesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	envMap := make(map[string]string)
	resp.Diagnostics.Append(plan.Variables.ElementsAs(ctx, &envMap, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateProjectEnv(plan.ProjectID.ValueString(), func(m map[string]string) {
		for k := range m {
			delete(m, k)
		}
		for k, v := range envMap {
			m[k] = v
		}
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating project environment variables", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ProjectID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ProjectEnvironmentVariablesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectEnvironmentVariablesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateProjectEnv(state.ProjectID.ValueString(), func(m map[string]string) {
		for k := range m {
			delete(m, k)
		}
	})
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting project environment variables", err.Error())
		return
	}
}

func (r *ProjectEnvironmentVariablesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := strings.TrimSpace(req.ID)
	if importID == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID cannot be empty.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), importID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
}
