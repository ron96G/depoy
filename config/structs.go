package config

import (
	"github.com/creasty/defaults"
	"github.com/google/uuid"
	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/metrics"
	"github.com/rgumi/depoy/route"
	"github.com/rgumi/depoy/storage"
	"github.com/rgumi/depoy/util"
	log "github.com/sirupsen/logrus"
)

type InputGateway struct {
	Addr         string              `yaml:"addr" json:"addr" default:":8080"`
	ReadTimeout  util.ConfigDuration `yaml:"read_timeout" json:"readTimeout" default:"\"5s\""`
	WriteTimeout util.ConfigDuration `yaml:"write_timeout" json:"writeTimeout" default:"\"5s\""`
	HTTPTimeout  util.ConfigDuration `yaml:"http_timeout" json:"httpTimeout" default:"\"5s\""`
	IdleTimeout  util.ConfigDuration `yaml:"idle_timeout" json:"idleTimeout" default:"\"5s\""`
	Routes       []*InputRoute       `yaml:"routes" json:"routes"`
}

type InputRoute struct {
	Name                string              `json:"name" yaml:"name" validate:"empty=false"`
	Prefix              string              `json:"prefix" yaml:"prefix" validate:"empty=false"`
	Methods             []string            `json:"methods" yaml:"methods" default:"[\"GET\", \"POST\", \"PUT\", \"DELETE\", \"PATCH\", \"HEAD\", \"OPTIONS\", \"TRACE\", \"TRACE\"]"`
	Host                string              `json:"host" yaml:"host" default:"*"`
	Rewrite             string              `json:"rewrite" yaml:"rewrite" validate:"empty=false"`
	CookieTTL           util.ConfigDuration `json:"cookie_ttl" yaml:"cookieTTL" default:"\"5m\""`
	Strategy            *route.Strategy     `json:"strategy" yaml:"strategy" validate:"nil=false"`
	HealthCheck         bool                `json:"healthcheck_bool" yaml:"healthcheckBool" default:"true"`
	HealthCheckInterval util.ConfigDuration `json:"healthcheck_interval" yaml:"healthcheckInterval" default:"\"5s\""`
	MonitoringInterval  util.ConfigDuration `json:"monitoring_interval" yaml:"monitoringInterval" default:"\"5s\""`
	Timeout             util.ConfigDuration `json:"timeout" yaml:"timeout" default:"\"5s\""`
	IdleTimeout         util.ConfigDuration `json:"idle_timeout" yaml:"idleTimeout" default:"\"5s\""`
	ScrapeInterval      util.ConfigDuration `json:"scrape_interval" yaml:"scrapeInterval" default:"\"5s\""`
	Proxy               string              `json:"proxy" yaml:"proxy" default:""`
	Backends            []*route.Backend    `json:"backends" yaml:"backends"`
}

func NewInputRoute() *InputRoute {
	route := new(InputRoute)
	defaults.Set(route)
	return route
}

func NewInputeGateway() *InputGateway {
	g := new(InputGateway)
	defaults.Set(g)
	return g
}

func ConvertRouteToInputRoute(r *route.Route) *InputRoute {
	inputRoute := &InputRoute{
		Name:                r.Name,
		Prefix:              r.Prefix,
		Rewrite:             r.Rewrite,
		Strategy:            r.Strategy,
		Proxy:               r.Proxy,
		Timeout:             util.ConfigDuration{r.Timeout},
		ScrapeInterval:      util.ConfigDuration{r.ScrapeInterval},
		Backends:            []*route.Backend{},
		CookieTTL:           util.ConfigDuration{r.CookieTTL},
		HealthCheck:         r.HealthCheck,
		HealthCheckInterval: util.ConfigDuration{r.HealthCheckInterval},
		MonitoringInterval:  util.ConfigDuration{r.MonitoringInterval},
		Host:                r.Host,
		IdleTimeout:         util.ConfigDuration{r.IdleTimeout},
		Methods:             r.Methods,
	}
	inputRoute.Backends = make([]*route.Backend, len(r.Backends))
	i := 0
	for _, backend := range r.Backends {
		inputRoute.Backends[i] = backend
		i++
	}
	return inputRoute
}

func ConvertGatewayToInputGateway(g *gateway.Gateway) *InputGateway {
	inputGateway := &InputGateway{
		Addr:         g.Addr,
		ReadTimeout:  util.ConfigDuration{g.ReadTimeout},
		WriteTimeout: util.ConfigDuration{g.WriteTimeout},
		HTTPTimeout:  util.ConfigDuration{g.HTTPTimeout},
		IdleTimeout:  util.ConfigDuration{g.IdleTimeout},
		Routes:       []*InputRoute{},
	}
	inputGateway.Routes = make([]*InputRoute, len(g.Routes))
	i := 0
	for _, r := range g.Routes {
		inputGateway.Routes[i] = ConvertRouteToInputRoute(r)
		i++
	}
	return inputGateway
}

func ConvertInputRouteToRoute(r *InputRoute) (*route.Route, error) {
	newRoute, err := route.New(
		r.Name,
		r.Prefix,
		r.Rewrite,
		r.Host,
		r.Proxy,
		r.Methods,
		r.Timeout.Duration,
		r.IdleTimeout.Duration,
		r.ScrapeInterval.Duration,
		r.HealthCheckInterval.Duration,
		r.MonitoringInterval.Duration,
		r.CookieTTL.Duration,
		r.HealthCheck,
	)

	for _, backend := range r.Backends {
		if backend.ID == uuid.Nil {
			log.Debugf("Setting new uuid for %s", r.Name)
			backend.ID = uuid.New()
		}
		for _, cond := range backend.Metricthresholds {
			cond.Compile()
		}
		log.Debugf("Adding existing backend %v to Route %v", backend.ID, r.Name)
		_, err = newRoute.AddExistingBackend(backend)
		if err != nil {
			return nil, err
		}
	}
	return newRoute, err
}

func ConvertInputGatewayToGateway(g *InputGateway) *gateway.Gateway {
	_, newMetricsRepo := metrics.NewMetricsRepository(
		storage.NewLocalStorage(RetentionPeriod, Granulartiy),
		Granulartiy, MetricsChannelPuffersize, ScrapeMetricsChannelPuffersize,
	)
	newGateway := gateway.NewGateway(
		g.Addr,
		newMetricsRepo,
		g.ReadTimeout.Duration,
		g.WriteTimeout.Duration,
		g.HTTPTimeout.Duration,
		g.IdleTimeout.Duration,
	)
	return newGateway
}
