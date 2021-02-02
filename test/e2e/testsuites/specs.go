/*
Copyright 2019 The Kubernetes Authors.

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

package testsuites

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/blob-csi-driver/pkg/blob"
	"sigs.k8s.io/blob-csi-driver/test/e2e/driver"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

type PodDetails struct {
	Cmd     string
	Volumes []VolumeDetails
}

type VolumeDetails struct {
	VolumeType            string
	FSType                string
	Encrypted             bool
	MountOptions          []string
	ClaimSize             string
	ReclaimPolicy         *v1.PersistentVolumeReclaimPolicy
	VolumeBindingMode     *storagev1.VolumeBindingMode
	AllowedTopologyValues []string
	VolumeMode            VolumeMode
	VolumeMount           VolumeMountDetails
	VolumeDevice          VolumeDeviceDetails
	// Optional, used with pre-provisioned volumes
	VolumeID string
	// Optional, used with PVCs created from snapshots
	DataSource         *DataSource
	ContainerName      string
	NodeStageSecretRef string
}

type VolumeMode int

const (
	FileSystem VolumeMode = iota
	Block
)

const (
	APIVersionv1alpha1 = "v1alpha1"
)

var (
	isAzureStackCloud                            = strings.EqualFold(os.Getenv("AZURE_CLOUD_NAME"), "AZURESTACKCLOUD")
	azureStackStorageEndpointSuffix              = os.Getenv("AZURE_STACK_STORAGE_ENDPOINT_SUFFIX")
	azurePublicCloudSupportedStorageAccountTypes = []string{"Standard_LRS", "Premium_LRS", "Standard_GRS", "Standard_RAGRS"}
	azureStackCloudSupportedStorageAccountTypes  = []string{"Standard_LRS", "Premium_LRS"}
)

type VolumeMountDetails struct {
	NameGenerate      string
	MountPathGenerate string
	ReadOnly          bool
}

type VolumeDeviceDetails struct {
	NameGenerate string
	DevicePath   string
}

type DataSource struct {
	Name string
}

func (pod *PodDetails) SetupWithDynamicVolumes(client clientset.Interface, namespace *v1.Namespace, csiDriver driver.DynamicPVTestDriver, storageClassParameters map[string]string) (*TestPod, []func()) {
	tpod := NewTestPod(client, namespace, pod.Cmd)
	cleanupFuncs := make([]func(), 0)
	for n, v := range pod.Volumes {
		tpvc, funcs := v.SetupDynamicPersistentVolumeClaim(client, namespace, csiDriver, storageClassParameters)
		cleanupFuncs = append(cleanupFuncs, funcs...)
		if v.VolumeMode == Block {
			tpod.SetupRawBlockVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", v.VolumeDevice.NameGenerate, n+1), v.VolumeDevice.DevicePath)
		} else {
			tpod.SetupVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", v.VolumeMount.NameGenerate, n+1), fmt.Sprintf("%s%d", v.VolumeMount.MountPathGenerate, n+1), v.VolumeMount.ReadOnly)
		}
	}
	return tpod, cleanupFuncs
}

// SetupWithDynamicMultipleVolumes each pod will be mounted with multiple volumes with different storage account types
func (pod *PodDetails) SetupWithDynamicMultipleVolumes(client clientset.Interface, namespace *v1.Namespace, csiDriver driver.DynamicPVTestDriver) (*TestPod, []func()) {
	tpod := NewTestPod(client, namespace, pod.Cmd)
	cleanupFuncs := make([]func(), 0)
	supportedStorageAccountTypes := azurePublicCloudSupportedStorageAccountTypes
	if isAzureStackCloud {
		supportedStorageAccountTypes = azureStackCloudSupportedStorageAccountTypes
	}
	accountTypeCount := len(supportedStorageAccountTypes)
	for n, v := range pod.Volumes {
		storageClassParameters := map[string]string{"skuName": supportedStorageAccountTypes[n%accountTypeCount]}
		tpvc, funcs := v.SetupDynamicPersistentVolumeClaim(client, namespace, csiDriver, storageClassParameters)
		cleanupFuncs = append(cleanupFuncs, funcs...)
		if v.VolumeMode == Block {
			tpod.SetupRawBlockVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", v.VolumeDevice.NameGenerate, n+1), v.VolumeDevice.DevicePath)
		} else {
			tpod.SetupVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", v.VolumeMount.NameGenerate, n+1), fmt.Sprintf("%s%d", v.VolumeMount.MountPathGenerate, n+1), v.VolumeMount.ReadOnly)
		}
	}
	return tpod, cleanupFuncs
}

func (pod *PodDetails) SetupWithPreProvisionedVolumes(client clientset.Interface, namespace *v1.Namespace, csiDriver driver.PreProvisionedVolumeTestDriver) (*TestPod, []func()) {
	tpod := NewTestPod(client, namespace, pod.Cmd)
	cleanupFuncs := make([]func(), 0)
	for n, v := range pod.Volumes {
		tpvc, funcs := v.SetupPreProvisionedPersistentVolumeClaim(client, namespace, csiDriver)
		cleanupFuncs = append(cleanupFuncs, funcs...)

		if v.VolumeMode == Block {
			tpod.SetupRawBlockVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", v.VolumeDevice.NameGenerate, n+1), v.VolumeDevice.DevicePath)
		} else {
			tpod.SetupVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", v.VolumeMount.NameGenerate, n+1), fmt.Sprintf("%s%d", v.VolumeMount.MountPathGenerate, n+1), v.VolumeMount.ReadOnly)
		}
	}
	return tpod, cleanupFuncs
}

func (pod *PodDetails) SetupWithExistingCredentials(client clientset.Interface, namespace *v1.Namespace, csiDriver driver.PreProvisionedVolumeTestDriver) (*TestPod, []func()) {
	tpod := NewTestPod(client, namespace, pod.Cmd)
	cleanupFuncs := make([]func(), 0)
	for n, volume := range pod.Volumes {
		resourceGroupName, accountName, containerName, err := blob.GetContainerInfo(volume.VolumeID)
		if err != nil {
			framework.ExpectNoError(err, fmt.Sprintf("Error GetContainerInfo from volumeID(%s): %v", volume.VolumeID, err))
		}
		parameters := map[string]string{
			"resourceGroup":  resourceGroupName,
			"storageAccount": accountName,
			"containerName":  containerName,
		}
		if isAzureStackCloud {
			parameters["storageendpointsuffix"] = azureStackStorageEndpointSuffix
		}

		ginkgo.By("creating the storageclass with existing credentials")
		sc := csiDriver.GetPreProvisionStorageClass(parameters, volume.MountOptions, volume.ReclaimPolicy, volume.VolumeBindingMode, volume.AllowedTopologyValues, namespace.Name)
		tsc := NewTestStorageClass(client, namespace, sc)
		createdStorageClass := tsc.Create()
		cleanupFuncs = append(cleanupFuncs, tsc.Cleanup)

		ginkgo.By("creating pvc with storageclass")
		tpvc := NewTestPersistentVolumeClaim(client, namespace, volume.ClaimSize, volume.VolumeMode, &createdStorageClass)
		tpvc.Create()
		cleanupFuncs = append(cleanupFuncs, tpvc.Cleanup)

		ginkgo.By("validating the pvc")
		tpvc.WaitForBound()
		tpvc.ValidateProvisionedPersistentVolume()
		tpod.SetupVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", volume.VolumeMount.NameGenerate, n+1), fmt.Sprintf("%s%d", volume.VolumeMount.MountPathGenerate, n+1), volume.VolumeMount.ReadOnly)
	}
	return tpod, cleanupFuncs
}

func (pod *PodDetails) SetupDeployment(client clientset.Interface, namespace *v1.Namespace, csiDriver driver.DynamicPVTestDriver, storageClassParameters map[string]string) (*TestDeployment, []func()) {
	cleanupFuncs := make([]func(), 0)
	volume := pod.Volumes[0]
	ginkgo.By("setting up the StorageClass")
	if isAzureStackCloud {
		storageClassParameters["storageendpointsuffix"] = azureStackStorageEndpointSuffix
	}
	storageClass := csiDriver.GetDynamicProvisionStorageClass(storageClassParameters, volume.MountOptions, volume.ReclaimPolicy, volume.VolumeBindingMode, volume.AllowedTopologyValues, namespace.Name)
	tsc := NewTestStorageClass(client, namespace, storageClass)
	createdStorageClass := tsc.Create()
	cleanupFuncs = append(cleanupFuncs, tsc.Cleanup)
	ginkgo.By("setting up the PVC")
	tpvc := NewTestPersistentVolumeClaim(client, namespace, volume.ClaimSize, volume.VolumeMode, &createdStorageClass)
	tpvc.Create()
	tpvc.WaitForBound()
	tpvc.ValidateProvisionedPersistentVolume()
	cleanupFuncs = append(cleanupFuncs, tpvc.Cleanup)
	ginkgo.By("setting up the Deployment")
	tDeployment := NewTestDeployment(client, namespace, pod.Cmd, tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", volume.VolumeMount.NameGenerate, 1), fmt.Sprintf("%s%d", volume.VolumeMount.MountPathGenerate, 1), volume.VolumeMount.ReadOnly)

	cleanupFuncs = append(cleanupFuncs, tDeployment.Cleanup)
	return tDeployment, cleanupFuncs
}

func (volume *VolumeDetails) SetupDynamicPersistentVolumeClaim(client clientset.Interface, namespace *v1.Namespace, csiDriver driver.DynamicPVTestDriver, storageClassParameters map[string]string) (*TestPersistentVolumeClaim, []func()) {
	cleanupFuncs := make([]func(), 0)
	ginkgo.By("setting up the StorageClass")
	if isAzureStackCloud {
		storageClassParameters["storageendpointsuffix"] = azureStackStorageEndpointSuffix
	}
	storageClass := csiDriver.GetDynamicProvisionStorageClass(storageClassParameters, volume.MountOptions, volume.ReclaimPolicy, volume.VolumeBindingMode, volume.AllowedTopologyValues, namespace.Name)
	tsc := NewTestStorageClass(client, namespace, storageClass)
	createdStorageClass := tsc.Create()
	cleanupFuncs = append(cleanupFuncs, tsc.Cleanup)
	ginkgo.By("setting up the PVC and PV")
	var tpvc *TestPersistentVolumeClaim
	if volume.DataSource != nil {
		dataSource := &v1.TypedLocalObjectReference{
			Name: volume.DataSource.Name,
		}
		tpvc = NewTestPersistentVolumeClaimWithDataSource(client, namespace, volume.ClaimSize, volume.VolumeMode, &createdStorageClass, dataSource)
	} else {
		tpvc = NewTestPersistentVolumeClaim(client, namespace, volume.ClaimSize, volume.VolumeMode, &createdStorageClass)
	}
	tpvc.Create()
	cleanupFuncs = append(cleanupFuncs, tpvc.Cleanup)
	// PV will not be ready until PVC is used in a pod when volumeBindingMode: WaitForFirstConsumer
	if volume.VolumeBindingMode == nil || *volume.VolumeBindingMode == storagev1.VolumeBindingImmediate {
		tpvc.WaitForBound()
		tpvc.ValidateProvisionedPersistentVolume()
	}

	return tpvc, cleanupFuncs
}

func (volume *VolumeDetails) SetupPreProvisionedPersistentVolumeClaim(client clientset.Interface, namespace *v1.Namespace, csiDriver driver.PreProvisionedVolumeTestDriver) (*TestPersistentVolumeClaim, []func()) {
	cleanupFuncs := make([]func(), 0)
	ginkgo.By("setting up the PV")
	attrib := make(map[string]string)
	if volume.ContainerName != "" {
		attrib["containerName"] = volume.ContainerName
	}
	nodeStageSecretRef := volume.NodeStageSecretRef
	if isAzureStackCloud {
		attrib["storageendpointsuffix"] = azureStackStorageEndpointSuffix
	}
	pv := csiDriver.GetPersistentVolume(volume.VolumeID, volume.FSType, volume.ClaimSize, volume.ReclaimPolicy, namespace.Name, attrib, nodeStageSecretRef)
	tpv := NewTestPreProvisionedPersistentVolume(client, pv)
	tpv.Create()
	ginkgo.By("setting up the PVC")
	tpvc := NewTestPersistentVolumeClaim(client, namespace, volume.ClaimSize, volume.VolumeMode, nil)
	tpvc.Create()
	cleanupFuncs = append(cleanupFuncs, tpvc.DeleteBoundPersistentVolume)
	cleanupFuncs = append(cleanupFuncs, tpvc.Cleanup)
	tpvc.WaitForBound()
	tpvc.ValidateProvisionedPersistentVolume()

	return tpvc, cleanupFuncs
}
