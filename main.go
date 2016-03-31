package main

import (
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

var (
	addr             = flag.String("web.listen-address", ":9100", "Address on witch to expose metrics and web address.")
	metricPath       = flag.String("web.telemetry-path", "/metrics", "Path under which to expose Prometheus metrics.")
	twitterServerUrl = flag.String("twitterserver.url", "", "URL to Twitter Server metric.json")
)

var namespace = "twitterserver"
var validStatNames = []string{"count", "sum", "avg", "min", "max", "stddev",
	"p50", "p90", "p95", "p99", "p9990", "p9999"}
var bucketLabels = []string{"bucket"}
var noLables = []string{}

var httpClient = http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost:   2,
		ResponseHeaderTimeout: 10 * time.Second,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
		}).Dial,
	},
}

var rp = regexp.MustCompile("[^a-zA-Z0-9_:]")

type exporter struct {
	sync.Mutex
	errors   prometheus.Counter
	duration prometheus.Gauge
}

func newTwitterExporter() *exporter {
	return &exporter{
		errors: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "exporter_scrape_errora_total",
				Help:      "Total scrape errors",
			}),
		duration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "exporter_last_scrape_duration_seconds",
				Help:      "The last scape duration",
			}),
	}
}

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.duration.Desc()
	ch <- e.errors.Desc()
}

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.Lock()
	defer e.Unlock()

	metricsChan := make(chan prometheus.Metric)
	go e.scrape(metricsChan)

	for metric := range metricsChan {
		ch <- metric
	}

	ch <- e.errors
	ch <- e.duration
}

func parseMetric(name string, value float64) (metric prometheus.Metric) {
	parsedKey := rp.ReplaceAllString(name, "_")
	metric = prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", parsedKey), "",
			noLables, nil,
		),
		prometheus.GaugeValue,
		value,
	)
	for _, stat := range validStatNames {
		if strings.HasSuffix(parsedKey, stat) {
			parsedKey = strings.Replace(parsedKey, "_"+stat, "", -1)
			metric = prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "", parsedKey), "",
					bucketLabels, nil,
				),
				prometheus.GaugeValue,
				value, stat,
			)
			break
		}
	}
	return metric
}

func (e *exporter) scrape(ch chan<- prometheus.Metric) {
	defer close(ch)

	now := time.Now().UnixNano()
	defer func() {
		e.duration.Set(float64(time.Now().UnixNano()-now) / 1000000000)
	}()

	recordErr := func(err error) {
		log.Warn(err)
		e.errors.Inc()
	}

	resp, err := httpClient.Get(*twitterServerUrl)
	if err != nil {
		recordErr(err)
		return
	}
	defer resp.Body.Close()

	var vars map[string]interface{}

	if err = json.NewDecoder(resp.Body).Decode(&vars); err != nil {
		recordErr(err)
		return
	}

	for name, v := range vars {
		v, ok := v.(float64)
		if !ok {
			continue
		}
		ch <- parseMetric(name, v)
	}
}

func main() {
	flag.Parse()

	exporter := newTwitterExporter()
	prometheus.MustRegister(exporter)

	http.Handle(*metricPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricPath, http.StatusMovedPermanently)
	})

	log.Info("starting twitterserver_exporter on ", *addr)

	log.Fatal(http.ListenAndServe(*addr, nil))
}
