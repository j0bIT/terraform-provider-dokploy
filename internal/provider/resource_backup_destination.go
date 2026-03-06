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

var _ resource.Resource = &BackupDestinationResource{}
var _ resource.ResourceWithImportState = &BackupDestinationResource{}

func NewBackupDestinationResource() resource.Resource {
	return &BackupDestinationResource{}
}

type BackupDestinationResource struct {
	client *client.DokployClient
}

type BackupDestinationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	Bucket          types.String `tfsdk:"bucket"`
	Region          types.String `tfsdk:"region"`
	Endpoint        types.String `tfsdk:"endpoint"`
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

func (r *BackupDestinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_destination"
}

func (r *BackupDestinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Dokploy backup destination (for example an S3 bucket).",
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
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Destination type. Defaults to s3.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "Bucket region.",
			},
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "S3 endpoint hostname or URL.",
			},
			"access_key_id": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "S3 access key ID.",
			},
			"secret_access_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "S3 secret access key.",
			},
		},
	}
}

func (r *BackupDestinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BackupDestinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BackupDestinationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Type.IsUnknown() || plan.Type.IsNull() || strings.TrimSpace(plan.Type.ValueString()) == "" {
		plan.Type = types.StringValue("s3")
	}

	created, err := r.client.CreateBackupDestination(client.BackupDestination{
		Name:            plan.Name.ValueString(),
		Type:            plan.Type.ValueString(),
		Bucket:          plan.Bucket.ValueString(),
		Region:          plan.Region.ValueString(),
		Endpoint:        plan.Endpoint.ValueString(),
		AccessKeyID:     plan.AccessKeyID.ValueString(),
		SecretAccessKey: plan.SecretAccessKey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating backup destination", err.Error())
		return
	}

	if strings.TrimSpace(created.ID) == "" {
		resp.Diagnostics.AddError("Error creating backup destination", "Dokploy did not return a destination ID")
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan = applyBackupDestinationState(plan, created)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *BackupDestinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BackupDestinationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destination, err := r.client.GetBackupDestination(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading backup destination", err.Error())
		return
	}

	state = applyBackupDestinationState(state, destination)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *BackupDestinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BackupDestinationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Type.IsUnknown() || plan.Type.IsNull() || strings.TrimSpace(plan.Type.ValueString()) == "" {
		plan.Type = types.StringValue("s3")
	}

	updated, err := r.client.UpdateBackupDestination(client.BackupDestination{
		ID:              plan.ID.ValueString(),
		Name:            plan.Name.ValueString(),
		Type:            plan.Type.ValueString(),
		Bucket:          plan.Bucket.ValueString(),
		Region:          plan.Region.ValueString(),
		Endpoint:        plan.Endpoint.ValueString(),
		AccessKeyID:     plan.AccessKeyID.ValueString(),
		SecretAccessKey: plan.SecretAccessKey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating backup destination", err.Error())
		return
	}

	plan = applyBackupDestinationState(plan, updated)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *BackupDestinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BackupDestinationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBackupDestination(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting backup destination", err.Error())
		return
	}
}

func (r *BackupDestinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyBackupDestinationState(state BackupDestinationResourceModel, destination *client.BackupDestination) BackupDestinationResourceModel {
	if destination == nil {
		return state
	}

	if strings.TrimSpace(destination.ID) != "" {
		state.ID = types.StringValue(destination.ID)
	}
	if strings.TrimSpace(destination.Name) != "" {
		state.Name = types.StringValue(destination.Name)
	}
	providerType := strings.TrimSpace(destination.Type)
	if providerType == "" {
		providerType = strings.TrimSpace(destination.Provider)
	}
	if providerType != "" {
		state.Type = types.StringValue(providerType)
	}
	if strings.TrimSpace(destination.Bucket) != "" {
		state.Bucket = types.StringValue(destination.Bucket)
	}
	if strings.TrimSpace(destination.Region) != "" {
		state.Region = types.StringValue(destination.Region)
	}
	if strings.TrimSpace(destination.Endpoint) != "" {
		state.Endpoint = types.StringValue(destination.Endpoint)
	}
	accessKeyID := strings.TrimSpace(destination.AccessKeyID)
	if accessKeyID == "" {
		accessKeyID = strings.TrimSpace(destination.AccessKey)
	}
	if accessKeyID != "" {
		state.AccessKeyID = types.StringValue(accessKeyID)
	}
	secretAccessKey := strings.TrimSpace(destination.SecretAccessKey)
	if secretAccessKey == "" {
		secretAccessKey = strings.TrimSpace(destination.SecretKey)
	}
	if secretAccessKey != "" {
		state.SecretAccessKey = types.StringValue(secretAccessKey)
	}

	return state
}
