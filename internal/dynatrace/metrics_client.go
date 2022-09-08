package dynatrace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/keptn-contrib/dynatrace-service/internal/common"
	"github.com/keptn-contrib/dynatrace-service/internal/sli/metrics"
)

// MetricsPath is the base endpoint for Metrics API v2
const MetricsPath = "/api/v2/metrics"

// MetricsQueryPath is the query endpoint for Metrics API v2
const MetricsQueryPath = MetricsPath + "/query"

// MetricsRequiredDelay is delay required between the end of a timeframe and an Metric V2 API request using it.
const MetricsRequiredDelay = 2 * time.Minute

// MetricsMaximumWait is maximum acceptable wait time between the end of a timeframe and an Metrics V2 API request using it.
const MetricsMaximumWait = 4 * time.Minute

const (
	fromKey           = "from"
	toKey             = "to"
	metricSelectorKey = "metricSelector"
	resolutionKey     = "resolution"
	entitySelectorKey = "entitySelector"
	mzSelectorKey     = "mzSelector"
)

// MetricsClientQueryRequest encapsulates the request for the MetricsClient's GetByQuery method.
type MetricsClientQueryRequest struct {
	query     metrics.Query
	timeframe common.Timeframe
}

// NewMetricsClientQueryRequest creates a new MetricsClientQueryRequest.
func NewMetricsClientQueryRequest(query metrics.Query, timeframe common.Timeframe) MetricsClientQueryRequest {
	return MetricsClientQueryRequest{
		query:     query,
		timeframe: timeframe,
	}
}

// RequestString encodes MetricsClientQueryRequest into a request string.
func (q *MetricsClientQueryRequest) RequestString() string {
	queryParameters := newQueryParameters()
	queryParameters.add(metricSelectorKey, q.query.GetMetricSelector())
	queryParameters.add(fromKey, common.TimestampToUnixMillisecondsString(q.timeframe.Start()))
	queryParameters.add(toKey, common.TimestampToUnixMillisecondsString(q.timeframe.End()))
	queryParameters.add(resolutionKey, q.query.GetResolution())
	if q.query.GetEntitySelector() != "" {
		queryParameters.add(entitySelectorKey, q.query.GetEntitySelector())
	}

	if q.query.GetMZSelector() != "" {
		queryParameters.add(mzSelectorKey, q.query.GetMZSelector())
	}

	return MetricsQueryPath + "?" + queryParameters.encode()
}

// MetricDefinition defines the output of /metrics/<metricID>
type MetricDefinition struct {
	MetricID           string   `json:"metricId"`
	DisplayName        string   `json:"displayName"`
	Description        string   `json:"description"`
	Unit               string   `json:"unit"`
	AggregationTypes   []string `json:"aggregationTypes"`
	Transformations    []string `json:"transformations"`
	DefaultAggregation struct {
		Type string `json:"type"`
	} `json:"defaultAggregation"`
	DimensionDefinitions []DimensionDefinition `json:"dimensionDefinitions"`
	EntityType           []string              `json:"entityType"`
}

type DimensionDefinition struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
}

// MetricData is struct for the response from /api/v2/metrics/query
type MetricData struct {
	Result []MetricSeriesCollection `json:"result"`
}

type MetricSeriesCollection struct {
	MetricID string         `json:"metricId"`
	Data     []MetricSeries `json:"data"`
	Warnings []string       `json:"warnings,omitempty"`
}

type MetricSeries struct {
	Dimensions   []string          `json:"dimensions"`
	DimensionMap map[string]string `json:"dimensionMap,omitempty"`
	Timestamps   []int64           `json:"timestamps"`
	Values       []*float64        `json:"values"`
}

// MetricsQueryFailedError represents an error for a metrics query that could not be retrieved because of an error.
type MetricsQueryFailedError struct {
	cause error
}

// Error returns a string representation of this error.
func (e *MetricsQueryFailedError) Error() string {
	return fmt.Sprintf("error querying Metrics API v2: %v", e.cause)
}

// Unwrap returns the cause of the MetricsQueryFailedError.
func (e *MetricsQueryFailedError) Unwrap() error {
	return e.cause
}

type MetricsQueryProcessingError struct {
	Message  string
	Warnings []string
}

// Error returns a string representation of this error.
func (e *MetricsQueryProcessingError) Error() string {
	if len(e.Warnings) > 0 {
		return fmt.Sprintf("%s. Warnings: %s", e.Message, strings.Join(e.Warnings, ", "))
	}
	return e.Message
}

// MetricsClient is a client for interacting with the Dynatrace problems endpoints
type MetricsClient struct {
	client ClientInterface
}

// NewMetricsClient creates a new MetricsClient
func NewMetricsClient(client ClientInterface) *MetricsClient {
	return &MetricsClient{
		client: client,
	}
}

// GetMetricDefinitionByID calls the Dynatrace API to retrieve MetricDefinition details.
func (mc *MetricsClient) GetMetricDefinitionByID(ctx context.Context, metricID string) (*MetricDefinition, error) {
	body, err := mc.client.Get(ctx, MetricsPath+"/"+metricID)
	if err != nil {
		return nil, err
	}

	var result MetricDefinition
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetSingleMetricSeriesCollectionByQuery executes the request, validates and returns a single metric series collection with at least one metric series or an error.
func (mc *MetricsClient) GetSingleMetricSeriesCollectionByQuery(ctx context.Context, request MetricsClientQueryRequest) (*MetricSeriesCollection, error) {
	metricData, err := mc.getMetricDataByQuery(ctx, request)
	if err != nil {
		return nil, &MetricsQueryFailedError{cause: err}
	}

	if len(metricData.Result) == 0 {
		return nil, &MetricsQueryProcessingError{Message: "Metrics API v2 returned zero metric series collections"}
	}

	if len(metricData.Result) > 1 {
		return nil, &MetricsQueryProcessingError{Message: fmt.Sprintf("Metrics API v2 returned %d metric series collections", len(metricData.Result))}
	}

	metricSeriesCollection := metricData.Result[0]
	if len(metricSeriesCollection.Data) == 0 {
		return nil, &MetricsQueryProcessingError{Message: "Metrics API v2 returned zero metric series", Warnings: metricSeriesCollection.Warnings}
	}

	return &metricSeriesCollection, nil
}

func (mc *MetricsClient) getMetricDataByQuery(ctx context.Context, request MetricsClientQueryRequest) (*MetricData, error) {
	err := NewTimeframeDelay(request.timeframe, MetricsRequiredDelay, MetricsMaximumWait).Wait(ctx)
	if err != nil {
		return nil, err
	}

	body, err := mc.client.Get(ctx, request.RequestString())
	if err != nil {
		return nil, err
	}

	var result MetricData
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
