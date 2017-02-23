package funcs

import (
	"log"
	"sync"

	"github.com/domeos/agent/g"
	info "github.com/google/cadvisor/info/v1"
	"github.com/open-falcon/common/model"
	//"strconv"
)

const (
	containerHistoryCount int = 2
)

var (
	containerStatHistory = make(map[string][containerHistoryCount]*info.ContainerInfo)
	csLock               = new(sync.RWMutex)
)

func UpdateContainerStat() error {

	g.UpdateCurrentContainers()
	containers := g.CurrentContainers()
	reqParams := &info.ContainerInfoRequest{
		NumStats: 1,
	}

	csLock.Lock()
	defer csLock.Unlock()
	for _, container := range containers {

		containerInfo, err := g.GetCotainerClient().DockerContainer(container, reqParams)
		if err != nil {
			log.Println("Get container info error : ", err.Error())
			continue
		}

		containerStatHistoryCopy := containerStatHistory[container]
		for i := containerHistoryCount - 1; i > 0; i-- {
			containerStatHistoryCopy[i] = containerStatHistoryCopy[i-1]
		}
		containerStatHistoryCopy[0] = &containerInfo
		containerStatHistory[container] = containerStatHistoryCopy
	}

	return nil
}

func ContainerPrepared(container string) bool {
	csLock.RLock()
	defer csLock.RUnlock()
	if _, ok := containerStatHistory[container]; !ok {
		return false
	}
	return containerStatHistory[container][1] != nil
}

func ContainerMetrics() (L []*model.MetricValue) {

	containers := g.CurrentContainers()

	machineInfo, err := g.GetCotainerClient().MachineInfo()
	if err != nil {
		return
	}

	for _, container := range containers {

		if !ContainerPrepared(container) {
			continue
		}

		tag := "id=" + container
		L = append(L, GaugeValue("container.cpu.usage.busy",
			float64(containerStatHistory[container][0].Stats[0].Cpu.Usage.Total-
				containerStatHistory[container][1].Stats[0].Cpu.Usage.Total)/
				float64(procStatHistory[0].Cpu.Total-
					procStatHistory[1].Cpu.Total)/100000.0, tag))

		L = append(L, GaugeValue("container.cpu.usage.user",
			float64(containerStatHistory[container][0].Stats[0].Cpu.Usage.User-
				containerStatHistory[container][1].Stats[0].Cpu.Usage.User)/
				float64(procStatHistory[0].Cpu.Total-
					procStatHistory[1].Cpu.Total)/100000.0, tag))

		L = append(L, GaugeValue("container.cpu.usage.system",
			float64(containerStatHistory[container][0].Stats[0].Cpu.Usage.System-
				containerStatHistory[container][1].Stats[0].Cpu.Usage.System)/
				float64(procStatHistory[0].Cpu.Total-
					procStatHistory[1].Cpu.Total)/100000.0, tag))
		/*
		   for i := 1; i <= int(machineInfo.NumCores); i++ {
		           L = append(L, GaugeValue("container.cpu.usage.core.busy",
		                   float64(containerStatHistory[container][0].Stats[0].Cpu.Usage.PerCpu[i-1] -
		                   containerStatHistory[container][1].Stats[0].Cpu.Usage.PerCpu[i-1]) /
		                   float64(procStatHistory[0].Cpu.Total -
		                   procStatHistory[1].Cpu.Total) / float64(machineInfo.NumCores) / 100000.0, tag + ",core=" + strconv.Itoa(i)))
		   }*/

		if containerStatHistory[container][0].Spec.Memory.Limit > uint64(machineInfo.MemoryCapacity) {
			L = append(L, GaugeValue("container.mem.limit", machineInfo.MemoryCapacity, tag))
			L = append(L, GaugeValue("container.mem.usage.percent", float64(containerStatHistory[container][0].Stats[0].Memory.Usage)/
				float64(machineInfo.MemoryCapacity)*100.0, tag))
		} else {
			L = append(L, GaugeValue("container.mem.limit", containerStatHistory[container][0].Spec.Memory.Limit, tag))
			if containerStatHistory[container][0].Spec.Memory.Limit == 0 {
				L = append(L, GaugeValue("container.mem.usage.percent", 0, tag))
			} else {
				L = append(L, GaugeValue("container.mem.usage.percent", float64(containerStatHistory[container][0].Stats[0].Memory.Usage)/
					float64(containerStatHistory[container][0].Spec.Memory.Limit)*100.0, tag))
			}
		}
		L = append(L, GaugeValue("container.mem.usage", containerStatHistory[container][0].Stats[0].Memory.Usage, tag))
		L = append(L, GaugeValue("container.mem.working_set", containerStatHistory[container][0].Stats[0].Memory.WorkingSet, tag))

		L = append(L, CounterValue("container.net.if.in.bytes", containerStatHistory[container][0].Stats[0].Network.RxBytes, tag))
		L = append(L, CounterValue("container.net.if.in.packets", containerStatHistory[container][0].Stats[0].Network.RxPackets, tag))
		L = append(L, CounterValue("container.net.if.in.errors", containerStatHistory[container][0].Stats[0].Network.RxErrors, tag))
		L = append(L, CounterValue("container.net.if.in.dropped", containerStatHistory[container][0].Stats[0].Network.RxDropped, tag))
		L = append(L, CounterValue("container.net.if.out.bytes", containerStatHistory[container][0].Stats[0].Network.TxBytes, tag))
		L = append(L, CounterValue("container.net.if.out.packets", containerStatHistory[container][0].Stats[0].Network.TxPackets, tag))
		L = append(L, CounterValue("container.net.if.out.errors", containerStatHistory[container][0].Stats[0].Network.TxErrors, tag))
		L = append(L, CounterValue("container.net.if.out.dropped", containerStatHistory[container][0].Stats[0].Network.TxDropped, tag))

		for _, fsStats := range containerStatHistory[container][0].Stats[0].Filesystem {

			fstag := tag + ",device=" + fsStats.Device
			L = append(L, GaugeValue("container.filesystem.limit", fsStats.Limit, fstag))
			L = append(L, GaugeValue("container.filesystem.usage", fsStats.Usage, fstag))
		}
	}
	return
}

func ContainerStatsForPage() (L map[string]map[string]interface{}) {

	L = make(map[string]map[string]interface{})

	g.UpdateCurrentContainers()

	containers := g.CurrentContainers()

	reqParams := &info.ContainerInfoRequest{
		NumStats: 1,
	}

	for _, container := range containers {

		L[container] = make(map[string]interface{})

		containerInfo, err := g.GetCotainerClient().DockerContainer(container, reqParams)
		if err != nil {
			log.Println("Get container info error : ", err.Error())
			continue
		}

		L[container]["container.cpu.usage.total"] = containerInfo.Stats[0].Cpu.Usage.Total
		L[container]["container.cpu.usage.system"] = containerInfo.Stats[0].Cpu.Usage.System
		L[container]["container.cpu.usage.user"] = containerInfo.Stats[0].Cpu.Usage.User
		L[container]["container.cpu.loadaverage"] = containerInfo.Stats[0].Cpu.LoadAverage

		L[container]["container.memory.usage"] = containerInfo.Stats[0].Memory.Usage
		L[container]["container.memory.workingset"] = containerInfo.Stats[0].Memory.WorkingSet

		L[container]["container.network.rxbytes"] = containerInfo.Stats[0].Network.RxBytes
		L[container]["container.network.rxerrors"] = containerInfo.Stats[0].Network.RxErrors
		L[container]["container.network.txbytes"] = containerInfo.Stats[0].Network.TxBytes
		L[container]["container.network.txerrors"] = containerInfo.Stats[0].Network.TxErrors

		for _, fsStats := range containerInfo.Stats[0].Filesystem {

			L[container]["container.filesystem.limit : "+fsStats.Device] = fsStats.Limit
			L[container]["container.filesystem.usage : "+fsStats.Device] = fsStats.Usage
		}
	}
	return
}
