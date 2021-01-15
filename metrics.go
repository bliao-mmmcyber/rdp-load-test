package guac

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"net/http"
	"strconv"
	"time"
)

type GuacServerWrapper struct {
	Server *Server
}

func (s *GuacServerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	start := time.Now()
	writer := &ResponseWriterWrapper{w: w}
	s.Server.ServeHTTP(writer, r)

	path := r.URL.Path
	elapsed := float64(time.Since(start)) / float64(time.Second)
	RecordHttpRequestDur(path, r.Method, elapsed)
	RecordHttpRequest(path, r.Method, writer.status)
}

type ResponseWriterWrapper struct {
	w http.ResponseWriter
	status int
}

func (writer *ResponseWriterWrapper) WriteHeader(statusCode int) {
	writer.status = statusCode
	writer.w.WriteHeader(statusCode)
}

func (writer *ResponseWriterWrapper) Write(b []byte) (int, error) {
	return writer.w.Write(b)
}

func (writer *ResponseWriterWrapper) Header() http.Header {
	return writer.w.Header()
}

var (
	rdpCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rdp_count",
		Help: "The total number of rdp connections",
	})

	requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests",
		},
		[]string{"code", "method", "url"},
	)

	requestsDur = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_requests_dur",
	}, []string{"url", "method"})
)

func init() {
	prometheus.MustRegister(requests)
	prometheus.MustRegister(requestsDur)
}

func WithMetrics(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		writer := &ResponseWriterWrapper{w: w}
		fn(writer, r)
		path := r.URL.Path
		elapsed := float64(time.Since(start)) / float64(time.Second)
		RecordHttpRequestDur(path, r.Method, elapsed)
		RecordHttpRequest(path, r.Method, writer.status)
	}

}

func IncRdpCount() {
	rdpCount.Inc()
}

func DecRdpCount() {
	rdpCount.Dec()
}

func RecordHttpRequest(url, method string, status int) {
	requests.WithLabelValues(strconv.Itoa(status), method, url).Inc()
}

func RecordHttpRequestDur(url, method string, duration float64) {
	requestsDur.WithLabelValues(url, method).Observe(duration)
}
