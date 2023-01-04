### Enabling custom metrics HPA for starboard-exporter

#### Required components

[Prometheus Adapter](https://github.com/kubernetes-sigs/prometheus-adapter)

#### Steps

1. Install prometheus adapter.
    1. Configure helm chart to connect with prometheus to get the metrics. Change the values as required.
        ```
        prometheus:
          # Value is templated
          url: http://prometheus-operated.monitoring.svc
          port: 9090
          path: ""
        ```
    2. Create seriesQuery & metricsQuery to fetch metrics for the required components. For example, labels like `app="starboard-exporter"` or `job="starboard-exporter"` can be used to filter the `starboard-exporter` metrics in prometheus. Change the values as required.
        ```
         rules:
           custom:
             - seriesQuery: 'scrape_duration_seconds{job="starboard-exporter"}'
               seriesFilters: []
               resources:
                 template: <<.Resource>>
               name:
                 as: "scrapedurationseconds"
               metricsQuery: scrape_duration_seconds{job="starboard-exporter"}
        ```

    #### Make sure prometheus adapter is scraping the metrics & exposing the custom metric

    ```bash
    export TEST_NAMESPACE=giantswarm
    kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/$TEST_NAMESPACE/pods/*/scrapedurationseconds" 
    kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/$TEST_NAMESPACE/services/*/scrapedurationseconds" | jq -r .
    ```

2. Enable `customMetricsHPA` from `values.yaml`
