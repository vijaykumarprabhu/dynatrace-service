package monitoring

import (
	"fmt"
	"github.com/keptn-contrib/dynatrace-service/internal/dynatrace"
	"github.com/keptn-contrib/dynatrace-service/internal/lib"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"

	keptnutils "github.com/keptn/go-utils/pkg/api/utils"
)

type Configuration struct {
	client *dynatrace.DynatraceHelper
}

func NewConfiguration(client *dynatrace.DynatraceHelper) *Configuration {
	return &Configuration{
		client: client,
	}
}

// ConfigureMonitoring configures Dynatrace for a Keptn project
func (mc *Configuration) ConfigureMonitoring(project string, shipyard *keptnv2.Shipyard) (*dynatrace.ConfiguredEntities, error) {

	configuredEntities := &dynatrace.ConfiguredEntities{
		TaggingRulesEnabled:         lib.IsTaggingRulesGenerationEnabled(),
		TaggingRules:                NewAutoTagCreation(mc.client).Create(),
		ProblemNotificationsEnabled: lib.IsProblemNotificationsGenerationEnabled(),
		ProblemNotifications:        NewProblemNotificationCreation(mc.client).Create(),
		ManagementZonesEnabled:      lib.IsManagementZonesGenerationEnabled(),
		ManagementZones:             []dynatrace.ConfigResult{},
		DashboardEnabled:            lib.IsDashboardsGenerationEnabled(),
		Dashboard:                   dynatrace.ConfigResult{},
		MetricEventsEnabled:         lib.IsMetricEventsGenerationEnabled(),
		MetricEvents:                []dynatrace.ConfigResult{},
	}

	if project != "" && shipyard != nil {
		configuredEntities.ManagementZones = NewManagementZoneCreation(mc.client).CreateFor(project, *shipyard)
		configuredEntities.Dashboard = NewDashboardCreation(mc.client).CreateFor(project, *shipyard)

		configHandler := keptnutils.NewServiceHandler("shipyard-controller:8080")

		var metricEvents []dynatrace.ConfigResult
		// try to create metric events - if one fails, don't fail the whole setup
		for _, stage := range shipyard.Spec.Stages {
			if shouldCreateMetricEvents(stage) {
				services, err := configHandler.GetAllServices(project, stage.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to retrieve services of project %s: %v", project, err.Error())
				}
				for _, service := range services {
					metricEvents = append(
						metricEvents,
						NewMetricEventCreation(mc.client).CreateFor(project, stage.Name, service.ServiceName)...)
				}
			}
		}
		configuredEntities.MetricEvents = metricEvents
	}
	return configuredEntities, nil
}

// shouldCreateMetricEvents checks if a task sequence with the name 'remediation' is available - this would be the equivalent of remediation_strategy: automated of Keptn < 0.8.x
func shouldCreateMetricEvents(stage keptnv2.Stage) bool {
	for _, taskSequence := range stage.Sequences {
		if taskSequence.Name == "remediation" {
			return true
		}
	}
	return false
}
