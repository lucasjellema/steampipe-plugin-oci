package oci

import (
	"context"
	"encoding/json"
	"strings"

	oci_common "github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"

	"github.com/hashicorp/go-hclog"
	"github.com/turbot/go-kit/types"

	"github.com/turbot/steampipe-plugin-sdk/v4/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v4/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v4/plugin/transform"
)

//// TABLE DEFINITION

func tableContainerInstancesContainerInstance(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:             "oci_container_instances_container_instance",
		Description:      "OCI Container Instances Container Instance",
		DefaultTransform: transform.FromCamel(),
		Get: &plugin.GetConfig{
			KeyColumns: plugin.SingleColumn("id"),
			Hydrate:    getContainerInstanceDetails,
		},
		HydrateDependencies: []plugin.HydrateDependencies{
			{
				Func:    getContainerInstanceContainers,
				Depends: []plugin.HydrateFunc{getContainerInstanceDetails},
			},
		},
		List: &plugin.ListConfig{
			Hydrate: listContainerInstances,
			KeyColumns: []*plugin.KeyColumn{
				{
					Name:    "compartment_id",
					Require: plugin.Optional,
				},
				{
					Name:    "display_name",
					Require: plugin.Optional,
				},
				{
					Name:    "id",
					Require: plugin.Optional,
				},
				{
					Name:    "lifecycle_state",
					Require: plugin.Optional,
				},
			},
		},
		GetMatrixItemFunc: BuildCompartementRegionList,
		Columns: []*plugin.Column{
			{
				Name:        "display_name",
				Description: "The name of the container instance.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("DisplayName"),
			},
			{
				Name:        "id",
				Description: "The OCID of the container instance.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "availability_domain",
				Description: "Availability Domain where the ContainerInstance is running.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "shape",
				Description: "The shape of the Container Instance. The shape determines the resources available to the Container Instance.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "ocpus",
				Description: "The total number of OCPUs available to the instance.",
				Type:        proto.ColumnType_DOUBLE,
			},
			{
				Name:        "memory_in_gbs",
				Description: "The total amount of memory available to the instance, in gigabytes.",
				Type:        proto.ColumnType_DOUBLE,
			},
			{
				Name:        "processor_description",
				Description: "A short description of the instance's processor (CPU).",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "networking_bandwidth_in_gbps",
				Description: "The networking bandwidth available to the instance, in gigabits per second.",
				Type:        proto.ColumnType_DOUBLE,
				Hydrate:     plugin.HydrateFunc(getContainerInstanceDetails),
			},
			{
				Name:        "container_count",
				Description: "The number of containers running on the instance",
				Type:        proto.ColumnType_INT,
			},
			{
				Name:        "containers",
				Description: "Details on all containers in the instance - such as image, restart attempts, working directory.",
				Type:        proto.ColumnType_JSON,
				Transform:   transform.From(transformContainers),
				Hydrate:     plugin.HydrateFunc(getContainerInstanceContainers),
			}, {
				Name:        "container_restart_policy",
				Description: "Container Restart Policy",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "fault_domain",
				Description: "Fault Domain where the ContainerInstance is running.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "lifecycle_state",
				Description: "The current state of the Container Instance.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "time_created",
				Description: "The date and time the Container Instance was created.",
				Type:        proto.ColumnType_TIMESTAMP,
				Transform:   transform.FromField("TimeCreated.Time"),
			},
			{
				Name:        "time_updated",
				Description: "The date and time the Container Instance was last modified.",
				Type:        proto.ColumnType_TIMESTAMP,
				Transform:   transform.FromField("TimeUpdated.Time"),
			},
			{
				Name:        "graceful_shutdown_timeout_in_seconds",
				Description: "The retention period of the messages in the queue, in seconds.",
				Type:        proto.ColumnType_INT,
			},
			{
				Name:        "volume_count",
				Description: "The number of volumes that attached to this Instance.",
				Type:        proto.ColumnType_INT,
			},

			// tags
			{
				Name:        "defined_tags",
				Description: ColumnDescriptionDefinedTags,
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "freeform_tags",
				Description: ColumnDescriptionFreefromTags,
				Type:        proto.ColumnType_JSON,
			},

			// // Standard Steampipe columns
			{
				Name:        "tags",
				Description: ColumnDescriptionTags,
				Type:        proto.ColumnType_JSON,
				Transform:   transform.From(containerInstanceTags),
			},
			{
				Name:        "title",
				Description: ColumnDescriptionTitle,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("DisplayName"),
			},

			// Standard OCI columns
			{
				Name:        "region",
				Description: ColumnDescriptionRegion,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Id").Transform(ociRegionName),
			},
			{
				Name:        "compartment_id",
				Description: ColumnDescriptionCompartment,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("CompartmentId"),
			},
			{
				Name:        "tenant_id",
				Description: ColumnDescriptionTenant,
				Type:        proto.ColumnType_STRING,
				Hydrate:     plugin.HydrateFunc(getTenantId).WithCache(),
				Transform:   transform.FromValue(),
			},
		},
	}
}

//// LIST FUNCTION

func listContainerInstances(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	compartment := plugin.GetMatrixItem(ctx)[matrixKeyCompartment].(string)
	region := plugin.GetMatrixItem(ctx)[matrixKeyRegion].(string)

	logger.Debug("listContainerInstances", "Compartment", compartment, "OCI_REGION", region)

	equalQuals := d.KeyColumnQuals

	// Return nil, if given compartment_id doesn't match
	if equalQuals["compartment_id"] != nil && compartment != equalQuals["compartment_id"].GetStringValue() {
		return nil, nil
	}
	// Create Session
	session, err := containerInstancesService(ctx, d, region)
	if err != nil {
		logger.Error("oci_container_instances_container_instances.listContainerInstances", "connection_error", err)
		return nil, err
	}

	// Build request parameters
	request, isValid := buildContainerInstanceFilters(equalQuals, logger)
	if !isValid {
		return nil, nil
	}
	request.CompartmentId = types.String(compartment)
	request.Limit = types.Int(1000)
	request.RequestMetadata = oci_common.RequestMetadata{
		RetryPolicy: getDefaultRetryPolicy(d.Connection),
	}

	limit := d.QueryContext.Limit
	if limit != nil {
		if *limit < int64(*request.Limit) {
			request.Limit = types.Int(int(*limit))
		}
	}

	pagesLeft := true
	for pagesLeft {
		response, err := session.ContainerInstancesClient.ListContainerInstances(ctx, request)
		if err != nil {
			logger.Error("oci_container_instances_container_instances.listContainerInstances", "api_error", err)
			return nil, err
		}

		for _, containerInstanceSummary := range response.Items {
			d.StreamListItem(ctx, containerInstanceSummary)

			// Context can be cancelled due to manual cancellation or the limit has been hit
			if d.QueryStatus.RowsRemaining(ctx) == 0 {
				return nil, nil
			}
		}
		if response.OpcNextPage != nil {
			request.Page = response.OpcNextPage
		} else {
			pagesLeft = false
		}
	}

	return nil, err
}

//// HYDRATE FUNCTIONS

type ContainerInstanceInfo struct {
	containerinstances.ContainerInstance
	NetworkingBandwidthInGbps float32
	ProcessorDescription string
	Ocpus float32
	MemoryInGBs float32
}

type ContainerDetails struct {
	DisplayName                  string
	ImageUrl                     string
	ExitCode                     int
	WorkingDirectory             string
	ContainerRestartAttemptCount int
	VcpusLimit                   float32
	MemoryLimitInGBs             float32
	TimeTerminated               oci_common.SDKTime
	TimeCreated                  oci_common.SDKTime
	TimeUpdated                  oci_common.SDKTime
}

type ContainerInstanceContainers struct {
	Containers []ContainerDetails
}

func getContainerInstanceContainers(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getContainerInstanceContainers")
	logger := plugin.Logger(ctx)

	// retrieve the ContainerInstanceDetails already produced by Get - to use it for retrieving more container details
	containerInstanceInfo := h.HydrateResults["getContainerInstanceDetails"].(ContainerInstanceInfo)
	containerInstance := containerInstanceInfo.ContainerInstance

	region := plugin.GetMatrixItem(ctx)[matrixKeyRegion].(string)
	compartment := plugin.GetMatrixItem(ctx)[matrixKeyCompartment].(string)
	logger.Debug("getContainerInstanceContainers", "Compartment", compartment, "OCI_REGION", region)
	// Create Session
	session, err := containerInstancesService(ctx, d, region)
	if err != nil {
		logger.Error("oci_container_instances_container_instance.getContainerInstanceContainers", "connection_error", err)
		return nil, err
	}
	containerInstanceContainers := ContainerInstanceContainers{}
	numberOfContainers := len(containerInstance.Containers)
	containerInstanceContainers.Containers = make([]ContainerDetails, numberOfContainers)
	for idx, container := range containerInstance.Containers {
		containerId := *container.ContainerId
		crequest := containerinstances.GetContainerRequest{
			ContainerId: types.String(containerId),
			RequestMetadata: oci_common.RequestMetadata{
				RetryPolicy: getDefaultRetryPolicy(d.Connection),
			},
		}
		cresponse, _ := session.ContainerInstancesClient.GetContainer(ctx, crequest)

		containerDetails := ContainerDetails{}
		containerDetails.DisplayName = *cresponse.DisplayName
		containerDetails.ImageUrl = *cresponse.ImageUrl
		if cresponse.WorkingDirectory != nil {
			containerDetails.WorkingDirectory = *cresponse.WorkingDirectory
		}
		if cresponse.TimeUpdated != nil {
			containerDetails.TimeUpdated = *cresponse.TimeUpdated
		}
		if cresponse.TimeCreated != nil {
			containerDetails.TimeCreated = *cresponse.TimeCreated
		}
		if cresponse.TimeTerminated != nil {
			containerDetails.TimeTerminated = *cresponse.TimeTerminated
		}
		if cresponse.ResourceConfig != nil {
			if cresponse.ResourceConfig.MemoryLimitInGBs != nil {
				containerDetails.MemoryLimitInGBs = *cresponse.ResourceConfig.MemoryLimitInGBs
			}
			if cresponse.ResourceConfig.VcpusLimit != nil {
				containerDetails.VcpusLimit = *cresponse.ResourceConfig.VcpusLimit
			}
		}
		containerDetails.ContainerRestartAttemptCount = *cresponse.ContainerRestartAttemptCount
		containerInstanceContainers.Containers[idx] = containerDetails
	}
	return containerInstanceContainers, nil
}

func getContainerInstanceDetails(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getContainerInstance")
	logger := plugin.Logger(ctx)
	region := plugin.GetMatrixItem(ctx)[matrixKeyRegion].(string)
	compartment := plugin.GetMatrixItem(ctx)[matrixKeyCompartment].(string)
	logger.Debug("getContainerInstance", "Compartment", compartment, "OCI_REGION", region)

	var id string
	if h.Item != nil {
		id = *h.Item.(containerinstances.ContainerInstanceSummary).Id
	} else {
		id = d.KeyColumnQuals["id"].GetStringValue()
		// Restrict the api call to only root compartment/ per region
		if !strings.HasPrefix(compartment, "ocid1.tenancy.oc1") {
			return nil, nil
		}
	}

	if id == "" {
		return nil, nil
	}
	// Create Session
	session, err := containerInstancesService(ctx, d, region)
	if err != nil {
		logger.Error("oci_container_instances_container_instance.getContainerInstance", "connection_error", err)
		return nil, err
	}

	request := containerinstances.GetContainerInstanceRequest{
		ContainerInstanceId: types.String(id),
		RequestMetadata: oci_common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(d.Connection),
		},
	}
	response, err := session.ContainerInstancesClient.GetContainerInstance(ctx, request)
	if err != nil {
		logger.Error("oci_container_instances_container_instance.getContainerInstance", "api_error", err)
		return nil, err
	}
	//	containerInstance := response.ContainerInstance
	containerInstanceInfo := ContainerInstanceInfo{}
	containerInstanceInfo.ContainerInstance = response.ContainerInstance
	
	containerInstanceInfo.NetworkingBandwidthInGbps = *containerInstanceInfo.ContainerInstance.ShapeConfig.NetworkingBandwidthInGbps
	containerInstanceInfo.MemoryInGBs = *containerInstanceInfo.ContainerInstance.ShapeConfig.MemoryInGBs
	containerInstanceInfo.Ocpus = *containerInstanceInfo.ContainerInstance.ShapeConfig.Ocpus
	containerInstanceInfo.ProcessorDescription = *containerInstanceInfo.ContainerInstance.ShapeConfig.ProcessorDescription
	return containerInstanceInfo, nil
}

// Build additional filters
func buildContainerInstanceFilters(equalQuals plugin.KeyColumnEqualsQualMap, logger hclog.Logger) (containerinstances.ListContainerInstancesRequest, bool) {
	request := containerinstances.ListContainerInstancesRequest{}
	isValid := true

	if equalQuals["displayName"] != nil && strings.Trim(equalQuals["displayName"].GetStringValue(), " ") != "" {
		request.DisplayName = types.String(equalQuals["displayName"].GetStringValue())
	}
	return request, isValid
}

// Priority order for tags
// 1. Defined Tags
// 2. Free-form tags
func containerInstanceTags(ctx context.Context, d *transform.TransformData) (interface{}, error) {
	var freeFormTags map[string]string
	var definedTags map[string]map[string]interface{}
	switch d.HydrateItem.(type) {
	case containerinstances.ContainerInstanceSummary:
		freeFormTags = d.HydrateItem.(containerinstances.ContainerInstanceSummary).FreeformTags
		definedTags = d.HydrateItem.(containerinstances.ContainerInstanceSummary).DefinedTags
	case containerinstances.ContainerInstance:
		freeFormTags = d.HydrateItem.(containerinstances.ContainerInstance).FreeformTags
		definedTags = d.HydrateItem.(containerinstances.ContainerInstance).DefinedTags
	default:
		return nil, nil
	}

	var tags map[string]interface{}
	if freeFormTags != nil {
		tags = map[string]interface{}{}
		for k, v := range freeFormTags {
			tags[k] = v
		}
	}

	if definedTags != nil {
		if tags == nil {
			tags = map[string]interface{}{}
		}
		for _, v := range definedTags {
			for key, value := range v {
				tags[key] = value
			}

		}
	}

	return tags, nil
}

// produce a valid JSON representation of the slice of containers
func transformContainers(_ context.Context, d *transform.TransformData) (interface{}, error) {
	containerInstanceContainers := d.HydrateItem.(ContainerInstanceContainers)
	containers := containerInstanceContainers.Containers
	containersJSON, err := json.Marshal(containers)
	if err != nil {
		return nil, err
	}

	return string(containersJSON), nil
}
