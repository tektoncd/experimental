package metrics

import (
	"net/http"
	"strconv"

	prom "contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
)

type MetricConfig struct {
	Namespace string

	// default "0.0.0.0"
	PrometheusHost string

	PrometheusPort int
}

type PrometheusServer struct {
	server   *http.Server
	exporter view.Exporter
}

func (p *PrometheusServer) GetExporter() view.Exporter { return p.exporter }

func (p *PrometheusServer) Start() {
	p.server.ListenAndServe()
}

func (p *PrometheusServer) Restart() {
	p.server.Close()
	p.server.ListenAndServe()
}

func (p *PrometheusServer) Stop() {
	p.server.Close()
}

func NewPrometheusExporter(config *MetricConfig) (*PrometheusServer, error) {
	e, err := prom.NewExporter(prom.Options{Namespace: config.Namespace})
	if err != nil {
		return nil, err
	}
	sm := http.NewServeMux()
	sm.Handle("/metrics", e)
	server := &http.Server{
		Addr:    config.PrometheusHost + ":" + strconv.Itoa(config.PrometheusPort),
		Handler: sm,
	}
	return &PrometheusServer{
		server:   server,
		exporter: e,
	}, nil
}
