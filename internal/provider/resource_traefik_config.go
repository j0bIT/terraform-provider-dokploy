package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/j0bit/terraform-provider-dokploy/internal/client"
)

var _ resource.Resource = &TraefikConfigResource{}
var _ resource.ResourceWithImportState = &TraefikConfigResource{}

func NewTraefikConfigResource() resource.Resource {
	return &TraefikConfigResource{}
}

type TraefikConfigResource struct {
	client *client.DokployClient
}

type TraefikConfigResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Scope         types.String `tfsdk:"scope"`
	ServerID      types.String `tfsdk:"server_id"`
	Config        types.String `tfsdk:"config"`
	ReloadOnApply types.Bool   `tfsdk:"reload_on_apply"`
}

func (r *TraefikConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traefik_config"
}

func (r *TraefikConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Dokploy global Traefik configuration via settings.read/update/reloadTraefikConfig endpoints.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
				Description: "Traefik settings scope. Supported values: main, web_server, middleware.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"server_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Optional Dokploy server ID for multi-server deployments.",
			},
			"config": schema.StringAttribute{
				Required:    true,
				Description: "Full Traefik configuration content to persist in Dokploy settings.",
			},
			"reload_on_apply": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "If true, triggers settings.reloadTraefik after create/update.",
			},
		},
	}
}

func (r *TraefikConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TraefikConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TraefikConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scope, err := normalizeTraefikConfigScopeForResource(plan.Scope)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope", err.Error())
		return
	}

	serverID := optionalStringPointer(plan.ServerID)
	if err := r.client.UpdateScopedTraefikConfig(scope, serverID, plan.Config.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error creating Traefik config", err.Error())
		return
	}

	if plan.ReloadOnApply.ValueBool() {
		if err := r.client.ReloadTraefik(serverID); err != nil {
			resp.Diagnostics.AddError("Error reloading Traefik", err.Error())
			return
		}
	}

	plan.Scope = types.StringValue(scope)
	plan.ID = types.StringValue(traefikConfigStateID(scope, plan.ServerID))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TraefikConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TraefikConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scope, err := normalizeTraefikConfigScopeForResource(state.Scope)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope", err.Error())
		return
	}

	config, err := r.client.ReadScopedTraefikConfig(scope, optionalStringPointer(state.ServerID))
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Traefik config", err.Error())
		return
	}

	state.Scope = types.StringValue(scope)
	state.ID = types.StringValue(traefikConfigStateID(scope, state.ServerID))
	state.Config = types.StringValue(config)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *TraefikConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TraefikConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scope, err := normalizeTraefikConfigScopeForResource(plan.Scope)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope", err.Error())
		return
	}

	serverID := optionalStringPointer(plan.ServerID)
	if err := r.client.UpdateScopedTraefikConfig(scope, serverID, plan.Config.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating Traefik config", err.Error())
		return
	}

	if plan.ReloadOnApply.ValueBool() {
		if err := r.client.ReloadTraefik(serverID); err != nil {
			resp.Diagnostics.AddError("Error reloading Traefik", err.Error())
			return
		}
	}

	plan.Scope = types.StringValue(scope)
	plan.ID = types.StringValue(traefikConfigStateID(scope, plan.ServerID))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TraefikConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Intentionally no-op: deleting this Terraform resource should not wipe Traefik config remotely.
}

func (r *TraefikConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID := strings.TrimSpace(req.ID)
	if importID == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID cannot be empty.")
		return
	}

	switch {
	case importID == "default", importID == "main:default":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "default")...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope"), "main")...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), types.StringNull())...)
	case strings.HasPrefix(importID, "server:"), strings.HasPrefix(importID, "main:server:"):
		serverID := strings.TrimSpace(strings.TrimPrefix(importID, "server:"))
		if strings.HasPrefix(importID, "main:server:") {
			serverID = strings.TrimSpace(strings.TrimPrefix(importID, "main:server:"))
		}
		if serverID == "" {
			resp.Diagnostics.AddError("Invalid Import ID", "Expected format server:<id>.")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "server:"+serverID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope"), "main")...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	case strings.HasPrefix(importID, "web_server:"), strings.HasPrefix(importID, "middleware:"):
		scope, serverID, parseErr := parseScopedTraefikConfigImportID(importID)
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid Import ID", parseErr.Error())
			return
		}
		normalizedScope, err := normalizeTraefikConfigScopeForResource(types.StringValue(scope))
		if err != nil {
			resp.Diagnostics.AddError("Invalid Import ID", err.Error())
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), buildTraefikConfigStateID(normalizedScope, serverID))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope"), normalizedScope)...)
		if serverID == nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), types.StringNull())...)
		} else {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), *serverID)...)
		}
	default:
		resp.Diagnostics.AddError("Invalid Import ID", "Use 'default', 'server:<id>', '<scope>:default', or '<scope>:server:<id>' where scope is web_server|middleware.")
	}
}

func traefikConfigStateID(scope string, serverID types.String) string {
	serverIDValue := ""
	if !serverID.IsNull() && !serverID.IsUnknown() {
		serverIDValue = strings.TrimSpace(serverID.ValueString())
	}
	if serverIDValue == "" {
		return buildTraefikConfigStateID(scope, nil)
	}
	return buildTraefikConfigStateID(scope, &serverIDValue)
}

func buildTraefikConfigStateID(scope string, serverID *string) string {
	if scope == "main" {
		if serverID == nil || strings.TrimSpace(*serverID) == "" {
			return "default"
		}
		return "server:" + strings.TrimSpace(*serverID)
	}

	if serverID == nil || strings.TrimSpace(*serverID) == "" {
		return scope + ":default"
	}

	return scope + ":server:" + strings.TrimSpace(*serverID)
}

func optionalStringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	trimmed := strings.TrimSpace(value.ValueString())
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func normalizeTraefikConfigScopeForResource(scope types.String) (string, error) {
	if scope.IsNull() || scope.IsUnknown() {
		return "main", nil
	}
	switch strings.TrimSpace(strings.ToLower(scope.ValueString())) {
	case "", "main":
		return "main", nil
	case "web_server", "webserver", "web-server":
		return "web_server", nil
	case "middleware":
		return "middleware", nil
	default:
		return "", fmt.Errorf("unsupported scope %q (supported: main, web_server, middleware)", scope.ValueString())
	}
}

func parseScopedTraefikConfigImportID(importID string) (string, *string, error) {
	if strings.HasSuffix(importID, ":default") {
		scope := strings.TrimSuffix(importID, ":default")
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return "", nil, fmt.Errorf("invalid import ID %q", importID)
		}
		return scope, nil, nil
	}

	if strings.Contains(importID, ":server:") {
		parts := strings.SplitN(importID, ":server:", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid import ID %q", importID)
		}
		scope := strings.TrimSpace(parts[0])
		serverID := strings.TrimSpace(parts[1])
		if scope == "" || serverID == "" {
			return "", nil, fmt.Errorf("invalid import ID %q", importID)
		}
		return scope, &serverID, nil
	}

	return "", nil, fmt.Errorf("invalid import ID %q", importID)
}
