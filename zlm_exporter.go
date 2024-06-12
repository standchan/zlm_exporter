package main

import (
	"crypto/tls"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	namespace = "zlmediakit"
)

var (
	serverLabelNames = []string{"backend", "server"}
)

type metricInfo struct {
	Desc *prometheus.Desc
	Type prometheus.ValueType
}

type metrics map[int]metricInfo

func (m metrics) String() string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	s := make([]string, len(keys))
	for i, k := range keys {
		s[i] = strconv.Itoa(k)
	}
	return strings.Join(s, ",")
}

var (
	serverMetrics  = metrics{}
	ZLMediaKitInfo = prometheus.NewDesc(prometheus.BuildFQName(namespace, "version", "info"), "ZLMediaKit version info.", []string{"branchName", "buildTime", "commitHash"}, nil)
)

type Exporter struct {
	client http.Client

	mutex         sync.RWMutex
	up            prometheus.Gauge
	totalScrapes  prometheus.Counter
	serverMetrics map[int]metricInfo
	log           log.Logger
}

func NewExporter(logger *log.Logger) (*Exporter, error) {
	return &Exporter{
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of ZLMediaKit successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrapes_total",
			Help:      "Current total ZLMediaKit scrapes.",
		}),
	}, nil
}

func newServerMetric(metricName string, docString string, t prometheus.ValueType, constLabels prometheus.Labels) metricInfo {
	return metricInfo{
		Desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "server", metricName),
			docString,
			serverLabelNames,
			constLabels,
		),
		Type: t,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.serverMetrics {
		ch <- m.Desc
	}
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	up := e.scrapeHandler(ch)
	ch <- prometheus.MustNewConstMetric(ZLMediaKitInfo, prometheus.GaugeValue, up)
}

func (e *Exporter) scrapeHandler(ch chan<- prometheus.Metric) (up float64) {
	e.totalScrapes.Inc()
	return 1
}

type versionInfo struct {
	BranchName string `json:"branchName"`
	BuildTime  string `json:"buildTime"`
	CommitHash string `json:"commitHash"`
}

func (e *Exporter) extractInfo(ch chan<- prometheus.Metric) {
	header := http.Header{}
	header.Add("secret", "xxxx")
	parsedURL, err := url.Parse("http://127.0.0.1/index/api/version")
	if err != nil {
		// 处理错误
		log.Fatal(err)
	}

	req := &http.Request{
		Method: http.MethodGet,
		URL:    parsedURL,
		Header: header,
	}

	res, err := e.client.Do(req)
	if err != nil {
		e.log.Println(res)
	}
	defer res.Body.Close()

}

func fetchHTTP(uri string, sslVerify, proxyFromEnv bool, timeout time.Duration) func() (io.ReadCloser, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !sslVerify}}
	if proxyFromEnv {
		tr.Proxy = http.ProxyFromEnvironment
	}
	client := http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	return func() (io.ReadCloser, error) {
		resp, err := client.Get(uri)
		if err != nil {
			return nil, err
		}
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
		}
		return resp.Body, nil
	}
}

// doc: https://prometheus.io/docs/instrumenting/writing_exporters/
// 1.metric must use base units
func main() {

}