# prom-count
get metrics on samples pushed to remote write endpoint.

### Overview
`prom-count` is a remote write endpoint that emits metrics about the timeseries it receives.

It was originally created to measure churn in active series over time but later expanded to replace some CPU intensive prometheus recording rules in the form of `count({__name__=~".+"} by (...)`.

### Trackers
Currently, it has 2 trackers:
1. the number of active timeseries over a period of time
1. the number of metrics seen with select labels
