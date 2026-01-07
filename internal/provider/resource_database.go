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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/j0bit/terraform-provider-dokploy/internal/client"
)

var _ resource.Resource = &DatabaseResource{}
var _ resource.ResourceWithImportState = &DatabaseResource{}

func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

type DatabaseResource struct {
	client *client.DokployClient
}

type DatabaseResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	EnvironmentID        types.String `tfsdk:"environment_id"`
	Type                 types.String `tfsdk:"type"`
	Name                 types.String `tfsdk:"name"`
	AppName              types.String `tfsdk:"app_name"`
	Description          types.String `tfsdk:"description"`
	DatabaseName         types.String `tfsdk:"database_name"`
	DatabaseUser         types.String `tfsdk:"database_user"`
	DatabasePassword     types.String `tfsdk:"database_password"`
	DatabaseRootPassword types.String `tfsdk:"database_root_password"`
	DockerImage          types.String `tfsdk:"docker_image"`
	ExternalPort         types.Int64  `tfsdk:"external_port"`
	ServerID             types.String `tfsdk:"server_id"`
	ApplicationStatus    types.String `tfsdk:"application_status"`
	ReplicaSets          types.Bool   `tfsdk:"replica_sets"`
	Env                  types.String `tfsdk:"env"`
	MemoryReservation    types.String `tfsdk:"memory_reservation"`
	MemoryLimit          types.String `tfsdk:"memory_limit"`
	CPUReservation       types.String `tfsdk:"cpu_reservation"`
	CPULimit             types.String `tfsdk:"cpu_limit"`
	Command              types.String `tfsdk:"command"`
	Args                 types.List   `tfsdk:"args"`
	Replicas             types.Int64  `tfsdk:"replicas"`
	StopGracePeriod      types.Int64  `tfsdk:"stop_grace_period"`
	RedeployOnUpdate     types.Bool   `tfsdk:"redeploy_on_update"`
}

func (r *DatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *DatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"environment_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"app_name": schema.StringAttribute{
				Computed: true,
				Optional: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"database_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_user": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"database_root_password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"docker_image": schema.StringAttribute{
				Optional: true,
			},
			"external_port": schema.Int64Attribute{
				Optional: true,
			},
			"server_id": schema.StringAttribute{
				Optional: true,
			},
			"application_status": schema.StringAttribute{
				Computed: true,
			},
			"replica_sets": schema.BoolAttribute{
				Optional: true,
			},
			"env": schema.StringAttribute{
				Optional: true,
			},
			"memory_reservation": schema.StringAttribute{
				Optional: true,
			},
			"memory_limit": schema.StringAttribute{
				Optional: true,
			},
			"cpu_reservation": schema.StringAttribute{
				Optional: true,
			},
			"cpu_limit": schema.StringAttribute{
				Optional: true,
			},
			"command": schema.StringAttribute{
				Optional: true,
			},
			"args": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"replicas": schema.Int64Attribute{
				Optional: true,
			},
			"stop_grace_period": schema.Int64Attribute{
				Optional: true,
			},
			"redeploy_on_update": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
	}
}

func (r *DatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatabaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dbType := plan.Type.ValueString()
	appName := plan.Name.ValueString()
	if !plan.AppName.IsNull() && !plan.AppName.IsUnknown() {
		appName = plan.AppName.ValueString()
	}

	var args []string
	if !plan.Args.IsNull() && !plan.Args.IsUnknown() {
		diags = plan.Args.ElementsAs(ctx, &args, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	database, err := r.client.CreateDatabase(
		plan.EnvironmentID.ValueString(),
		dbType,
		appName,
		plan.Name.ValueString(),
		plan.Description.ValueString(),
		plan.DatabaseName.ValueString(),
		plan.DatabaseUser.ValueString(),
		plan.DatabasePassword.ValueString(),
		plan.DatabaseRootPassword.ValueString(),
		plan.DockerImage.ValueString(),
		plan.ServerID.ValueString(),
		plan.ReplicaSets.ValueBool(),
		args,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", err.Error())
		return
	}

	r.setModelFromDatabase(ctx, &plan, database)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatabaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.GetDatabase(state.ID.ValueString(), state.Type.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading database", err.Error())
		return
	}

	r.setModelFromDatabase(ctx, &state, db)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DatabaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DatabaseResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dbType := plan.Type.ValueString()

	var args []string
	diags = plan.Args.ElementsAs(ctx, &args, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.UpdateDatabase(
		plan.ID.ValueString(),
		dbType,
		plan.Name.ValueString(),
		plan.AppName.ValueString(),
		plan.Description.ValueString(),
		plan.DatabaseName.ValueString(),
		plan.DatabaseUser.ValueString(),
		plan.DatabasePassword.ValueString(),
		plan.DatabaseRootPassword.ValueString(),
		plan.DockerImage.ValueString(),
		plan.ServerID.ValueString(),
		plan.ReplicaSets.ValueBool(),
		plan.Env.ValueString(),
		plan.MemoryReservation.ValueString(),
		plan.MemoryLimit.ValueString(),
		plan.CPUReservation.ValueString(),
		plan.CPULimit.ValueString(),
		plan.Command.ValueString(),
		plan.ApplicationStatus.ValueString(),
		plan.Replicas.ValueInt64(),
		plan.StopGracePeriod.ValueInt64(),
		args,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating database", err.Error())
		return
	}

	if plan.RedeployOnUpdate.ValueBool() {
		err := r.client.DeployDatabase(plan.ID.ValueString(), dbType)
		if err != nil {
			resp.Diagnostics.AddWarning("Database updated but redeploy failed", err.Error())
		}
	}

	r.setModelFromDatabase(ctx, &plan, db)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatabaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDatabaseWithType(state.ID.ValueString(), state.Type.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting database", err.Error())
		return
	}
}

func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DatabaseResource) setModelFromDatabase(ctx context.Context, model *DatabaseResourceModel, db *client.Database) {
	model.ID = types.StringValue(db.ID)
	model.Type = types.StringValue(db.Type)
	model.Name = types.StringValue(db.Name)
	model.AppName = types.StringValue(db.AppName)
	model.Description = types.StringValue(db.Description)
	model.DatabaseName = types.StringValue(db.DatabaseName)
	model.DatabaseUser = types.StringValue(db.DatabaseUser)
	model.DockerImage = types.StringValue(db.DockerImage)
	model.ExternalPort = types.Int64Value(db.ExternalPort)
	model.ServerID = types.StringValue(db.ServerID)
	model.ApplicationStatus = types.StringValue(db.ApplicationStatus)
	model.Env = types.StringValue(db.Env)
	model.MemoryReservation = types.StringValue(db.MemoryReservation)
	model.MemoryLimit = types.StringValue(db.MemoryLimit)
	model.CPUReservation = types.StringValue(db.CPUReservation)
	model.CPULimit = types.StringValue(db.CPULimit)
	model.Command = types.StringValue(db.Command)
	model.Replicas = types.Int64Value(db.Replicas)
	model.StopGracePeriod = types.Int64Value(db.StopGracePeriodSwarm)

	if db.Args == nil || len(db.Args) == 0 {
		model.Args = types.ListNull(types.StringType)
	}
}
