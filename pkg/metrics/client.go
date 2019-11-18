//Provides instruments to read and cache Node Metrics from the custom metrics API.
package metrics

import (
	"errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/metrics/pkg/apis/custom_metrics/v1beta2"
	customclient "k8s.io/metrics/pkg/client/custom_metrics"
	"log"
	"time"
)

//Client knows how to query CustomMetricsAPI to return Node Metrics.
type Client interface {
	GetNodeMetric(metricName string) (NodeMetricsInfo, error)
}

//NodeMetric holds information on a single piece of telemetry data.
type NodeMetric struct {
	Timestamp time.Time
	Window    time.Duration
	Value     resource.Quantity
}

//NodeMetricsInfo holds a map of metric information related to a single named metric. The key for the map is the name of the node.
type NodeMetricsInfo map[string]NodeMetric

//customMetrics client embeds a client for the custom Metrics API
type customMetricsClient struct {
	customclient.CustomMetricsClient
}

//NewClient creates a new Metrics Client including discovering and mapping the available APIs, and pulling the API version.
func NewClient(config *restclient.Config) customMetricsClient {
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(config)
	cachedDiscoveryClient := cacheddiscovery.NewMemCacheClient(discoveryClient)
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()
	apiVersionsGetter := customclient.NewAvailableAPIsGetter(discoveryClient)
	metricsClient := customMetricsClient{customclient.NewForConfig(config, restMapper, apiVersionsGetter)}
	return metricsClient
}

//GetNodeMetric gets the given metric, time Window for Metric and timestamp for each node in the cluster.
func (c customMetricsClient) GetNodeMetric(metricName string) (NodeMetricsInfo, error) {
	log.Print("right her")
	metrics, err := c.RootScopedMetrics().GetForObjects(schema.GroupKind{Kind: "Node"}, labels.NewSelector(), metricName, labels.NewSelector())
	log.Print("right now")
	if err != nil {
		return nil, errors.New("unable to fetch metrics from custom metrics API: " + err.Error())
	}
	if len(metrics.Items) == 0 {
		return nil, errors.New("no metrics returned from custom metrics API")
	}
	output := wrapMetrics(metrics)
	return output, err
}

//wrapMetrics parses the custom metrics API MetricValueList type into a NodeCustomMetricInfo
func wrapMetrics(metrics *v1beta2.MetricValueList) NodeMetricsInfo {
	result := make(NodeMetricsInfo, len(metrics.Items))
	for _, m := range metrics.Items {
		window := time.Minute
		if m.WindowSeconds != nil {
			window = time.Duration(*m.WindowSeconds) * time.Second
		}
		result[m.DescribedObject.Name] = NodeMetric{
			Timestamp: m.Timestamp.Time,
			Window:    window,
			Value:     m.Value,
		}
	}
	return result
}
