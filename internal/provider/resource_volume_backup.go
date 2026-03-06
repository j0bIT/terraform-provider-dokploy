package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/j0bit/terraform-provider-dokploy/internal/client"
)

var _ resource.Resource = &VolumeBackupResource{}
var _ resource.ResourceWithImportState = &VolumeBackupResource{}

func NewVolumeBackupResource() resource.Resource {
	return &VolumeBackupResource{}
}

type VolumeBackupResource struct {
	client *client.DokployClient
}

type VolumeBackupResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	ComposeID       types.String `tfsdk:"compose_id"`
	AppName         types.String `tfsdk:"app_name"`
	ServiceName     types.String `tfsdk:"service_name"`
	VolumeName      types.String `tfsdk:"volume_name"`
	DestinationID   types.String `tfsdk:"destination_id"`
	DestinationName types.String `tfsdk:"destination_name"`
	CronExpression  types.String `tfsdk:"cron_expression"`
	Prefix          types.String `tfsdk:"prefix"`
	KeepLatestCount types.Int64  `tfsdk:"keep_latest_count"`
	Enabled         types.Bool   `tfsdk:"enabled"`
}

func (r *VolumeBackupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup"
}

func (r *VolumeBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dokploy volume backup for a compose service volume.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"compose_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Compose app name used by Dokploy to resolve concrete volume names. If omitted, it is resolved from compose_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_name": schema.StringAttribute{
				Required: true,
			},
			"volume_name": schema.StringAttribute{
				Required: true,
			},
			"destination_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Backup destination ID. If omitted, destination_name is resolved to an ID using destination.all.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"destination_name": schema.StringAttribute{
				Optional:    true,
				Description: "Backup destination name used when destination_id is not provided.",
			},
			"cron_expression": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Cron expression controlling backup schedule. Defaults to \"0 3 * * *\".",
			},
			"prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Prefix used for backup artifact naming.",
			},
			"keep_latest_count": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Description: "Number of most recent backups to keep. Defaults to 14.",
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Description: "Whether the backup schedule is enabled. Defaults to true.",
			},
		},
	}
}

func (r *VolumeBackupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeBackupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VolumeBackupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applyVolumeBackupDefaults(&plan)
	resolvedAppName, err := r.resolveAppName(plan.AppName, plan.ComposeID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid compose backup configuration", err.Error())
		return
	}
	plan.AppName = types.StringValue(resolvedAppName)

	destinationID, err := r.resolveDestinationID(plan.DestinationID, plan.DestinationName)
	if err != nil {
		resp.Diagnostics.AddError("Invalid backup destination configuration", err.Error())
		return
	}

	created, err := r.client.CreateVolumeBackup(client.VolumeBackup{
		Name:            plan.Name.ValueString(),
		ServiceType:     "compose",
		ComposeID:       plan.ComposeID.ValueString(),
		AppName:         plan.AppName.ValueString(),
		ServiceName:     plan.ServiceName.ValueString(),
		VolumeName:      resolveComposeVolumeName(plan.AppName.ValueString(), plan.VolumeName.ValueString()),
		DestinationID:   destinationID,
		CronExpression:  plan.CronExpression.ValueString(),
		Prefix:          plan.Prefix.ValueString(),
		KeepLatestCount: plan.KeepLatestCount.ValueInt64(),
		Enabled:         plan.Enabled.ValueBool(),
		TurnOff:         !plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating volume backup", err.Error())
		return
	}

	if strings.TrimSpace(created.ID) == "" {
		resp.Diagnostics.AddError("Error creating volume backup", "Dokploy did not return a volume backup ID")
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.DestinationID = types.StringValue(destinationID)
	plan = applyVolumeBackupState(plan, created)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *VolumeBackupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VolumeBackupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	backup, err := r.client.GetVolumeBackup(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading volume backup", err.Error())
		return
	}

	state = applyVolumeBackupState(state, backup)
	if state.DestinationName.IsUnknown() {
		state.DestinationName = types.StringNull()
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *VolumeBackupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VolumeBackupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	applyVolumeBackupDefaults(&plan)
	resolvedAppName, err := r.resolveAppName(plan.AppName, plan.ComposeID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid compose backup configuration", err.Error())
		return
	}
	plan.AppName = types.StringValue(resolvedAppName)

	destinationID, err := r.resolveDestinationID(plan.DestinationID, plan.DestinationName)
	if err != nil {
		resp.Diagnostics.AddError("Invalid backup destination configuration", err.Error())
		return
	}

	updated, err := r.client.UpdateVolumeBackup(client.VolumeBackup{
		ID:              plan.ID.ValueString(),
		Name:            plan.Name.ValueString(),
		ServiceType:     "compose",
		ComposeID:       plan.ComposeID.ValueString(),
		AppName:         plan.AppName.ValueString(),
		ServiceName:     plan.ServiceName.ValueString(),
		VolumeName:      resolveComposeVolumeName(plan.AppName.ValueString(), plan.VolumeName.ValueString()),
		DestinationID:   destinationID,
		CronExpression:  plan.CronExpression.ValueString(),
		Prefix:          plan.Prefix.ValueString(),
		KeepLatestCount: plan.KeepLatestCount.ValueInt64(),
		Enabled:         plan.Enabled.ValueBool(),
		TurnOff:         !plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating volume backup", err.Error())
		return
	}

	plan.DestinationID = types.StringValue(destinationID)
	plan = applyVolumeBackupState(plan, updated)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *VolumeBackupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VolumeBackupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteVolumeBackup(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting volume backup", err.Error())
		return
	}
}

func (r *VolumeBackupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyVolumeBackupDefaults(plan *VolumeBackupResourceModel) {
	if plan.CronExpression.IsUnknown() || plan.CronExpression.IsNull() || strings.TrimSpace(plan.CronExpression.ValueString()) == "" {
		plan.CronExpression = types.StringValue("0 3 * * *")
	}
	if plan.KeepLatestCount.IsUnknown() || plan.KeepLatestCount.IsNull() {
		plan.KeepLatestCount = types.Int64Value(14)
	}
	if plan.Enabled.IsUnknown() || plan.Enabled.IsNull() {
		plan.Enabled = types.BoolValue(true)
	}
	if plan.DestinationID.IsUnknown() {
		plan.DestinationID = types.StringNull()
	}
	if plan.DestinationName.IsUnknown() {
		plan.DestinationName = types.StringNull()
	}
	if plan.AppName.IsUnknown() || plan.AppName.IsNull() || strings.TrimSpace(plan.AppName.ValueString()) == "" {
		plan.AppName = types.StringNull()
	}
}

func applyVolumeBackupState(state VolumeBackupResourceModel, backup *client.VolumeBackup) VolumeBackupResourceModel {
	if backup == nil {
		return state
	}

	if strings.TrimSpace(backup.ID) != "" {
		state.ID = types.StringValue(backup.ID)
	}
	if strings.TrimSpace(backup.Name) != "" {
		state.Name = types.StringValue(backup.Name)
	}
	if strings.TrimSpace(backup.ComposeID) != "" {
		state.ComposeID = types.StringValue(backup.ComposeID)
	}
	if strings.TrimSpace(backup.AppName) != "" {
		state.AppName = types.StringValue(backup.AppName)
	}
	if strings.TrimSpace(backup.ServiceName) != "" {
		state.ServiceName = types.StringValue(backup.ServiceName)
	}
	if strings.TrimSpace(backup.VolumeName) != "" {
		state.VolumeName = types.StringValue(stripComposeVolumePrefix(state.AppName.ValueString(), backup.VolumeName))
	}
	if strings.TrimSpace(backup.DestinationID) != "" {
		state.DestinationID = types.StringValue(backup.DestinationID)
	}
	if strings.TrimSpace(backup.CronExpression) != "" {
		state.CronExpression = types.StringValue(backup.CronExpression)
	}
	if strings.TrimSpace(backup.Prefix) != "" {
		state.Prefix = types.StringValue(backup.Prefix)
	}
	if backup.KeepLatestCount > 0 {
		state.KeepLatestCount = types.Int64Value(backup.KeepLatestCount)
	}

	enabled := true
	if !state.Enabled.IsNull() && !state.Enabled.IsUnknown() {
		enabled = state.Enabled.ValueBool()
	}
	if backup.TurnOff {
		enabled = false
	} else if backup.Enabled {
		enabled = true
	}
	state.Enabled = types.BoolValue(enabled)

	return state
}

func (r *VolumeBackupResource) resolveDestinationID(destinationID, destinationName types.String) (string, error) {
	if !destinationID.IsNull() && !destinationID.IsUnknown() && strings.TrimSpace(destinationID.ValueString()) != "" {
		return strings.TrimSpace(destinationID.ValueString()), nil
	}

	if !destinationName.IsNull() && !destinationName.IsUnknown() && strings.TrimSpace(destinationName.ValueString()) != "" {
		destination, err := r.client.FindBackupDestinationByName(destinationName.ValueString())
		if err != nil {
			return "", err
		}
		return destination.ID, nil
	}

	return "", fmt.Errorf("set either destination_id or destination_name")
}

func (r *VolumeBackupResource) resolveAppName(appName, composeID types.String) (string, error) {
	if !appName.IsNull() && !appName.IsUnknown() && strings.TrimSpace(appName.ValueString()) != "" {
		return strings.TrimSpace(appName.ValueString()), nil
	}

	if composeID.IsNull() || composeID.IsUnknown() || strings.TrimSpace(composeID.ValueString()) == "" {
		return "", fmt.Errorf("compose_id is required to resolve app_name")
	}

	comp, err := r.client.GetCompose(strings.TrimSpace(composeID.ValueString()))
	if err != nil {
		return "", fmt.Errorf("failed to resolve compose app_name from compose_id: %w", err)
	}

	if comp != nil {
		if strings.TrimSpace(comp.AppName) != "" {
			return strings.TrimSpace(comp.AppName), nil
		}
		if strings.TrimSpace(comp.Name) != "" {
			return strings.TrimSpace(comp.Name), nil
		}
	}

	return "", fmt.Errorf("compose %s did not return app_name", strings.TrimSpace(composeID.ValueString()))
}

func resolveComposeVolumeName(appName, volumeName string) string {
	appName = strings.TrimSpace(appName)
	volumeName = strings.TrimSpace(volumeName)
	if appName == "" || volumeName == "" {
		return volumeName
	}

	prefix := appName + "_"
	if strings.HasPrefix(volumeName, prefix) {
		return volumeName
	}
	return prefix + volumeName
}

func stripComposeVolumePrefix(appName, volumeName string) string {
	appName = strings.TrimSpace(appName)
	volumeName = strings.TrimSpace(volumeName)
	if appName == "" || volumeName == "" {
		return volumeName
	}

	prefix := appName + "_"
	return strings.TrimPrefix(volumeName, prefix)
}
