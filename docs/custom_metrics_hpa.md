### Enabling custom metrics HPA for starboard-exporter

#### Required components

[Prometheus Adapter](https://github.com/kubernetes-sigs/prometheus-adapter)

#### Steps

1. Install prometheus adapter.
    1. Configure helm chart to connect with prometheus to get the metrics
        ```
        prometheus:
          # Value is templated
          url: http://prometheus-operated.gaia-prometheus.svc
          port: 9090
          path: "/gaia"
        ```
    2. Create seriesQuery & metricsQuery accordingly to fetch metrics of the required component. For example:
        ```
         rules:
           custom:
             - seriesQuery: 'scrape_duration_seconds{app="starboard-exporter"}'
               seriesFilters: []
               resources:
                 template: <<.Resource>>
               name:
                 as: "scrapedurationseconds"
               metricsQuery: scrape_duration_seconds{app="starboard-exporter"}
        ```

    #### Make sure prometheus adapter is scarping the metrics & exposing custom metric

    ```bash
    export TEST_NAMESPACE=giantswarm
    kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/$TEST_NAMESPACE/pods/*/scrapedurationseconds" 
    kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/$TEST_NAMESPACE/services/*/scrapedurationseconds" | jq -r .
    ```

2. Enable `customMetricsHpa` from `values.yaml`
