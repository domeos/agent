package main

import (
	"flag"
	"log"
	"os"

	"github.com/domeos/agent/cron"
	"github.com/domeos/agent/funcs"
	"github.com/domeos/agent/g"
	agenthttp "github.com/domeos/agent/http"
	"github.com/google/cadvisor/client"
	"github.com/google/cadvisor/container"
)

type metricSetValue struct {
	container.MetricSet
}

func main() {

	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("agentversion", false, "show version")
	check := flag.Bool("check", false, "check collector")

	flag.Parse()

	if *version {
		os.Exit(0)
	}

	if *check {
		funcs.CheckCollector()
		os.Exit(0)
	}

	g.ParseConfig(*cfg)

	g.InitRootDir()
	g.InitLocalIp()
	g.InitRpcClients()

	//if g.Config().Apiserver == "" {
	//	os.Exit(0)
	//}

	// kubeClient, err := k8s.CreateKubeClient(g.Config().Apiserver)
	// if err != nil {
	// 	log.Fatalf("Failed to k8s create client: %v", err)
	// }
	// k8s.InitializeMetricCollection(kubeClient)

	// Start to run cAdvisor

	localIP := g.LocalIp
	if len(g.Config().IP) > 0 {
		localIP = g.Config().IP
	}
	client, err := client.NewClient("http://" + localIP + ":" + g.Config().CadvisorPort)
	if err != nil {
		log.Fatalf("Failed to k8s create client: %v", err)
		os.Exit(0)
	}
	g.SetContainerClient(client)

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
