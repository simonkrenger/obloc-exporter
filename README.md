# O'Bloc Prometheus Exporter

Small Golang program to export the current [O'Bloc](https://obloc.ch) utilization for Prometheus.

~~~
$ podman run --rm -d -p 8081:8081 quay.io/simonkrenger/obloc-exporter:latest
$ curl localhost:8081/metrics
...
# HELP obloc_utilization_percent The current O'Bloc utilization
# TYPE obloc_utilization_percent gauge
obloc_utilization_percent 56
~~~

## Why?

[O'Bloc](https://obloc.ch/) is a climbing gym in Bern, Switzerland and they measure how many people are in the gym at a given time. They publish this information on their website.

This project exports this information as a Prometheus Exporter so it can be queried by Prometheus.
