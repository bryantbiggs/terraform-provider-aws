// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package eks

import (
	"context"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	awstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_eks_cluster", name="Cluster")
// @Tags(identifierAttribute="arn")
func NewClusterResource(context.Context) (resource.ResourceWithConfigure, error) {
	r := &clusterResource{}

	r.SetDefaultCreateTimeout(30 * time.Minute)
	r.SetDefaultUpdateTimeout(60 * time.Minute)
	r.SetDefaultDeleteTimeout(15 * time.Minute)

	return r, nil
}

type clusterResource struct {
	framework.ResourceWithConfigure
	framework.WithTimeouts
}

func (r *clusterResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "aws_eks_cluster"
}

func (r *clusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			"bootstrap_self_managed_addons": schema.BoolAttribute{
				Default:  booldefault.StaticBool(true),
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrCreatedAt: schema.StringAttribute{
				CustomType: timetypes.RFC3339Type{},
				Computed:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled_cluster_log_types": schema.ListAttribute{
				CustomType: fwtypes.ListOfStringType,
				Optional:   true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrEndpoint: schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrID: framework.IDAttribute(),
			names.AttrName: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^[0-9A-Za-z][0-9A-Za-z_-]*$`), "must start with alphanumeric character and consist of alphanumeric characters, or hyphens"),
					stringvalidator.LengthBetween(1, 100),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"platform_version": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrRoleARN: schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrStatus: schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
			names.AttrVersion: schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"access_config": schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[accessConfigModel](ctx),
				Attributes: map[string]schema.Attribute{
					"authentication_mode": schema.StringAttribute{
						CustomType: fwtypes.StringEnumType[awstypes.AuthenticationMode](),
						Optional:   true,
					},
					"bootstrap_cluster_creator_admin_permissions": schema.BoolAttribute{
						Optional: true,
					},
				},
			},
			"certificate_authority": schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[certificateAuthorityModel](ctx),
				Attributes: map[string]schema.Attribute{
					"data": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"encryption_config": schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[encryptionConfigModel](ctx),
				Attributes: map[string]schema.Attribute{
					names.AttrResources: schema.ListAttribute{
						CustomType: fwtypes.ListOfStringType,
						Required:   true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"provider": schema.SingleNestedBlock{
						CustomType: fwtypes.NewObjectTypeOf[encryptionConfigProviderModel](ctx),
						Attributes: map[string]schema.Attribute{
							"key_arn": schema.StringAttribute{
								Required: true,
							},
						},
					},
				},
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("outpost_config")),
				},
			},
			"identity": schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[identityModel](ctx),
				Blocks: map[string]schema.Block{
					"oidc": schema.SingleNestedBlock{
						CustomType: fwtypes.NewObjectTypeOf[identityOidcModel](ctx),
						Attributes: map[string]schema.Attribute{
							names.AttrIssuer: schema.StringAttribute{
								Computed: true,
							},
						},
					},
				},
			},
			"kubernetes_network_config": schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[kubernetesNetworkConfigModel](ctx),
				Attributes: map[string]schema.Attribute{
					"ip_family": schema.StringAttribute{
						CustomType: fwtypes.StringEnumType[awstypes.IpFamily](),
						Optional:   true,
						Computed:   true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"service_ipv4_cidr": schema.StringAttribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"service_ipv6_cidr": schema.StringAttribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("outpost_config")),
				},
			},
			"outpost_config": schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[outpostConfigModel](ctx),
				Attributes: map[string]schema.Attribute{
					"control_plane_instance_type": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"outpost_arns": schema.ListAttribute{
						CustomType: fwtypes.ListOfStringType,
						Required:   true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"control_plane_placement": schema.SingleNestedBlock{
						CustomType: fwtypes.NewObjectTypeOf[controlPlanePlacementModel](ctx),
						Attributes: map[string]schema.Attribute{
							"group_name": schema.StringAttribute{
								Required: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.RequiresReplace(),
								},
							},
						},
					},
				},
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("encryption_config")),
					objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("kubernetes_network_config")),
				},
			},
			"upgrade_policy": schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[upgradePolicyModel](ctx),
				Attributes: map[string]schema.Attribute{
					"support_type": schema.StringAttribute{
						CustomType: fwtypes.StringEnumType[awstypes.SupportType](),
						Optional:   true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			names.AttrVPCConfig: schema.SingleNestedBlock{
				CustomType: fwtypes.NewObjectTypeOf[vpcConfigModel](ctx),
				Attributes: map[string]schema.Attribute{
					"cluster_security_group_id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"endpoint_private_access": schema.BoolAttribute{
						Optional: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
						Default: booldefault.StaticBool(false),
					},
					"endpoint_public_access": schema.BoolAttribute{
						Optional: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
						Default: booldefault.StaticBool(true),
					},
					"public_access_cidrs": schema.ListAttribute{
						Computed:   true,
						CustomType: fwtypes.ListOfStringType,
						Optional:   true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					names.AttrSecurityGroupIDs: schema.ListAttribute{
						CustomType: fwtypes.ListOfStringType,
						Optional:   true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					names.AttrSubnetIDs: schema.ListAttribute{
						CustomType: fwtypes.ListOfStringType,
						Required:   true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.RequiresReplace(),
						},
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
					},
					names.AttrVPCID: schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
		},
	}

	if s.Blocks == nil {
		s.Blocks = make(map[string]schema.Block)
	}
	s.Blocks["timeouts"] = timeouts.Block(ctx, timeouts.Opts{
		Create: true,
		Update: true,
		Delete: true,
	})

	response.Schema = s
}

func (r *clusterResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data clusterResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().EKSClient(ctx)

	input := &eks.CreateClusterInput{
		BootstrapSelfManagedAddons: flex.ExpandBoolValue(data.BootstrapSelfManagedAddons),
		Name:                       aws.String(data.Name.ValueString()),
		RoleArn:                    aws.String(data.RoleARN.ValueString()),
		ResourcesVpcConfig: &awstypes.VpcConfigRequest{
			SubnetIds: flex.ExpandStringValueSet(data.VPCConfig.SubnetIDs),
		},
	}
	response.Diagnostics.Append(fwflex.Expand(ctx, data, input)...)
	if response.Diagnostics.HasError() {
		return
	}

	// Additional fields.

	output, err := conn.CreateCluster(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("Error creating EKS Cluster", err.Error())
		return
	}

	// Set fields from the output to the state...
	data.ARN = types.StringValue(aws.ToString(output.Cluster.Arn))
	data.Status = types.StringValue(string(output.Cluster.Status))

	diags = response.State.Set(ctx, data)
	response.Diagnostics.Append(diags...)

	createTimeout := r.CreateTimeout(ctx, data.Timeouts)

	data.ID = types.StringValue("TODO")

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *clusterResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data clusterResourceModel
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	output, err := r.client.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(data.Name.ValueString()),
	})
	if err != nil {
		response.Diagnostics.AddError("Error reading EKS Cluster", err.Error())
		return
	}

	// Set fields from the output to the state...
	data.ARN = types.StringValue(aws.ToString(output.Cluster.Arn))
	data.Status = types.StringValue(string(output.Cluster.Status))

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *clusterResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state clusterResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	// Implement update logic here...

	response.Diagnostics.Append(request.Plan.Get(ctx, &new)...)

	if response.Diagnostics.HasError() {
		return
	}
	updateTimeout := r.UpdateTimeout(ctx, new.Timeouts)

	response.Diagnostics.Append(response.State.Set(ctx, &new)...)
}

func (r *clusterResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state clusterResourceModel
	diags := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: aws.String(state.Name.ValueString()),
	})
	if err != nil {
		response.Diagnostics.AddError("Error deleting EKS Cluster", err.Error())
		return
	}
}

func (r *clusterResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(names.AttrName), request, response)
}

func (r *clusterResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	r.SetTagsAll(ctx, request, response)
}

type clusterResourceModel struct {
	AccessConfig               accessConfigModel                                                `tfsdk:"access_config"`
	ARN                        fwtypes.ARN                                                      `tfsdk:"arn"`
	BootstrapSelfManagedAddons fwtypes.ListNestedObjectValueOf[bootstrapSelfManagedAddonsModel] `tfsdk:"bootstrap_self_managed_addons"`
	CertificateAuthority       certificateAuthorityModel                                        `tfsdk:"certificate_authority"`
	ClusterId                  types.String                                                     `tfsdk:"cluster_id"`
	CreatedAt                  timetypes.RFC3339                                                `tfsdk:"created_at"`
	EnabledClusterLogTypes     fwtypes.ListValueOf[fwtypes.StringEnum[awstypes.LogType]]        `tfsdk:"enabled_cluster_log_types"`
	EncryptionConfig           encryptionConfigModel                                            `tfsdk:"encryption_config"`
	Endpoint                   types.String                                                     `tfsdk:"endpoint"`
	Identity                   identityModel                                                    `tfsdk:"identity"`
	KubernetesNetworkConfig    kubernetesNetworkConfigModel                                     `tfsdk:"kubernetes_network_config"`
	Name                       types.String                                                     `tfsdk:"name"`
	OutpostConfig              outpostConfigModel                                               `tfsdk:"outpost_config"`
	PlatformVersion            types.String                                                     `tfsdk:"platform_version"`
	RoleARN                    fwtypes.ARN                                                      `tfsdk:"role_arn"`
	Status                     fwtypes.StringEnum[awstypes.ClusterStatus]                       `tfsdk:"status"`
	Tags                       types.Map                                                        `tfsdk:"tags"`
	TagsAll                    types.Map                                                        `tfsdk:"tags_all"`
	Timeouts                   timeouts.Value                                                   `tfsdk:"timeouts"`
	UpgradePolicy              fwtypes.ListNestedObjectValueOf[upgradePolicyModel]              `tfsdk:"upgrade_policy"`
	Version                    types.String                                                     `tfsdk:"version"`
	VPCConfig                  vpcConfigModel                                                   `tfsdk:"vpc_config"`
}

type accessConfigModel struct {
	AuthenticationMode                      fwtypes.StringEnum[awstypes.AuthenticationMode] `tfsdk:"authentication_mode"`
	BootstrapClusterCreatorAdminPermissions types.Bool                                      `tfsdk:"bootstrap_cluster_creator_admin_permissions"`
}

type bootstrapSelfManagedAddonsModel struct {
	Data types.String `tfsdk:"data"`
}

type certificateAuthorityModel struct {
	Data types.String `tfsdk:"data"`
}

type encryptionConfigModel struct {
	Provider  encryptionConfigProviderModel     `tfsdk:"provider"`
	Resources fwtypes.ListValueOf[types.String] `tfsdk:"resources"`
}

type encryptionConfigProviderModel struct {
	KeyARN types.String `tfsdk:"key_arn"`
}

type identityModel struct {
	Oidc identityOidcModel `tfsdk:"oidc"`
}

type identityOidcModel struct {
	Issuer types.String `tfsdk:"issuer"`
}

type kubernetesNetworkConfigModel struct {
	IpFamily        fwtypes.StringEnum[awstypes.IpFamily] `tfsdk:"ip_family"`
	ServiceIpv4Cidr types.String                          `tfsdk:"service_ipv4_cidr"`
	ServiceIpv6Cidr types.String                          `tfsdk:"service_ipv6_cidr"`
}

type outpostConfigModel struct {
	ControlPlaneInstanceType types.String                      `tfsdk:"control_plane_instance_type"`
	ControlPlanePlacement    controlPlanePlacementModel        `tfsdk:"control_plane_placement"`
	OutpostArns              fwtypes.ListValueOf[types.String] `tfsdk:"outpost_arns"`
}

type controlPlanePlacementModel struct {
	GroupName types.String `tfsdk:"group_name"`
}

type upgradePolicyModel struct {
	SupportType fwtypes.StringEnum[awstypes.SupportType] `tfsdk:"support_type"`
}

type vpcConfigModel struct {
	ClusterSecurityGroupID types.String                           `tfsdk:"cluster_security_group_id"`
	EndpointPrivateAccess  types.Bool                             `tfsdk:"endpoint_private_access"`
	EndpointPublicAccess   types.Bool                             `tfsdk:"endpoint_public_access"`
	PublicAccessCidrs      fwtypes.ListValueOf[fwtypes.CIDRBlock] `tfsdk:"public_access_cidrs"`
	SecurityGroupIDs       fwtypes.ListValueOf[types.String]      `tfsdk:"security_group_ids"`
	SubnetIDs              fwtypes.ListValueOf[types.String]      `tfsdk:"subnet_ids"`
	VpcID                  types.String                           `tfsdk:"vpc_id"`
}
