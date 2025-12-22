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

var _ resource.Resource = &EnvironmentVariableResource{}
var _ resource.ResourceWithImportState = &EnvironmentVariableResource{}

func NewEnvironmentVariableResource() resource.Resource {
	return &EnvironmentVariableResource{}
}

type EnvironmentVariableResource struct {
	client *client.DokployClient
}

type EnvironmentVariableResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Key           types.String `tfsdk:"key"`
	Value         types.String `tfsdk:"value"`
	Scope         types.String `tfsdk:"scope"`
}

func (r *EnvironmentVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variable"
}

func (r *EnvironmentVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"application_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope": schema.StringAttribute{
				Optional: true,
				Computed: true,
				// Default: "run_time",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *EnvironmentVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentVariableResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Scope.IsUnknown() || plan.Scope.IsNull() {
		plan.Scope = types.StringValue("run_time")
	}

	// Check if already exists (strict check)
	vars, err := r.client.GetVariablesByApplication(plan.ApplicationID.ValueString())
	if err == nil {
		for _, v := range vars {
			if v.Key == plan.Key.ValueString() {
				resp.Diagnostics.AddWarning("Variable already exists", fmt.Sprintf("Environment variable with key '%s' already exists for application '%s'. It will be overwritten.", plan.Key.ValueString(), plan.ApplicationID.ValueString()))
			}
		}
	}

	variable, err := r.client.CreateVariable(
		plan.ApplicationID.ValueString(),
		plan.Key.ValueString(),
		plan.Value.ValueString(),
		plan.Scope.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating variable", err.Error())
		return
	}

	plan.ID = types.StringValue(variable.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *EnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentVariableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vars, err := r.client.GetVariablesByApplication(state.ApplicationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading variables", err.Error())
		return
	}

	found := false
	for _, v := range vars {
		// Matching by ID is safest
		if v.ID == state.ID.ValueString() {
			state.Key = types.StringValue(v.Key)
			state.Scope = types.StringValue(v.Scope)
			// Value is sensitive, we might not get it back or want to compare it.
			// Ideally we don't update state Value from API to avoid showing it in plan/drift if API masks it.
			// But if API returns it, we could.
			// Let's keep state value as is.
			found = true
			break
		}
	}

	if !found {
		// Fallback: match by key?
		// If ID changed but key is same (recreated outside TF?)
		// But ID is primary.
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *EnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No update support
}

func (r *EnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EnvironmentVariableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteVariable(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting variable", err.Error())
		return
	}
}

func (r *EnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
