package oci

import (
	"context"
	"strings"

	"github.com/oracle/oci-go-sdk/v44/common"
	"github.com/oracle/oci-go-sdk/v44/database"
	"github.com/turbot/go-kit/types"
	"github.com/turbot/steampipe-plugin-sdk/v2/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin/transform"
)

//// TABLE DEFINITION

func tableOciDatabaseDBHome(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "oci_database_db_home",
		Description: "OCI Database DB Home",
		List: &plugin.ListConfig{
			Hydrate: listDatabaseDBHomes,
			KeyColumns: []*plugin.KeyColumn{
				{
					Name:    "compartment_id",
					Require: plugin.Optional,
				},
				{
					Name:    "db_system_id",
					Require: plugin.Optional,
				},
				{
					Name:    "db_version",
					Require: plugin.Optional,
				},
				{
					Name:    "display_name",
					Require: plugin.Optional,
				},
				{
					Name:    "lifecycle_state",
					Require: plugin.Optional,
				},
				{
					Name:    "vm_cluster_id",
					Require: plugin.Optional,
				},
			},
		},
		Get: &plugin.GetConfig{
			KeyColumns: plugin.SingleColumn("id"),
			Hydrate:    getDatabaseDBHome,
		},
		GetMatrixItem: BuildCompartementRegionList,
		Columns: []*plugin.Column{
			{
				Name:        "display_name",
				Description: "The user-friendly name for the database home. It does not have to be unique.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "id",
				Description: "The OCID of the database home.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromCamel(),
			},
			{
				Name:        "lifecycle_state",
				Description: "The current state of the database home.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "db_system_id",
				Description: "The OCID of the DB system.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromCamel(),
			},
			{
				Name:        "time_created",
				Description: "The date and time the database home was created.",
				Type:        proto.ColumnType_TIMESTAMP,
				Transform:   transform.FromField("TimeCreated.Time"),
			},

			// other columns
			{
				Name:        "database_software_image_id",
				Description: "The database software image OCID.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromCamel(),
			},
			{
				Name:        "db_home_location",
				Description: "The location of the oracle database home.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "db_version",
				Description: "The oracle database version.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "kms_key_id",
				Description: "The OCID of the key container that is used as the master encryption key in database transparent data encryption (TDE) operations.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromCamel(),
			},
			{
				Name:        "last_patch_history_entry_id",
				Description: "The OCID of the last patch history.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromCamel(),
			},
			{
				Name:        "lifecycle_details",
				Description: "Additional information about the current lifecycle state.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "vm_cluster_id",
				Description: "The OCID of the VM cluster.",
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromCamel(),
			},

			// json fields
			{
				Name:        "one_off_patches",
				Description: "List of one-off patches for database homes.",
				Type:        proto.ColumnType_JSON,
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

			// Standard Steampipe columns
			{
				Name:        "tags",
				Description: ColumnDescriptionTags,
				Type:        proto.ColumnType_JSON,
				Transform:   transform.From(dbHomeTags),
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

func listDatabaseDBHomes(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	region := plugin.GetMatrixItem(ctx)[matrixKeyRegion].(string)
	compartment := plugin.GetMatrixItem(ctx)[matrixKeyCompartment].(string)
	logger.Debug("listDatabaseDBHomes", "Compartment", compartment, "OCI_REGION", region)

	equalQuals := d.KeyColumnQuals

	// Return nil, if given compartment_id doesn't match
	if equalQuals["compartment_id"] != nil && compartment != equalQuals["compartment_id"].GetStringValue() {
		return nil, nil
	}

	// Create Session
	session, err := databaseService(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// Build request parameters
	request := buildDatabaseDBHomeFilters(equalQuals)
	request.CompartmentId = types.String(compartment)
	request.Limit = types.Int(1000)
	request.RequestMetadata = common.RequestMetadata{
		RetryPolicy: getDefaultRetryPolicy(d.Connection),
	}

	limit := d.QueryContext.Limit
	if d.QueryContext.Limit != nil {
		if *limit < int64(*request.Limit) {
			request.Limit = types.Int(int(*limit))
		}
	}

	pagesLeft := true
	for pagesLeft {
		response, err := session.DatabaseClient.ListDbHomes(ctx, request)
		if err != nil {
			return nil, err
		}

		for _, dbHome := range response.Items {
			d.StreamListItem(ctx, dbHome)

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

//// HYDRATE FUNCTION

func getDatabaseDBHome(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	region := plugin.GetMatrixItem(ctx)[matrixKeyRegion].(string)
	compartment := plugin.GetMatrixItem(ctx)[matrixKeyCompartment].(string)
	logger.Debug("getDatabaseDBHome", "Compartment", compartment, "OCI_REGION", region)

	// Restrict the api call to only root compartment/ per region
	if !strings.HasPrefix(compartment, "ocid1.tenancy.oc1") {
		return nil, nil
	}

	id := d.KeyColumnQuals["id"].GetStringValue()

	// handle empty id in get call
	if id == "" {
		return nil, nil
	}

	// Create Session
	session, err := databaseService(ctx, d, region)
	if err != nil {
		return nil, err
	}

	request := database.GetDbHomeRequest{
		DbHomeId: types.String(id),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(d.Connection),
		},
	}

	response, err := session.DatabaseClient.GetDbHome(ctx, request)
	if err != nil {
		return nil, err
	}

	return response.DbHome, nil
}

//// TRANSFORM FUNCTION

func dbHomeTags(_ context.Context, d *transform.TransformData) (interface{}, error) {
	freeformTags := dbHomeFreeformTags(d.HydrateItem)

	var tags map[string]interface{}

	if freeformTags != nil {
		tags = map[string]interface{}{}
		for k, v := range freeformTags {
			tags[k] = v
		}
	}

	definedTags := dbHomeDefinedTags(d.HydrateItem)

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

func dbHomeFreeformTags(item interface{}) map[string]string {
	switch item := item.(type) {
	case database.DbHome:
		return item.FreeformTags
	case database.DbHomeSummary:
		return item.FreeformTags
	}
	return nil
}

func dbHomeDefinedTags(item interface{}) map[string]map[string]interface{} {
	switch item := item.(type) {
	case database.DbHome:
		return item.DefinedTags
	case database.DbHomeSummary:
		return item.DefinedTags
	}
	return nil
}

// Build additional filters
func buildDatabaseDBHomeFilters(equalQuals plugin.KeyColumnEqualsQualMap) database.ListDbHomesRequest {
	request := database.ListDbHomesRequest{}

	if equalQuals["db_system_id"] != nil {
		request.DbSystemId = types.String(equalQuals["db_system_id"].GetStringValue())
	}
	if equalQuals["db_version"] != nil {
		request.DbVersion = types.String(equalQuals["db_version"].GetStringValue())
	}
	if equalQuals["display_name"] != nil {
		request.DisplayName = types.String(equalQuals["display_name"].GetStringValue())
	}
	if equalQuals["lifecycle_state"] != nil {
		request.LifecycleState = database.DbHomeSummaryLifecycleStateEnum(equalQuals["lifecycle_state"].GetStringValue())
	}
	if equalQuals["vm_cluster_id"] != nil {
		request.VmClusterId = types.String(equalQuals["vm_cluster_id"].GetStringValue())
	}

	return request
}
