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

var _ resource.Resource = &PortResource{}
var _ resource.ResourceWithImportState = &PortResource{}

func NewPortResource() resource.Resource {
	return &PortResource{}
}

type PortResource struct {
	client *client.DokployClient
}

type PortResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	PublishedPort types.Int64  `tfsdk:"published_port"`
	TargetPort    types.Int64  `tfsdk:"target_port"`
	Protocol      types.String `tfsdk:"protocol"`
	PublishMode   types.String `tfsdk:"publish_mode"`
}

func (r *PortResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port"
}

func (r *PortResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an application port binding in Dokploy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"published_port": schema.Int64Attribute{
				Required: true,
			},
			"target_port": schema.Int64Attribute{
				Required: true,
			},
			"protocol": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Port protocol. Defaults to tcp.",
			},
			"publish_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Port publish mode. Defaults to ingress.",
			},
		},
	}
}

func (r *PortResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Protocol.IsUnknown() || plan.Protocol.IsNull() || strings.TrimSpace(plan.Protocol.ValueString()) == "" {
		plan.Protocol = types.StringValue("tcp")
	}
	if plan.PublishMode.IsUnknown() || plan.PublishMode.IsNull() || strings.TrimSpace(plan.PublishMode.ValueString()) == "" {
		plan.PublishMode = types.StringValue("ingress")
	}

	createdPort, err := r.client.CreatePort(client.Port{
		ApplicationID: plan.ApplicationID.ValueString(),
		PublishedPort: plan.PublishedPort.ValueInt64(),
		TargetPort:    plan.TargetPort.ValueInt64(),
		Protocol:      plan.Protocol.ValueString(),
		PublishMode:   plan.PublishMode.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating port", err.Error())
		return
	}
	if strings.TrimSpace(createdPort.ID) == "" {
		resp.Diagnostics.AddError("Error creating port", "Dokploy did not return a port ID")
		return
	}

	plan.ID = types.StringValue(createdPort.ID)
	if createdPort.ApplicationID != "" {
		plan.ApplicationID = types.StringValue(createdPort.ApplicationID)
	}
	if createdPort.PublishedPort > 0 {
		plan.PublishedPort = types.Int64Value(createdPort.PublishedPort)
	}
	if createdPort.TargetPort > 0 {
		plan.TargetPort = types.Int64Value(createdPort.TargetPort)
	}
	if strings.TrimSpace(createdPort.Protocol) != "" {
		plan.Protocol = types.StringValue(createdPort.Protocol)
	}
	if strings.TrimSpace(createdPort.PublishMode) != "" {
		plan.PublishMode = types.StringValue(createdPort.PublishMode)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *PortResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	port, err := r.client.GetPort(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading port", err.Error())
		return
	}

	state.ID = types.StringValue(port.ID)
	if port.ApplicationID != "" {
		state.ApplicationID = types.StringValue(port.ApplicationID)
	}
	if port.PublishedPort > 0 {
		state.PublishedPort = types.Int64Value(port.PublishedPort)
	}
	if port.TargetPort > 0 {
		state.TargetPort = types.Int64Value(port.TargetPort)
	}
	if strings.TrimSpace(port.Protocol) != "" {
		state.Protocol = types.StringValue(port.Protocol)
	} else if state.Protocol.IsNull() || state.Protocol.IsUnknown() {
		state.Protocol = types.StringValue("tcp")
	}
	if strings.TrimSpace(port.PublishMode) != "" {
		state.PublishMode = types.StringValue(port.PublishMode)
	} else if state.PublishMode.IsNull() || state.PublishMode.IsUnknown() {
		state.PublishMode = types.StringValue("ingress")
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *PortResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Protocol.IsUnknown() || plan.Protocol.IsNull() || strings.TrimSpace(plan.Protocol.ValueString()) == "" {
		plan.Protocol = types.StringValue("tcp")
	}
	if plan.PublishMode.IsUnknown() || plan.PublishMode.IsNull() || strings.TrimSpace(plan.PublishMode.ValueString()) == "" {
		plan.PublishMode = types.StringValue("ingress")
	}

	updatedPort, err := r.client.UpdatePort(client.Port{
		ID:            plan.ID.ValueString(),
		PublishedPort: plan.PublishedPort.ValueInt64(),
		TargetPort:    plan.TargetPort.ValueInt64(),
		Protocol:      plan.Protocol.ValueString(),
		PublishMode:   plan.PublishMode.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating port", err.Error())
		return
	}
	if strings.TrimSpace(updatedPort.ID) == "" {
		resp.Diagnostics.AddError("Error updating port", "Dokploy did not return a port ID")
		return
	}

	if updatedPort.ApplicationID != "" {
		plan.ApplicationID = types.StringValue(updatedPort.ApplicationID)
	}
	if updatedPort.PublishedPort > 0 {
		plan.PublishedPort = types.Int64Value(updatedPort.PublishedPort)
	}
	if updatedPort.TargetPort > 0 {
		plan.TargetPort = types.Int64Value(updatedPort.TargetPort)
	}
	if strings.TrimSpace(updatedPort.Protocol) != "" {
		plan.Protocol = types.StringValue(updatedPort.Protocol)
	}
	if strings.TrimSpace(updatedPort.PublishMode) != "" {
		plan.PublishMode = types.StringValue(updatedPort.PublishMode)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *PortResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePort(state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") || strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting port", err.Error())
		return
	}
}

func (r *PortResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
