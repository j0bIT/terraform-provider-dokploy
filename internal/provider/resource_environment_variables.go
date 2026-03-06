package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/j0bit/terraform-provider-dokploy/internal/client"
)

var _ resource.Resource = &EnvironmentVariablesResource{}
var _ resource.ResourceWithImportState = &EnvironmentVariablesResource{}

func NewEnvironmentVariablesResource() resource.Resource {
	return &EnvironmentVariablesResource{}
}

type EnvironmentVariablesResource struct {
	client *client.DokployClient
}

type EnvironmentVariablesResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	ComposeID     types.String `tfsdk:"compose_id"`
	Variables     types.Map    `tfsdk:"variables"`
	CreateEnvFile types.Bool   `tfsdk:"create_env_file"`
}

func (r *EnvironmentVariablesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_variables"
}

func (r *EnvironmentVariablesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages all environment variables for a Dokploy application or compose stack as a single resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"application_id": schema.StringAttribute{
				Optional: true,
			},
			"compose_id": schema.StringAttribute{
				Optional: true,
			},
			"variables": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Sensitive:   true,
			},
			"create_env_file": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
		},
	}
}

func (r *EnvironmentVariablesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EnvironmentVariablesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentVariablesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envMap := make(map[string]string)
	diags = plan.Variables.ElementsAs(ctx, &envMap, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetType, targetID, err := getEnvironmentVariableTarget(plan.ApplicationID, plan.ComposeID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Association", err.Error())
		return
	}

	updateFn := func(m map[string]string) {
		for k, v := range envMap {
			m[k] = v
		}
	}

	if targetType == "application" {
		err = r.client.UpdateApplicationEnv(targetID, updateFn, plan.CreateEnvFile.ValueBoolPointer())
	} else {
		err = r.client.UpdateComposeEnv(targetID, updateFn, plan.CreateEnvFile.ValueBoolPointer())
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating environment variables", err.Error())
		return
	}

	plan.ID = types.StringValue(targetID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *EnvironmentVariablesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentVariablesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetType, targetID, err := getEnvironmentVariableTarget(state.ApplicationID, state.ComposeID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Association", err.Error())
		return
	}

	envMap := map[string]string{}
	if targetType == "application" {
		app, appErr := r.client.GetApplication(targetID)
		if appErr != nil {
			if strings.Contains(appErr.Error(), "Not Found") || strings.Contains(appErr.Error(), "404") {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("Error reading application", appErr.Error())
			return
		}
		envMap = client.ParseEnv(app.Env)
	} else {
		comp, compErr := r.client.GetCompose(targetID)
		if compErr != nil {
			if strings.Contains(compErr.Error(), "Not Found") || strings.Contains(compErr.Error(), "404") {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("Error reading compose", compErr.Error())
			return
		}
		envMap = client.ParseEnv(comp.Env)
	}

	state.ID = types.StringValue(targetID)
	state.Variables, diags = types.MapValueFrom(ctx, types.StringType, envMap)
	resp.Diagnostics.Append(diags...)

	// The CreateEnvFile attribute is not stored in the API, so we keep the configured value.
	// If it's not configured, Terraform will use the default.

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *EnvironmentVariablesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentVariablesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envMap := make(map[string]string)
	diags = plan.Variables.ElementsAs(ctx, &envMap, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetType, targetID, err := getEnvironmentVariableTarget(plan.ApplicationID, plan.ComposeID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Association", err.Error())
		return
	}

	updateFn := func(m map[string]string) {
		// Clear existing vars and set new ones
		for k := range m {
			delete(m, k)
		}
		for k, v := range envMap {
			m[k] = v
		}
	}

	if targetType == "application" {
		err = r.client.UpdateApplicationEnv(targetID, updateFn, plan.CreateEnvFile.ValueBoolPointer())
	} else {
		err = r.client.UpdateComposeEnv(targetID, updateFn, plan.CreateEnvFile.ValueBoolPointer())
	}

	if err != nil {
		resp.Diagnostics.AddError("Error updating environment variables", err.Error())
		return
	}

	plan.ID = types.StringValue(targetID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *EnvironmentVariablesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EnvironmentVariablesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetType, targetID, err := getEnvironmentVariableTarget(state.ApplicationID, state.ComposeID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Association", err.Error())
		return
	}

	clearFn := func(m map[string]string) {
		for k := range m {
			delete(m, k)
		}
	}

	if targetType == "application" {
		err = r.client.UpdateApplicationEnv(targetID, clearFn, state.CreateEnvFile.ValueBoolPointer())
	} else {
		err = r.client.UpdateComposeEnv(targetID, clearFn, state.CreateEnvFile.ValueBoolPointer())
	}

	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting environment variables", err.Error())
		return
	}
}

func (r *EnvironmentVariablesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := strings.TrimSpace(req.ID)
	if importID == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID cannot be empty.")
		return
	}

	var applicationID types.String = types.StringNull()
	var composeID types.String = types.StringNull()

	switch {
	case strings.HasPrefix(importID, "application:"):
		id := strings.TrimSpace(strings.TrimPrefix(importID, "application:"))
		if id == "" {
			resp.Diagnostics.AddError("Invalid Import ID", "Expected format application:<id>.")
			return
		}
		applicationID = types.StringValue(id)
	case strings.HasPrefix(importID, "compose:"):
		id := strings.TrimSpace(strings.TrimPrefix(importID, "compose:"))
		if id == "" {
			resp.Diagnostics.AddError("Invalid Import ID", "Expected format compose:<id>.")
			return
		}
		composeID = types.StringValue(id)
	default:
		// Backward compatibility: raw IDs are treated as application IDs.
		applicationID = types.StringValue(importID)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), applicationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("compose_id"), composeID)...)

	if !applicationID.IsNull() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), applicationID)...)
	} else {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), composeID)...)
	}
}

func getEnvironmentVariableTarget(applicationID, composeID types.String) (string, string, error) {
	hasApplicationID := !applicationID.IsNull() && !applicationID.IsUnknown() && applicationID.ValueString() != ""
	hasComposeID := !composeID.IsNull() && !composeID.IsUnknown() && composeID.ValueString() != ""

	if hasApplicationID && hasComposeID {
		return "", "", fmt.Errorf("only one of application_id or compose_id can be provided")
	}
	if !hasApplicationID && !hasComposeID {
		return "", "", fmt.Errorf("either application_id or compose_id must be provided")
	}

	if hasApplicationID {
		return "application", applicationID.ValueString(), nil
	}

	return "compose", composeID.ValueString(), nil
}
