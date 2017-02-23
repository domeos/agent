/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8s

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/pkg/api/v1"
)

var (
	dsecReplicationControllerStatusReplicas = prometheus.NewDesc(
		"kube_replication_controller_status_replicas",
		"The number of replicas per deployment.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusReplicasAvailable = prometheus.NewDesc(
		"kube_replication_controller_status_replicas_available",
		"The number of available replicas per deployment.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusReplicasUnavailable = prometheus.NewDesc(
		"kube_replication_controller_status_replicas_unavailable",
		"The number of unavailable replicas per deployment.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusReplicasUpdated = prometheus.NewDesc(
		"kube_replication_controller_status_replicas_updated",
		"The number of updated replicas per deployment.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
)

type replicationcontrollerStore interface {
	List() (rc []v1.ReplicationController, err error)
}

// deploymentCollector collects metrics about all deployments in the cluster.
type replicationcontrollerCollector struct {
	store replicationcontrollerStore
}

// Describe implements the prometheus.Collector interface.
func (rcc *replicationcontrollerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- dsecReplicationControllerStatusReplicas
	ch <- descReplicationControllerStatusReplicasAvailable
}

// Collect implements the prometheus.Collector interface.
func (rcc *replicationcontrollerCollector) Collect(ch chan<- prometheus.Metric) {
	rcs, err := rcc.store.List()
	if err != nil {
		glog.Errorf("listing deployments failed: %s", err)
		return
	}
	for _, r := range rcs {
		rcc.collectReplicaontController(ch, r)
	}
}

func (rcc *replicationcontrollerCollector) collectReplicaontController(ch chan<- prometheus.Metric, rc v1.ReplicationController) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{rc.Namespace, rc.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(dsecReplicationControllerStatusReplicas, float64(rc.Status.Replicas))
	addGauge(descReplicationControllerStatusReplicasAvailable, float64(rc.Status.ReadyReplicas))
}
