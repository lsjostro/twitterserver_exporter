# Prometheus TwitterServer exporter

This is an exporter for Prometheus to get instrumentation data for [TwitterServer](https://twitter.github.io/twitter-server/index.html)

## Build and run

    make
    ./twitterserver_exporter <flags>

### Flags

Name                           | Description
-------------------------------|------------
web.listen-address             | Address to listen on for web interface and telemetry.
web.telemetry-path             | Path under which to expose metrics.
twitterserver.url              | [URL](#twitterserver-url) to metrics.json

#### TwitterServer URL

URL to metrics.json ``http://host:port/admin/metrics.json``.
