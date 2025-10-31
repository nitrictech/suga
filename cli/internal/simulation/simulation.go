package simulation

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/nitrictech/suga/cli/internal/netx"
	"github.com/nitrictech/suga/cli/internal/simulation/database"
	"github.com/nitrictech/suga/cli/internal/simulation/middleware"
	"github.com/nitrictech/suga/cli/internal/simulation/service"
	"github.com/nitrictech/suga/cli/internal/style"
	"github.com/nitrictech/suga/cli/internal/style/icons"
	"github.com/nitrictech/suga/cli/pkg/schema"
	"github.com/nitrictech/suga/cli/pkg/tui"
	pubsubpb "github.com/nitrictech/suga/proto/pubsub/v2"
	storagepb "github.com/nitrictech/suga/proto/storage/v2"
	"github.com/samber/lo"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
)

type SimulationServer struct {
	fs      afero.Fs
	appDir  string
	appSpec *schema.Application
	storagepb.UnimplementedStorageServer
	pubsubpb.UnimplementedPubsubServer

	apiPort         netx.ReservedPort
	fileServerPort  int
	services        map[string]*service.ServiceSimulation
	databaseManager *database.DatabaseManager
	servicesWg      sync.WaitGroup
}

const (
	SUGA_SERVICE_MIN_PORT = 50051
	SUGA_SERVICE_MAX_PORT = 50999
	ENTRYPOINT_MIN_PORT   = 3000
	ENTRYPOINT_MAX_PORT   = 3999
)

func (s *SimulationServer) startSugaApis() error {
	srv := grpc.NewServer()

	storagepb.RegisterStorageServer(srv, s)
	pubsubpb.RegisterPubsubServer(srv, s)

	host := os.Getenv("SUGA_HOST")
	portEnv := os.Getenv("SUGA_PORT")

	if portEnv != "" {
		portInt, err := strconv.Atoi(portEnv)
		if err != nil {
			return fmt.Errorf("failed to parse port: %v", err)
		}

		s.apiPort, err = netx.ReservePort(portInt)
		if err != nil {
			return err
		}
	} else {
		openPort, err := netx.GetNextPort(netx.MinPort(SUGA_SERVICE_MIN_PORT), netx.MaxPort(SUGA_SERVICE_MAX_PORT))
		if err != nil {
			return fmt.Errorf("failed to find open port: %v", err)
		}

		s.apiPort = openPort
	}

	addr := net.JoinHostPort(host, s.apiPort.String())

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	fmt.Println(tui.SugaIntro("App", s.appSpec.Name, "Addr", addr))

	go func() {
		err := srv.Serve(lis)
		if err != nil {
			log.Fatalf("failed to serve listener")
		}
	}()

	return nil
}

func (s *SimulationServer) startDatabases(output io.Writer) error {
	databaseIntents := s.appSpec.DatabaseIntents

	// Create and start a single PostgreSQL instance
	dbManager, err := database.NewDatabaseManager(s.appSpec.Name, s.appDir)
	if err != nil {
		return err
	}

	err = dbManager.Start()
	if err != nil {
		return err
	}

	s.databaseManager = dbManager

	fmt.Fprintf(output, "%s\n\n", style.Purple("Databases"))

	// Create all databases in the PostgreSQL instance
	for dbName, dbIntent := range databaseIntents {
		err = dbManager.CreateDatabase(dbName, *dbIntent)
		if err != nil {
			return err
		}

		fmt.Fprintf(output, "%s Starting %s postgresql://localhost:%d/%s\n",
			greenCheck,
			styledName(dbName, style.Purple),
			dbManager.GetPort(),
			dbName)
	}

	fmt.Fprint(output, "\n")

	return nil
}

var greenCheck = style.Green(icons.Check)

func (s *SimulationServer) startEntrypoints(services map[string]*service.ServiceSimulation) error {
	resourceIntents := s.appSpec.GetResourceIntents()
	serviceProxies := map[string]*httputil.ReverseProxy{}
	for serviceName, service := range services {
		url := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", service.GetPort()),
		}

		serviceProxies[serviceName] = httputil.NewSingleHostReverseProxy(url)
	}

	for entrypointName, entrypoint := range s.appSpec.EntrypointIntents {
		// Reserve a port
		reservedPort, err := netx.GetNextPort(netx.MinPort(ENTRYPOINT_MIN_PORT), netx.MaxPort(ENTRYPOINT_MAX_PORT))
		if err != nil {
			return err
		}

		router := http.NewServeMux()

		for route, target := range entrypoint.Routes {
			spec, ok := resourceIntents[target.TargetName]
			if !ok {
				return fmt.Errorf("resource %s does not exist", target.TargetName)
			}

			if spec.GetType() != "service" && spec.GetType() != "bucket" {
				return fmt.Errorf("only buckets and services can be routed to entrypoints got type :%s", spec.GetType())
			}

			var proxyHandler http.Handler
			styleColor := style.Teal
			if spec.GetType() == "service" {
				service := services[target.TargetName]

				url := &url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("localhost:%d", service.GetPort()),
					Path:   target.BasePath,
				}

				proxyHandler = httputil.NewSingleHostReverseProxy(url)

			} else if spec.GetType() == "bucket" {
				url := &url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("localhost:%d", s.fileServerPort),
					Path:   strings.TrimSuffix(fmt.Sprintf("/%s/%s", target.TargetName, target.BasePath), "/"),
				}
				proxyHandler = httputil.NewSingleHostReverseProxy(url)
				styleColor = style.Green
			}

			proxyLogMiddleware := middleware.ProxyLogging(styledName(entrypointName, style.Orange), styledName(target.TargetName, styleColor), false)
			router.Handle(route, http.StripPrefix(strings.TrimSuffix(route, "/"), proxyLogMiddleware(proxyHandler)))
		}

		go func() {
			err := http.ListenAndServe(fmt.Sprintf(":%d", reservedPort), router)
			if err != nil {
				log.Fatalf("failed to start entrypoint %s: %v", entrypointName, err)
			}
		}()

		fmt.Printf("%s Starting %s http://localhost:%d\n", greenCheck, styledName(entrypointName, style.Orange), reservedPort)
	}

	return nil
}

// CopyDir copies the content of src to dst. src should be a full path.
func (s *SimulationServer) CopyDir(dst, src string) error {
	return afero.Walk(s.fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// copy to this path
		outpath := filepath.Join(dst, strings.TrimPrefix(path, src))

		if info.IsDir() {
			return s.fs.MkdirAll(outpath, info.Mode())
		}

		// handle irregular files
		if !info.Mode().IsRegular() {
			switch info.Mode().Type() & os.ModeType {
			case os.ModeSymlink:
				// For symlinks, we'll just copy the file contents instead
				// since not all Afero filesystems support symlinks
				in, err := s.fs.Open(path)
				if err != nil {
					return err
				}
				defer in.Close()

				fh, err := s.fs.Create(outpath)
				if err != nil {
					return err
				}
				defer fh.Close()

				_, err = io.Copy(fh, in)
				return err
			}
			return nil
		}

		// copy contents of regular file efficiently
		in, err := s.fs.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		// create output
		fh, err := s.fs.Create(outpath)
		if err != nil {
			return err
		}
		defer fh.Close()

		// make it the same
		err = s.fs.Chmod(outpath, info.Mode())
		if err != nil {
			return err
		}

		// copy content
		_, err = io.Copy(fh, in)
		return err
	})
}

func (s *SimulationServer) startServices(output io.Writer) (<-chan service.ServiceEvent, error) {
	serviceIntents := s.appSpec.ServiceIntents

	eventChans := []<-chan service.ServiceEvent{}

	for serviceName, serviceIntent := range serviceIntents {
		port, err := netx.GetNextPort()
		if err != nil {
			return nil, err
		}

		// Clone the service intent to add database connection strings
		intentCopy := *serviceIntent
		if intentCopy.Env == nil {
			intentCopy.Env = make(map[string]string)
		}

		// Inject database connection strings for databases this service has access to
		if s.databaseManager != nil {
			for dbName, dbIntent := range s.appSpec.DatabaseIntents {
				// Check if this service has access to this database
				if dbIntent.Access != nil {
					if _, hasAccess := dbIntent.Access[serviceName]; hasAccess {
						envKey := s.databaseManager.GetEnvVarKey(dbName)
						if envKey == "" {
							return nil, fmt.Errorf("database %s is missing env_var_key but is accessed by service %s", dbName, serviceName)
						}
						connStr := s.databaseManager.GetConnectionString(dbName)
						intentCopy.Env[envKey] = connStr
					}
				}
			}
		}

		simulatedService, eventChan, err := service.NewServiceSimulation(serviceName, intentCopy, port, s.apiPort)
		if err != nil {
			return nil, err
		}

		eventChans = append(eventChans, eventChan)
		s.services[serviceName] = simulatedService

		fmt.Fprintf(output, "%s Starting %s\n", greenCheck, styledName(serviceName, style.Teal))
	}

	for _, simulatedService := range s.services {
		s.servicesWg.Add(1)
		go func(svc *service.ServiceSimulation) {
			defer s.servicesWg.Done()
			err := svc.Start(true)
			if err != nil {
				log.Fatalf("failed to start simulated service: %v", err)
			}
		}(simulatedService)
	}

	// Combine all of the events
	combinedEventsChan := lo.FanIn(100, eventChans...)

	return combinedEventsChan, nil
}

func (s *SimulationServer) handleServiceOutputs(output io.Writer, events <-chan service.ServiceEvent) {

	err := s.fs.MkdirAll(ServicesLogsDir, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to make service log directory: %v", err)
	}

	serviceWriters := make(map[string]io.Writer, len(s.appSpec.ServiceIntents))
	serviceLogs := make(map[string]io.WriteCloser, len(s.appSpec.ServiceIntents))
	for serviceName := range s.appSpec.ServiceIntents {
		serviceWriters[serviceName] = NewPrefixWriter(styledName(serviceName, style.Teal)+" ", output)

		serviceLogPath, err := GetServiceLogPath(s.appDir, serviceName)
		if err != nil {
			log.Fatalf("failed to get service log path for service %s: %v", serviceName, err)
		}

		err = s.fs.Remove(serviceLogPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("failed to remove service log path for service %s: %v", serviceName, err)
		}

		file, err := s.fs.OpenFile(serviceLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("failed to open log file for service %s: %v", serviceName, err)
		}
		serviceLogs[serviceName] = file
	}

	defer func() {
		for _, log := range serviceLogs {
			log.Close()
		}
	}()

	for {
		event := <-events

		if event.Output != nil {
			// write some kind of output for that service
			if writer, ok := serviceWriters[event.GetName()]; ok {
				_, err := writer.Write(event.Content)
				if err != nil {
					log.Fatalf("failed to write service event content")
				}
			} else {
				log.Fatalf("failed to retrieve output writer for service %s", event.GetName())
			}
		}

		if serviceLog, ok := serviceLogs[event.GetName()]; ok {
			_, err := serviceLog.Write(event.Content)
			if err != nil {
				log.Fatalf("failed to write service log")
			}
		}

		if event.PreviousStatus != event.GetStatus() {
			if event.GetStatus() == service.Status_Restarting {
				fmt.Fprintf(output, "\n%s Restarting %s\n\n", style.Red(icons.CircleEmpty), styledName(event.GetName(), style.Teal))
			}
		}
	}
}

var styledNames = map[string]string{}

func styledName(name string, styleFunc func(...string) string) string {
	_, ok := styledNames[name]
	if !ok {
		styledNames[name] = styleFunc(fmt.Sprintf("[%s]", name))
	}

	return styledNames[name]
}

func (s *SimulationServer) Start(output io.Writer) error {
	err := s.startSugaApis()
	if err != nil {
		return err
	}

	// Start databases before services so they can connect
	if len(s.appSpec.DatabaseIntents) > 0 {
		err := s.startDatabases(output)
		if err != nil {
			return err
		}
	}

	var svcEvents <-chan service.ServiceEvent

	if len(s.appSpec.ServiceIntents) > 0 {
		fmt.Fprintf(output, "%s\n\n", style.Teal("Services"))
		svcEvents, err = s.startServices(output)
		if err != nil {
			return err
		}
		fmt.Fprint(output, "\n")
	}

	if len(s.appSpec.BucketIntents) > 0 {
		err := s.startBuckets()
		if err != nil {
			return err
		}
	}

	if len(s.appSpec.EntrypointIntents) > 0 {
		fmt.Fprintf(output, "%s\n\n", style.Orange("Entrypoints"))
		err = s.startEntrypoints(s.services)
		if err != nil {
			return err
		}
		fmt.Fprint(output, "\n")
	}

	fmt.Println(style.Gray("Use Ctrl-C to exit\n"))

	// block on handling service outputs for now
	s.handleServiceOutputs(output, svcEvents)

	return nil
}

// Stop gracefully shuts down the simulation server and cleans up resources
func (s *SimulationServer) Stop() error {
	// Stop services first before stopping database
	for serviceName, svc := range s.services {
		if svc != nil {
			fmt.Printf("Stopping service %s...\n", serviceName)
			svc.Signal(os.Interrupt)
		}
	}

	// Wait for all service goroutines to complete
	s.servicesWg.Wait()

	// Stop the database manager after services have shut down
	if s.databaseManager != nil {
		fmt.Println("Stopping database...")
		if err := s.databaseManager.Stop(); err != nil {
			return fmt.Errorf("failed to stop database manager: %w", err)
		}
	}

	return nil
}

type SimulationServerOption func(*SimulationServer)

func WithAppDirectory(appDir string) SimulationServerOption {
	return func(s *SimulationServer) {
		s.appDir = appDir
	}
}

func NewSimulationServer(fs afero.Fs, appSpec *schema.Application, opts ...SimulationServerOption) *SimulationServer {
	simServer := &SimulationServer{
		fs:       fs,
		appSpec:  appSpec,
		appDir:   ".",
		services: make(map[string]*service.ServiceSimulation),
	}

	for _, o := range opts {
		o(simServer)
	}

	return simServer
}
