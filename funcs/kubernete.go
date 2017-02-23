package funcs

import (
	"sync"

	"strings"

	"github.com/domeos/agent/g"
	"github.com/open-falcon/common/model"
	dto "github.com/prometheus/client_model/go"
)

const (
	k8sHistoryCount int = 2
)

var (
	k8sStatHistory [k8sHistoryCount][]*dto.MetricFamily
	ksLock         = new(sync.RWMutex)
)

func UpdateK8sStat() error {
	g.UpdateK8sStat()
	mfs := g.GetK8sStat()
	ksLock.Lock()
	defer ksLock.Unlock()
	for index, _ := range k8sStatHistory {
		if index != 0 && k8sStatHistory[index-1] != nil {
			k8sStatHistory[index] = k8sStatHistory[index-1]
		}
	}
	k8sStatHistory[0] = mfs
	return nil
}

func k8sStatPrepared() bool {
	ksLock.RLock()
	defer ksLock.RUnlock()
	return k8sStatHistory[0] != nil
}

func K8sMetrics() (L []*model.MetricValue) {
	if !k8sStatPrepared() {
		return nil
	}
	for _, mf := range k8sStatHistory[0] {
		switch mf.GetName() {
		case "kube_replication_controller_status_replicas":
		case "kube_replication_controller_status_replicas_available":
			for _, metric := range mf.GetMetric() {
				for _, label := range metric.GetLabel() {
					if strings.EqualFold(label.GetName(), "replicationcontroller") {
						L = append(L, GaugeValue(mf.GetName(), metric.GetGauge().GetValue(), "rcName="+label.GetValue()))
					}
				}
			}
			break
		}
	}
	return L
}
