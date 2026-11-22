package data_readonly

import (
	"context"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func NewSystemSummary() datasource.DataSource {
	return newDataSource("system_summary", "Reads the system name, operating system, architecture, and API version.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetSystemSummaryWithResponse(ctx)
		},
	)
}

func NewCertificateOptions() datasource.DataSource {
	return newDataSource("certificate_options", "Reads the available certificate creation algorithms and key sizes.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetCertOptionsWithResponse(ctx)
		},
	)
}

func NewDirtySubsystems() datasource.DataSource {
	return newDataSource("dirty_subsystems", "Reads which subsystem configurations have pending changes.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetDirtySubsystemsWithResponse(ctx)
		},
	)
}

func NewFirmwareInfo() datasource.DataSource {
	return newDataSource("firmware_info", "Reads the installed firmware version and available upgrade information.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetFirmwareWithResponse(ctx)
		},
	)
}

func NewGatewayPriorities() datasource.DataSource {
	return newDataSource("gateway_priorities", "Reads configured gateways and their priority labels.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetGatewayPrioritiesWithResponse(ctx)
		},
	)
}

func NewGatewayGroups() datasource.DataSource {
	return newDataSource("gateway_groups", "Reads configured gateway groups.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetGatewayGroupsWithResponse(ctx)
		},
	)
}

func NewGatewaysStatus() datasource.DataSource {
	return newSensitiveDataSource("gateways_status", "Reads gateway connectivity and monitor status, including network addresses.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetGatewaysStatusWithResponse(ctx)
		},
	)
}

func NewInterfacesStatus() datasource.DataSource {
	return newSensitiveDataSource("interfaces_status", "Reads configured interface status and network details.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetInterfacesStatusWithResponse(ctx)
		},
	)
}

func NewMonitoringDatasets() datasource.DataSource {
	return newDataSource("monitoring_datasets", "Reads the available system monitoring datasets.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetMonitoringDatasetsWithResponse(ctx)
		},
	)
}

func NewQueueStats() datasource.DataSource {
	return newDataSource("queue_stats", "Reads traffic-shaping queue statistics.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetQueueStatsWithResponse(ctx)
		},
	)
}

func NewRoutesDirtyState() datasource.DataSource {
	return newDataSource("routes_dirty_state", "Reads whether routing configuration has pending changes.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetRoutesDirtyStateWithResponse(ctx)
		},
	)
}

func NewSystemStatus() datasource.DataSource {
	return newSensitiveDataSource("system_status", "Reads system and interface status details, which may include license and controller information.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetStatusWithResponse(ctx)
		},
	)
}

func NewInterfaceStats() datasource.DataSource {
	return newSensitiveDataSource("interface_stats", "Reads network interface traffic statistics and host addresses.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetIfStatsWithResponse(ctx)
		},
	)
}

func NewInterfaceWidgets() datasource.DataSource {
	return newSensitiveDataSource("interface_widgets", "Reads interface status and address details shown by the system dashboard.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetInterfacesWidgetsWithResponse(ctx)
		},
	)
}

func NewInterfaceStatisticsWidget() datasource.DataSource {
	return newDataSource("interface_statistics_widget", "Reads packet, byte, error, and collision counts for configured interfaces.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetInterfaceStatisticsWidgetWithResponse(ctx)
		},
	)
}
