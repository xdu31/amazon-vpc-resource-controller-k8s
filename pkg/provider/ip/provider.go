/*


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

package ip

import (
	"fmt"
	"sync"

	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/aws/ec2"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/aws/ec2/api"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/aws/vpc"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/config"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/k8s"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/pool"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/provider"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/provider/ip/eni"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/utils"
	"github.com/aws/amazon-vpc-resource-controller-k8s/pkg/worker"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ipv4Provider struct {
	// lock to allow multiple routines to access the cache concurrently
	lock sync.RWMutex
	// log is the logger initialized with ip provider details
	log logr.Logger
	// ec2APIHelper to make ec2 API calls for IPv4 management
	ec2APIHelper api.EC2APIHelper
	// workerPool with worker routine to execute asynchronous job on the ip provider
	workerPool worker.Worker
	// instanceResources stores the ENIManager and the resource pool per instance
	instanceProviderAndPool map[string]ResourceProviderAndPool
	// k8sWrapper to list the pods on node initialization
	k8sWrapper k8s.K8sWrapper
	// config is the warm pool configuration for the resource IPv4
	config *config.WarmPoolConfig
	// notifyPoolAdded notifies the handler when a new pool is added
	notifyPoolAdded func(resourceName string, nodeName string, pool pool.Pool)
	// notifyPoolDeleted notifies the handler when a new pool is removed
	notifyPoolDeleted func(resourceName string, nodeName string)
}

// InstanceResource contains the instance's ENI manager and the resource pool
type ResourceProviderAndPool struct {
	eniManager   eni.ENIManager
	resourcePool pool.Pool
}

func NewIPv4Provider(log logr.Logger, ec2APIHelper api.EC2APIHelper, workerPool worker.Worker,
	k8sWrapper k8s.K8sWrapper) provider.ResourceProvider {
	return &ipv4Provider{
		log:                     log,
		ec2APIHelper:            ec2APIHelper,
		workerPool:              workerPool,
		k8sWrapper:              k8sWrapper,
	}
}

func (p *ipv4Provider) InitResource(instance ec2.EC2Instance) error {
	nodeName := instance.Name()

	eniManager := eni.NewENIManager(instance)
	presentIPs, err := eniManager.InitResources(p.ec2APIHelper)
	if err != nil {
		return err
	}

	pods, err := p.k8sWrapper.ListPods(nodeName)
	if err != nil {
		return err
	}

	podToResourceMap := map[string]string{}
	usedIPSet := map[string]struct{}{}
	for _, pod := range pods.Items {
		annotation, present := pod.Annotations[config.ResourceNameIPAddress]
		if !present {
			continue
		}
		podToResourceMap[utils.GetNamespacedName(pod.Namespace, pod.Name)] = annotation
		usedIPSet[annotation] = struct{}{}
	}

	warmResources := difference(presentIPs, usedIPSet)

	resourcePool := pool.NewResourcePool(nil, p.config, podToResourceMap,
		warmResources, instance.Name(), getCapacity(instance.Type(), instance.Os()))

	p.putInstanceProviderAndPool(nodeName, resourcePool, eniManager)
	p.notifyPoolAdded(config.ResourceNameIPAddress, nodeName, resourcePool)

	return nil
}

func (p *ipv4Provider) DeInitResource(instance ec2.EC2Instance) error {
	nodeName := instance.Name()
	p.getInstanceProviderAndPool(nodeName)
	p.notifyPoolDeleted(config.ResourceNameIPAddress, nodeName)
	return nil
}

// UpdateResourceCapacity updates the resource capacity based on the type of instance
func (p *ipv4Provider) UpdateResourceCapacity(instance ec2.EC2Instance) error {
	instanceType := instance.Type()
	instanceName := instance.Name()
	os := instance.Os()

	capacity := getCapacity(instanceType, os)

	err := p.k8sWrapper.AdvertiseCapacityIfNotSet(instance.Name(), config.ResourceNameIPAddress, capacity)
	if err != nil {
		return err
	}
	p.log.V(1).Info("advertised capacity",
		"instance", instanceName, "instance type", instanceType, "os", os, "capacity", capacity)

	return nil
}

// SubmitAsyncJob submits an asynchronous job to the worker pool
func (p *ipv4Provider) SubmitAsyncJob(job interface{}) {
	p.workerPool.SubmitJob(job)
}

// ProcessAsyncJob processes the job, the function should be called using the worker pool in order to be processed
// asynchronously
func (p *ipv4Provider) ProcessAsyncJob(job interface{}) (ctrl.Result, error) {
	warmPoolJob, isValid := job.(worker.WarmPoolJob)
	if !isValid {
		return ctrl.Result{}, fmt.Errorf("invalid job type")
	}

	switch warmPoolJob.Operations {
	case worker.OperationCreate:
		p.CreatePrivateIPv4AndUpdatePool(warmPoolJob)
	case worker.OperationDeleted:
		p.DeletePrivateIPv4AndUpdatePool(warmPoolJob)
	}

	return ctrl.Result{}, nil
}

// CreatePrivateIPv4AndUpdatePool executes the Create IPv4 workflow by assigning the desired number of IPv4 address
// provided in the warm pool job
func (p *ipv4Provider) CreatePrivateIPv4AndUpdatePool(job worker.WarmPoolJob) {
	instanceResource, found := p.getInstanceProviderAndPool(job.NodeName)
	if !found {
		p.log.Error(fmt.Errorf("cannot find the instance provider and pool form the cache"), "node", job.NodeName)
		return
	}
	didSucceed := true
	ips, err := instanceResource.eniManager.CreateIPV4Address(job.ResourceCount, p.ec2APIHelper, p.log)
	if err != nil {
		p.log.Error(err, "failed to create all/some of the IPv4 addresses", "created ips", ips)
		didSucceed = false
	}
	job.Resources = ips
	p.updatePoolAndReconcileIfRequired(instanceResource.resourcePool, job, didSucceed)
}

// DeletePrivateIPv4AndUpdatePool executes the Delete IPv4 workflow for the list of IPs provided in the warm pool job
func (p *ipv4Provider) DeletePrivateIPv4AndUpdatePool(job worker.WarmPoolJob) {
	instanceResource, found := p.getInstanceProviderAndPool(job.NodeName)
	if !found {
		p.log.Error(fmt.Errorf("cannot find the instance provider and pool form the cache"), "node", job.NodeName)
		return
	}
	didSucceed := true
	failedIPs, err := instanceResource.eniManager.DeleteIPV4Address(job.Resources, p.ec2APIHelper, p.log)
	if err != nil {
		p.log.Error(err, "failed to delete all/some of the IPv4 addresses", "failed ips", failedIPs)
		didSucceed = false
	}
	job.Resources = failedIPs
	p.updatePoolAndReconcileIfRequired(instanceResource.resourcePool, job, didSucceed)
}

// updatePoolAndReconcileIfRequired updates the resource pool and reconcile again and submit a new job if required
func (p *ipv4Provider) updatePoolAndReconcileIfRequired(resourcePool pool.Pool, job worker.WarmPoolJob, didSucceed bool) {
	shouldReconcile := resourcePool.UpdatePool(job, didSucceed)

	if shouldReconcile {
		job := resourcePool.ReconcilePool()
		if job.Operations != worker.OperationIgnore {
			p.SubmitAsyncJob(job)
		}
	}
}

// putInstanceProviderAndPool stores the node's instance provider and pool to the cache
func (p *ipv4Provider) putInstanceProviderAndPool(nodeName string, resourcePool pool.Pool, manager eni.ENIManager) {
	p.lock.Lock()
	defer p.lock.Unlock()

	resource := ResourceProviderAndPool{
		eniManager:   manager,
		resourcePool: resourcePool,
	}

	p.instanceProviderAndPool[nodeName] = resource
}

// getInstanceProviderAndPool returns the node's instance provider and pool from the cache
func (p *ipv4Provider) getInstanceProviderAndPool(nodeName string) (ResourceProviderAndPool, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	resource, found := p.instanceProviderAndPool[nodeName]
	return resource, found
}

// deleteInstanceProviderAndPool deletes the node's instance provider and pool from the cache
func (p *ipv4Provider) deleteInstanceProviderAndPool(nodeName string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.instanceProviderAndPool, nodeName)
}

// getCapacity returns the capacity based on the instance type and the instance os
func getCapacity(instanceType string, instanceOs string) int {
	// Assign only 1st ENIs non primary IP
	ipLimit, found := vpc.InstanceIPsAvailable[instanceType]
	eniLimit := vpc.InstanceENIsAvailable[instanceType]
	if !found {
		return 0
	}
	var capacity int
	if instanceOs == config.OSWindows {
		capacity = ipLimit - 1
	} else {
		capacity = (ipLimit - 1) * eniLimit
	}

	return capacity
}

// difference returns the difference between the slice and the map in the argument
func difference(allIPs []string, usedIPSet map[string]struct{}) []string {
	var notUsed []string
	for _, ip := range allIPs {
		if _, found := usedIPSet[ip]; !found {
			notUsed = append(notUsed, ip)
		}
	}
	return notUsed
}