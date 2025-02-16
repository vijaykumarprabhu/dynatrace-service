# SLI-provider

The dynatrace-service can support the evaluation of the quality gates by retrieving SLIs for a Keptn project, stage or service in response to a `sh.keptn.event.get-sli.triggered` event. Two modes are available: 

- [SLIs via a combination of `dynatrace/sli.yaml` files located on the Keptn service, stage and project](slis-via-files.md), or 
- [SLI and SLOs based on a Dynatrace dashboard](slis-via-dashboard.md).

The mode selected by the dynatrace-service depends on the value of the `dashboard` key in the `dynatrace/dynatrace.conf.yaml` used for a particular event as outlined in [Dashboard SLI-mode configuration (`dashboard`)](dynatrace-conf-yaml-file.md#dashboard-sli-mode-configuration-dashboard)

To help you understand the queries used for obtaining the SLIs, the dynatrace-service includes a custom `query` field in each element of `indicatorValues` in the `sh.keptn.event.get-sli.finished` event. This consists of the path and query string of the associated API request and is viewable directly in the Event payload in the Bridge. 

## Specifying the units of SLIs based on the Metrics v2 API 
The dynatrace-service always returns SLIs in the same units as the underlying metric expression. To convert between units, append a [`:toUnit(<sourceUnit>,<targetUnit>)` transformation](https://www.dynatrace.com/support/help/dynatrace-api/environment-api/metric-v2/metric-selector#to-unit) to the metric expression (e.g. in the **Code** tab of the Data Explorer). For example, `builtin:service.response.time:toUnit(MicroSecond,MilliSecond)` will produce a service response time metric in milliseconds. Alternatively, for file-based SLIs, the [`MV2` prefix](slis-via-files.md#converted-metrics-prefix-mv2) may be used to convert microseconds to milliseconds or bytes to kilobytes in a concise way. For example, `MV2;MicroSecond;metricSelector=builtin:service.response.time:splitBy():avg&entitySelector=type(SERVICE)` will convert the `builtin:service.response.time` metric from microseconds to milliseconds.

## SLI evaluation in auto-remediation workflows

As part of its auto-remediation sequence, Keptn also evaluates SLOs after executing the remediation action. By default, the auto-remediation workflow can be terminated if and only if the problem has been closed in Dynatrace.

To support this, the dynatrace-service will automatically query the status of the problem that originally triggered the workflow using Dynatrace's Problem API v2. It will then append an SLI `problem_open` with the value `0` (=problem no longer open) or `1` (=problem still open). Furthermore, a default key SLO is added with a  pass criteria of `<=0` ensuring that the evaluation will only succeed if the problem is closed:

```yaml
objectives:
- sli: problem_open
  pass:
  - criteria:
    - <=0
  key_sli: true
```

Alternatively, if you'd like to add a custom SLO definition, simply override the default by defining an SLI named `problem_open` together with the appropriate pass and warning annotations.

**Note:** The Dynatrace problem associated with the remediation workflow is tracked via a label containing the Dynatrace Problem URL that is added to each Keptn event in the sequence.


## Known Limitations

- The Dynatrace Metrics API provides data with the "eventual consistency" approach. Therefore, the metrics data retrieved can be incomplete or even contain inconsistencies for timeframes within two hours of the current time. Usually, it takes a minute to catch up, but in extreme situations this might not be enough. The dynatrace-service tries to mitigate this issue by delaying SLI retrieval by up to 120 seconds in situations where the evaluation end time is close to the current time.
