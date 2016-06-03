package main

import (
        "os"
        "fmt"
        "log"
	"flag"
        "time"
        "runtime"
        "syscall"
        "net/http"
        "os/signal"
        "net/http/pprof"
	"github.com/domeos/agent/cron"
	"github.com/domeos/agent/funcs"
	"github.com/domeos/agent/g"
	agenthttp "github.com/domeos/agent/http"
        cadvisorhttp "github.com/google/cadvisor/http"
        "github.com/google/cadvisor/manager"
        "github.com/google/cadvisor/utils/sysfs"
        "github.com/google/cadvisor/version"
        "github.com/google/cadvisor/cache/memory"
	"github.com/google/cadvisor/container"

)

var argIp = flag.String("listen_ip", "", "IP to listen on, defaults to all IPs")
var argPort = flag.Int("port", 8080, "port to listen")
var maxProcs = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).")

var versionFlag = flag.Bool("version", false, "print cAdvisor version and exit")

var httpAuthFile = flag.String("http_auth_file", "", "HTTP auth file for the web UI")
var httpAuthRealm = flag.String("http_auth_realm", "localhost", "HTTP auth realm for the web UI")
var httpDigestFile = flag.String("http_digest_file", "", "HTTP digest file for the web UI")
var httpDigestRealm = flag.String("http_digest_realm", "localhost", "HTTP digest file for the web UI")

var prometheusEndpoint = flag.String("prometheus_endpoint", "/metrics", "Endpoint to expose Prometheus metrics on")

var maxHousekeepingInterval = flag.Duration("max_housekeeping_interval", 60*time.Second, "Largest interval to allow between container housekeepings")
var allowDynamicHousekeeping = flag.Bool("allow_dynamic_housekeeping", true, "Whether to allow the housekeeping interval to be dynamic")

var enableProfiling = flag.Bool("profiling", false, "Enable profiling via web interface host:port/debug/pprof/")

var storageDuration = flag.Duration("storage_duration", 2*time.Minute, "How long to keep data stored (Default: 2min).")

var ignoreMetrics metricSetValue = metricSetValue{container.MetricSet{container.NetworkTcpUsageMetrics: struct{}{}}}

type metricSetValue struct {
	container.MetricSet
}

func main() {

	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("agentversion", false, "show version")
	check := flag.Bool("check", false, "check collector")

	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	if *check {
		funcs.CheckCollector()
		os.Exit(0)
	}

	g.ParseConfig(*cfg)

	g.InitRootDir()
	g.InitLocalIps()
	g.InitRpcClients()

        // Start to run cAdvisor

        g.SetContainerManager(startContainerMonitor())

	funcs.BuildMappers()

	go cron.InitDataHistory()

        cron.ReportAgentStatus()
        cron.SyncMinePlugins()
        cron.SyncBuiltinMetrics()
        cron.SyncTrustableIps()
        cron.Collect()

	go agenthttp.Start()

        select {}
}

func startContainerMonitor() manager.Manager {
        flag.Parse()

        if *versionFlag {
                log.Printf("cAdvisor version %s (%s)\n", version.Info["version"], version.Info["revision"])
                os.Exit(0)
        }

        setMaxProcs()

        memoryStorage := memory.New(*storageDuration, nil)

        sysFs, err := sysfs.NewRealSysFs()
        if err != nil {
                log.Fatalf("Failed to create a system interface: %s", err)
        }

        containerManager, err := manager.New(memoryStorage, sysFs, *maxHousekeepingInterval, *allowDynamicHousekeeping, ignoreMetrics.MetricSet)
        if err != nil {
                log.Fatalf("Failed to create a Container Manager: %s", err)
        }

        mux := http.NewServeMux()

        if *enableProfiling {
                mux.HandleFunc("/debug/pprof/", pprof.Index)
                mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
                mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
                mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
        }

        // Register all HTTP handlers.
        err = cadvisorhttp.RegisterHandlers(mux, containerManager, *httpAuthFile, *httpAuthRealm, *httpDigestFile, *httpDigestRealm)
        if err != nil {
                log.Fatalf("Failed to register HTTP handlers: %v", err)
        }

        cadvisorhttp.RegisterPrometheusHandler(mux, containerManager, *prometheusEndpoint, nil)

        // Start the manager.
        if err := containerManager.Start(); err != nil {
                log.Fatalf("Failed to start container manager: %v", err)
        }

        // Install signal handler.
        installSignalHandler(containerManager)

        log.Printf("Starting cAdvisor version: %s-%s on port %d", version.Info["version"], version.Info["revision"], *argPort)

        addr := fmt.Sprintf("%s:%d", *argIp, *argPort)
        go http.ListenAndServe(addr, mux)

        return containerManager
}

func setMaxProcs() {
        // Allow as many threads as we have cores unless the user specified a value.
        var numProcs int
        if *maxProcs < 1 {
                numProcs = runtime.NumCPU()
        } else {
                numProcs = *maxProcs
        }
        runtime.GOMAXPROCS(numProcs)

        // Check if the setting was successful.
        actualNumProcs := runtime.GOMAXPROCS(0)
        if actualNumProcs != numProcs {
                log.Printf("Specified max procs of %v but using %v", numProcs, actualNumProcs)
        }
}

func installSignalHandler(containerManager manager.Manager) {
        c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

        // Block until a signal is received.
        go func() {
                sig := <-c
                if err := containerManager.Stop(); err != nil {
                        log.Printf("Failed to stop container manager: %v", err)
                }
                log.Printf("Exiting given signal: %v", sig)
                os.Exit(0)
        }()
}
