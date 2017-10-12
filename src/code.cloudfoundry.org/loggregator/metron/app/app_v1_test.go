package app_test

import (
	"net"
	"sync"

	"code.cloudfoundry.org/loggregator/healthendpoint"
	"code.cloudfoundry.org/loggregator/metricemitter"
	"code.cloudfoundry.org/loggregator/metron/app"
	"code.cloudfoundry.org/loggregator/plumbing"
	"code.cloudfoundry.org/loggregator/testservers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("v1 App", func() {
	It("uses DopplerAddrWithAZ for AZ affinity", func() {
		spyLookup := newSpyLookup()
		gaugeMap := stubGaugeMap()

		promRegistry := prometheus.NewRegistry()
		he := healthendpoint.New(promRegistry, gaugeMap)
		clientCreds, err := plumbing.NewClientCredentials(
			testservers.Cert("metron.crt"),
			testservers.Cert("metron.key"),
			testservers.Cert("loggregator-ca.crt"),
			"doppler",
		)
		Expect(err).ToNot(HaveOccurred())

		config := testservers.BuildMetronConfig("localhost", 1234)
		config.Zone = "something-bad"
		expectedHost, _, err := net.SplitHostPort(config.DopplerAddrWithAZ)
		Expect(err).ToNot(HaveOccurred())

		app := app.NewV1App(
			&config,
			he,
			clientCreds,
			spyMetricClient{},
			app.WithLookup(spyLookup.lookup),
		)
		go app.Start()

		Eventually(spyLookup.calledWith).Should(HaveKey(expectedHost))
	})
})

type spyLookup struct {
	mu          sync.Mutex
	_calledWith map[string]struct{}
}

func newSpyLookup() *spyLookup {
	return &spyLookup{
		_calledWith: make(map[string]struct{}),
	}
}

func (s *spyLookup) calledWith() map[string]struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s._calledWith
}

func (s *spyLookup) lookup(host string) ([]net.IP, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s._calledWith[host] = struct{}{}
	return []net.IP{
		net.IPv4(byte(127), byte(0), byte(0), byte(1)),
	}, nil
}

type spyMetricClient struct {
	app.MetricClient
}

// NewCounter(name string, opts ...metricemitter.MetricOption) *metricemitter.Counter
func (spyMetricClient) NewGauge(name, unit string, opts ...metricemitter.MetricOption) *metricemitter.Gauge {
	return metricemitter.NewGauge(name, unit, "test-source-id")
}

func stubGaugeMap() map[string]prometheus.Gauge {
	return map[string]prometheus.Gauge{
		// metric-documentation-health: (dopplerConnections)
		// Number of connections open to dopplers.
		"dopplerConnections": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "loggregator",
				Subsystem: "metron",
				Name:      "dopplerConnections",
				Help:      "Number of connections open to dopplers",
			},
		),
		// metric-documentation-health: (dopplerV1Streams)
		// Number of V1 gRPC streams to dopplers.
		"dopplerV1Streams": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "loggregator",
				Subsystem: "metron",
				Name:      "dopplerV1Streams",
				Help:      "Number of V1 gRPC streams to dopplers",
			},
		),
		// metric-documentation-health: (dopplerV2Streams)
		// Number of V2 gRPC streams to dopplers.
		"dopplerV2Streams": prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "loggregator",
				Subsystem: "metron",
				Name:      "dopplerV2Streams",
				Help:      "Number of V2 gRPC streams to dopplers",
			},
		),
	}
}
