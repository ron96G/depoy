package gateway

import (
	"depoy/metrics"
	"depoy/route"
	"depoy/router"
	"depoy/storage"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"
)

//Gateway has a HTTP-Server which has Routes configured for it
type Gateway struct {
	mux          sync.Mutex `yaml:"-" json:"-"`
	Addr         string
	ReadTimeout  int
	WriteTimeout int
	Routes       map[string]*route.Route
	Router       map[string]*router.Router `yaml:"-" json:"-"`
	MetricsRepo  *metrics.Repository
}

//NewGateway returns a new instance of Gateway
func NewGateway(addr string, readTimeout, writeTimeout int) *Gateway {
	g := new(Gateway)
	_, g.MetricsRepo = metrics.NewMetricsRepository(storage.NewLocalStorage(), 5*time.Second)

	go g.MetricsRepo.Listen()

	g.Addr = addr
	g.Routes = make(map[string]*route.Route)

	// map for each HOST
	g.Router = make(map[string]*router.Router)

	// any HOST router
	g.Router["*"] = router.NewRouter()

	// set defaults
	g.ReadTimeout = readTimeout
	g.WriteTimeout = writeTimeout

	return g
}

func (g *Gateway) Reload() {

	log.Info("Reloading backend")

	newRouter := make(map[string]*router.Router)
	newRouter["*"] = router.NewRouter()

	for _, routeItem := range g.Routes {

		if _, found := newRouter[routeItem.Host]; !found {
			// host does not exist
			newRouter[routeItem.Host] = router.NewRouter()
		}

		// add all routes
		for _, method := range routeItem.Methods {
			newRouter[routeItem.Host].AddHandler(method, routeItem.Prefix, routeItem.GetHandler())
		}
	}

	// overwrite existing tree with new
	g.Router = newRouter
}

// Run starts the HTTP-Server of the Gateway
func (g *Gateway) Run() {
	server := http.Server{
		Addr:              g.Addr,
		Handler:           http.TimeoutHandler(g, 10*time.Second, "HTTP Handling Timeout"),
		WriteTimeout:      time.Duration(g.WriteTimeout) * time.Millisecond,
		ReadTimeout:       time.Duration(g.ReadTimeout) * time.Millisecond,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func (g *Gateway) checkIfExists(newRoute *route.Route) error {

	for routeName, route := range g.Routes {
		if routeName == newRoute.Name {
			return fmt.Errorf("Route with name %s already exists", routeName)
		}
		if route.Prefix == newRoute.Prefix && route.Host == newRoute.Host {
			return fmt.Errorf(
				"Route with combination of prefix (%s) and host (%s) already exist. Existing Route: %s",
				route.Prefix, route.Host, routeName)
		}
	}
	return nil
}

func (g *Gateway) GetRoute(routeName string) *route.Route {

	if route, found := g.Routes[routeName]; found {
		return route
	}
	return nil
}

func (g *Gateway) RegisterRoute(newRoute *route.Route) error {
	var err error
	log.Debugf("Trying to register new route %s", newRoute.Name)

	g.mux.Lock()
	defer g.mux.Unlock()

	if newRoute.Name == "" {
		return fmt.Errorf("Route.Name cannot be empty")
	}

	// create id of route
	if err = g.checkIfExists(newRoute); err != nil {
		return err
	}

	newRoute.MetricsRepo = g.MetricsRepo

	if len(newRoute.Backends) == 0 {
		return fmt.Errorf("Route %s has no backends configured for it", newRoute.Name)

	}

	g.Routes[newRoute.Name] = newRoute
	g.Reload()
	return nil
}

func (g *Gateway) RemoveRoute(name string) *route.Route {

	g.mux.Lock()
	defer g.mux.Unlock()

	if route, exists := g.Routes[name]; exists {
		log.Debugf("Removing %s from Gateway.Routes", name)

		route.StopAll()

		for id := range route.Backends {
			route.RemoveBackend(id)
		}

		delete(g.Routes, name)

		g.Reload()
		return route
	}

	return nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if router, found := g.Router[req.Host]; found {
		log.Debugf("Found explicit match for %s", req.Host)
		router.ServeHTTP(w, req)

	} else {
		log.Debugf("Did not find explicit match for %s. Using any Host", req.Host)
		g.Router["*"].ServeHTTP(w, req)
	}
}

func (g *Gateway) GetRouteByName(name string) *route.Route {
	return g.Routes[name]
}

func (g *Gateway) GetRoutes() map[string]*route.Route {
	return g.Routes
}

// SaveConfigToFile saves the current config of the Gateway to a file
func (g *Gateway) SaveConfigToFile(filename string) error {
	b, err := g.ReadConfig()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, b, 0777)
	if err != nil {
		return err
	}

	return nil
}

// ReadConfig reads the current config of the Gateway and returns a []byte
func (g *Gateway) ReadConfig() ([]byte, error) {
	b, err := yaml.Marshal(g)
	if err != nil {
		return nil, err
	}
	return b, nil
}
