package g

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"net"

	"github.com/google/cadvisor/client"
	info "github.com/google/cadvisor/info/v1"
	"github.com/open-falcon/common/model"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/toolkits/slice"
)

var Root string

func InitRootDir() {
	var err error
	Root, err = os.Getwd()
	if err != nil {
		log.Fatalln("getwd fail:", err)
	}
}

var LocalIp string

func InitLocalIp() {
        if Config().Heartbeat.Enabled {
		conn, err := net.DialTimeout("tcp",Config().Heartbeat.Addr,time.Second*10)
		if err != nil {
			log.Println("get local addr failed !")
		}else{
			LocalIp = strings.Split(conn.LocalAddr().String(),":")[0]
			conn.Close()
		}
	}else{
		log.Println("hearbeat is not enabled, can't get localip")
	}
}

var (
	HbsClient *SingleConnRpcClient
)

func InitRpcClients() {
	if Config().Heartbeat.Enabled {
		HbsClient = &SingleConnRpcClient{
			RpcServer: Config().Heartbeat.Addr,
			Timeout:   time.Duration(Config().Heartbeat.Timeout) * time.Millisecond,
		}
	}

}

func SendToTransfer(metrics []*model.MetricValue) {
	if len(metrics) == 0 {
		return
	}

	debug := Config().Debug

	if debug {
		log.Printf("=> <Total=%d> %v\n", len(metrics), metrics[0])
	}

	var resp model.TransferResponse
	SendMetrics(metrics, &resp)

	if debug {
		log.Println("<=", &resp)
	}
}

var (
	reportUrls     map[string]string
	reportUrlsLock = new(sync.RWMutex)
)

func ReportUrls() map[string]string {
	reportUrlsLock.RLock()
	defer reportUrlsLock.RUnlock()
	return reportUrls
}

func SetReportUrls(urls map[string]string) {
	reportUrlsLock.RLock()
	defer reportUrlsLock.RUnlock()
	reportUrls = urls
}

var (
	reportPorts     []int64
	reportPortsLock = new(sync.RWMutex)
)

func ReportPorts() []int64 {
	reportPortsLock.RLock()
	defer reportPortsLock.RUnlock()
	return reportPorts
}

func SetReportPorts(ports []int64) {
	reportPortsLock.Lock()
	defer reportPortsLock.Unlock()
	reportPorts = ports
}

var (
	duPaths     []string
	duPathsLock = new(sync.RWMutex)
)

func DuPaths() []string {
	duPathsLock.RLock()
	defer duPathsLock.RUnlock()
	return duPaths
}

func SetDuPaths(paths []string) {
	duPathsLock.Lock()
	defer duPathsLock.Unlock()
	duPaths = paths
}

var (
	// tags => {1=>name, 2=>cmdline}
	// e.g. 'name=falcon-agent'=>{1=>falcon-agent}
	// e.g. 'cmdline=xx'=>{2=>xx}
	reportProcs     map[string]map[int]string
	reportProcsLock = new(sync.RWMutex)
)

func ReportProcs() map[string]map[int]string {
	reportProcsLock.RLock()
	defer reportProcsLock.RUnlock()
	return reportProcs
}

func SetReportProcs(procs map[string]map[int]string) {
	reportProcsLock.Lock()
	defer reportProcsLock.Unlock()
	reportProcs = procs
}

var (
	ips     []string
	ipsLock = new(sync.Mutex)
)

func TrustableIps() []string {
	ipsLock.Lock()
	defer ipsLock.Unlock()
	return ips
}

func SetTrustableIps(ipStr string) {
	arr := strings.Split(ipStr, ",")
	ipsLock.Lock()
	defer ipsLock.Unlock()
	ips = arr
}

func IsTrustable(remoteAddr string) bool {
	ip := remoteAddr
	idx := strings.LastIndex(remoteAddr, ":")
	if idx > 0 {
		ip = remoteAddr[0:idx]
	}

	if ip == "127.0.0.1" {
		return true
	}

	return slice.ContainsString(TrustableIps(), ip)
}

var (
	currentContainers     []string
	currentContainersLock = new(sync.RWMutex)
)

func CurrentContainers() []string {
	currentContainersLock.RLock()
	defer currentContainersLock.RUnlock()
	return currentContainers
}

func SetCurrentContainers(containers []string) {
	currentContainersLock.Lock()
	defer currentContainersLock.Unlock()
	currentContainers = containers
}

func UpdateCurrentContainers() {
	query := &info.ContainerInfoRequest{
		NumStats: 3,
	}
	dockerContainers, err := GetCotainerClient().AllDockerContainers(query)
	if err != nil {
		log.Println("Get docker containers error : %s", err.Error())
		return
	}
	containers := make([]string, 0)
	for _, container := range dockerContainers {
		containers = append(containers, container.Id)
	}
	SetCurrentContainers(containers)
}

var (
	cadvisorClient *client.Client
	clientLock     = new(sync.RWMutex)
)

func GetCotainerClient() *client.Client {
	clientLock.RLock()
	defer clientLock.RUnlock()
	return cadvisorClient
}

func SetContainerClient(setClient *client.Client) {
	clientLock.Lock()
	defer clientLock.Unlock()
	cadvisorClient = setClient
}

var (
	k8sStat     []*dto.MetricFamily
	k8sStatLock = new(sync.RWMutex)
)

func GetK8sStat() []*dto.MetricFamily {
	k8sStatLock.RLock()
	defer k8sStatLock.RUnlock()
	return k8sStat
}

func SetK8sStat(setK8sStat []*dto.MetricFamily) {
	k8sStatLock.Lock()
	defer k8sStatLock.Unlock()
	k8sStat = setK8sStat
}

func UpdateK8sStat() {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		log.Println("Get kubernetes info err : %s", err.Error())
	} else {
		SetK8sStat(mfs)
	}
}
