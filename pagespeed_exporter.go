package main

import (
	"flag"
	"github.com/Teodor5/pagespeed_exporter_without_pwa/collector"
	"github.com/Teodor5/pagespeed_exporter_without_pwa/handler"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"os"
)

var (
	googleApiKey    string
	listenerAddress string
	targets         arrayFlags
	parallel        bool
)

var (
	Version string
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	parseFlags()

	log.Infof("starting pagespeed exporter version %s on address %s for %d targets", Version, listenerAddress, len(targets))

	collectorFactory := collector.NewFactory()
	// Register prometheus target collectors only if there is more than one target
	if len(targets) > 0 {
		requests := collector.CalculateScrapeRequests(targets...)

		psc, errCollector := collectorFactory.Create(collector.Config{
			ScrapeRequests: requests,
			GoogleAPIKey:   googleApiKey,
			Parallel:       parallel,
		})
		if errCollector != nil {
			log.WithError(errCollector).Fatal("could not instantiate collector")
		}
		prometheus.MustRegister(psc)
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler.NewIndexHandler())
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/probe", handler.NewProbeHandler(googleApiKey, parallel, collectorFactory))

	server := http.Server{
		Addr:    listenerAddress,
		Handler: mux,
	}

	log.Fatal(server.ListenAndServe())
}

func parseFlags() {
	flag.StringVar(&googleApiKey, "api-key", getenv("PAGESPEED_API_KEY", ""), "sets the google API key used for pagespeed")
	flag.StringVar(&listenerAddress, "listener", getenv("PAGESPEED_LISTENER", ":9271"), "sets the listener address for the exporters")
	flag.BoolVar(&parallel, "parallel", getenv("PAGESPEED_PARALLEL", "false") == "true", "forces parallel execution for pagespeed")
	targetsFlag := flag.String("targets", getenv("PAGESPEED_TARGETS", ""), "comma separated list of targets to measure")
	flag.Var(&targets, "t", "multiple argument parameters")

	flag.Parse()

	if *targetsFlag != "" {
		additionalTargets := strings.Split(*targetsFlag, ",")
		targets = append(targets, additionalTargets...)
	}

	if len(targets) == 0 || targets[0] == "" {
		log.Info("no targets specified, listening from collector")
	}
}

func getenv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
