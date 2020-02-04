//
// Copyright (c) 2019-2020 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package component

import (
	"errors"
	"github.com/che-incubator/che-workspace-crd-operator/pkg/controller/workspace/server"
	"strings"

	workspaceApi "github.com/che-incubator/che-workspace-crd-operator/pkg/apis/workspace/v1alpha1"
	. "github.com/che-incubator/che-workspace-crd-operator/pkg/controller/workspace/config"
	. "github.com/che-incubator/che-workspace-crd-operator/pkg/controller/workspace/model"
	"github.com/eclipse/che-plugin-broker/model"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ConvertToCoreObjects(workspace *workspaceApi.Workspace) (*WorkspaceProperties, *workspaceApi.WorkspaceRouting, []ComponentInstanceStatus, []runtime.Object, error) {

	uid, err := uuid.Parse(string(workspace.ObjectMeta.UID))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	workspaceProperties := WorkspaceProperties{
		Namespace:     workspace.Namespace,
		WorkspaceId:   "workspace" + strings.Join(strings.Split(uid.String(), "-")[0:3], ""),
		WorkspaceName: workspace.Name,
		Started:       workspace.Spec.Started,
		RoutingClass:  workspace.Spec.RoutingClass,
	}

	if !workspaceProperties.Started {
		return &workspaceProperties, &workspaceApi.WorkspaceRouting{
			ObjectMeta: metav1.ObjectMeta{
				Name:      workspaceProperties.WorkspaceId,
				Namespace: workspaceProperties.Namespace,
			},
			Spec: workspaceApi.WorkspaceRoutingSpec{
				Exposed:             workspaceProperties.Started,
				RoutingClass:        workspaceProperties.RoutingClass,
				IngressGlobalDomain: ControllerCfg.GetIngressGlobalDomain(),
				WorkspacePodSelector: map[string]string{
					CheOriginalNameLabel: CheOriginalName,
					WorkspaceIDLabel:     workspaceProperties.WorkspaceId,
				},
				Services: map[string]workspaceApi.ServiceDescription{},
			},
		}, nil, []runtime.Object{}, nil
	}

	mainDeployment, err := buildMainDeployment(workspaceProperties, workspace)
	if err != nil {
		return &workspaceProperties, nil, nil, nil, err
	}

	err = setupPersistentVolumeClaim(workspace, mainDeployment)
	if err != nil {
		return &workspaceProperties, nil, nil, nil, err
	}

	cheRestApisK8sObjects, externalUrl, err := server.AddCheRestApis(workspaceProperties, &mainDeployment.Spec.Template.Spec)
	if err != nil {
		return &workspaceProperties, nil, nil, nil, err
	}
	workspaceProperties.CheApiExternal = externalUrl

	workspaceRouting, componentStatuses, k8sComponentsObjects, err := setupComponents(workspaceProperties, workspace.Spec.Devfile, mainDeployment)
	if err != nil {
		return &workspaceProperties, nil, nil, nil, err
	}
	k8sComponentsObjects = append(k8sComponentsObjects, cheRestApisK8sObjects...)

	return &workspaceProperties, workspaceRouting, componentStatuses, append(k8sComponentsObjects, mainDeployment), nil
}

func buildMainDeployment(wkspProps WorkspaceProperties, workspace *workspaceApi.Workspace) (*appsv1.Deployment, error) {
	var workspaceDeploymentName = wkspProps.WorkspaceId + "." + CheOriginalName
	var terminationGracePeriod int64
	var replicas int32
	if wkspProps.Started {
		replicas = 1
	}

	var autoMountServiceAccount = ServiceAccount != ""

	fromIntOne := intstr.FromInt(1)

	var user int64 = 1234

	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspaceDeploymentName,
			Namespace: workspace.Namespace,
			Labels: map[string]string{
				WorkspaceIDLabel: wkspProps.WorkspaceId,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment":         workspaceDeploymentName,
					CheOriginalNameLabel: CheOriginalName,
					WorkspaceIDLabel:     wkspProps.WorkspaceId,
				},
			},
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: "RollingUpdate",
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &fromIntOne,
					MaxUnavailable: &fromIntOne,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment":         workspaceDeploymentName,
						CheOriginalNameLabel: CheOriginalName,
						WorkspaceIDLabel:     wkspProps.WorkspaceId,
						WorkspaceNameLabel:   wkspProps.WorkspaceName,
					},
					Name: workspaceDeploymentName,
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken:  &autoMountServiceAccount,
					RestartPolicy:                 "Always",
					TerminationGracePeriodSeconds: &terminationGracePeriod,
					Containers:                    []corev1.Container{},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &user,
						FSGroup:   &user,
					},
				},
			},
		},
	}
	if ServiceAccount != "" {
		deploy.Spec.Template.Spec.ServiceAccountName = ServiceAccount
	}

	return &deploy, nil
}

func setupPersistentVolumeClaim(workspace *workspaceApi.Workspace, deployment *appsv1.Deployment) error {
	var workspaceClaim = corev1.PersistentVolumeClaimVolumeSource{
		ClaimName: "claim-che-workspace",
	}
	deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "claim-che-workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &workspaceClaim,
			},
		},
	}
	return nil
}

func setupComponents(names WorkspaceProperties, devfile workspaceApi.DevfileSpec, deployment *appsv1.Deployment) (*workspaceApi.WorkspaceRouting, []ComponentInstanceStatus, []runtime.Object, error) {
	components := devfile.Components
	k8sObjects := []runtime.Object{}

	pluginFQNs := []model.PluginFQN{}

	componentInstanceStatuses := []ComponentInstanceStatus{}

	for _, component := range components {
		var componentType = component.Type
		var err error
		var componentInstanceStatus *ComponentInstanceStatus
		switch componentType {
		case workspaceApi.CheEditor, workspaceApi.ChePlugin:
			componentInstanceStatus, err = setupChePlugin(names, &component)
			if err != nil {
				return nil, nil, nil, err
			}
			if componentInstanceStatus.PluginFQN != nil {
				pluginFQNs = append(pluginFQNs, *componentInstanceStatus.PluginFQN)
			}
			break
		case workspaceApi.Kubernetes, workspaceApi.Openshift:
			componentInstanceStatus, err = setupK8sLikeComponent(names, &component)
			break
		case workspaceApi.Dockerimage:
			componentInstanceStatus, err = setupDockerimageComponent(names, devfile.Commands, &component)
			break
		}
		if err != nil {
			return nil, nil, nil, err
		}
		k8sObjects = append(k8sObjects, componentInstanceStatus.ExternalObjects...)
		componentInstanceStatuses = append(componentInstanceStatuses, *componentInstanceStatus)
	}

	err := mergeWorkspaceAdditions(deployment, componentInstanceStatuses, k8sObjects)
	if err != nil {
		return nil, nil, nil, err
	}

	precreateSubpathsInitContainer(names, &deployment.Spec.Template.Spec)
	initContainersK8sObjects, err := setupPluginInitContainers(names, &deployment.Spec.Template.Spec, pluginFQNs)
	if err != nil {
		return nil, nil, nil, err
	}

	k8sObjects = append(k8sObjects, initContainersK8sObjects...)

	workspaceRouting := buildWorkspaceRouting(names, componentInstanceStatuses)

	// TODO store the annotation of the workspaceAPi: with the defer ????

	return workspaceRouting, componentInstanceStatuses, k8sObjects, nil
}

func buildWorkspaceRouting(wkspProperties WorkspaceProperties, componentInstanceStatuses []ComponentInstanceStatus) *workspaceApi.WorkspaceRouting {
	services := map[string]workspaceApi.ServiceDescription{}
	for _, componentInstanceStatus := range componentInstanceStatuses {
		for containerName, container := range componentInstanceStatus.Containers {
			containerEndpoints := []workspaceApi.Endpoint{}
			for _, port := range container.Ports {
				port64 := int64(port)
				for _, endpoint := range componentInstanceStatus.Endpoints {
					if endpoint.Port != port64 {
						continue
					}
					if endpoint.Attributes == nil {
						endpoint.Attributes = map[workspaceApi.EndpointAttribute]string{}
					}
					// public is the default.
					if _, exists := endpoint.Attributes[workspaceApi.PUBLIC_ENDPOINT_ATTRIBUTE]; !exists {
						endpoint.Attributes[workspaceApi.PUBLIC_ENDPOINT_ATTRIBUTE] = "true"
					}
					containerEndpoints = append(containerEndpoints, endpoint)
				}
			}
			if len(containerEndpoints) > 0 {
				services[containerName] = workspaceApi.ServiceDescription{
					ServiceName: containerServiceName(wkspProperties, containerName),
					Endpoints:   containerEndpoints,
				}
			}
		}
	}
	return &workspaceApi.WorkspaceRouting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wkspProperties.WorkspaceId,
			Namespace: wkspProperties.Namespace,
		},
		Spec: workspaceApi.WorkspaceRoutingSpec{
			Exposed:             wkspProperties.Started,
			RoutingClass:        wkspProperties.RoutingClass,
			IngressGlobalDomain: ControllerCfg.GetIngressGlobalDomain(),
			WorkspacePodSelector: map[string]string{
				CheOriginalNameLabel: CheOriginalName,
				WorkspaceIDLabel:     wkspProperties.WorkspaceId,
			},
			Services: services,
		},
	}
}

//TODO Think of the admission controller to add the name of the user in the workspace?
// In any case add the name of the users in the custom resource of the workspace. + the workspace routing class.
func precreateSubpathsInitContainer(names WorkspaceProperties, podSpec *corev1.PodSpec) {
	podSpec.InitContainers = append(podSpec.InitContainers, corev1.Container{
		Name:    "precreate-subpaths",
		Image:   "registry.access.redhat.com/ubi8/ubi-minimal",
		Command: []string{"/usr/bin/mkdir"},
		Args: []string{
			"-p",
			"-v",
			"-m",
			"777",
			"/tmp/che-workspaces/" + names.WorkspaceId,
		},
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				MountPath: "/tmp/che-workspaces",
				Name:      "claim-che-workspace",
				ReadOnly:  false,
			},
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
	})
}

func mergeWorkspaceAdditions(workspaceDeployment *appsv1.Deployment, componentInstanceStatuses []ComponentInstanceStatus, k8sObjects []runtime.Object) error {
	workspacePodAdditions := []corev1.PodTemplateSpec{}
	for _, componentInstanceStatus := range componentInstanceStatuses {
		if componentInstanceStatus.WorkspacePodAdditions == nil {
			continue
		}
		workspacePodAdditions = append(workspacePodAdditions, *componentInstanceStatus.WorkspacePodAdditions)
	}
	workspacePodTemplate := &workspaceDeployment.Spec.Template
	containers := map[string]corev1.Container{}
	initContainers := map[string]corev1.Container{}
	volumes := map[string]corev1.Volume{}
	pullSecrets := map[string]corev1.LocalObjectReference{}

	for _, addition := range workspacePodAdditions {
		for annotKey, annotValue := range addition.Annotations {
			workspacePodTemplate.Annotations[annotKey] = annotValue
		}

		for labelKey, labelValue := range addition.Labels {
			workspacePodTemplate.Labels[labelKey] = labelValue
		}

		for _, container := range addition.Spec.Containers {
			if _, exists := containers[container.Name]; exists {
				return errors.New("Duplicate containers in the workspace definition: " + container.Name)
			}
			containers[container.Name] = container
			workspacePodTemplate.Spec.Containers = append(workspacePodTemplate.Spec.Containers, container)
		}

		for _, container := range addition.Spec.InitContainers {
			if _, exists := initContainers[container.Name]; exists {
				return errors.New("Duplicate init conainers in the workspace definition: " + container.Name)
			}
			initContainers[container.Name] = container
			workspacePodTemplate.Spec.InitContainers = append(workspacePodTemplate.Spec.InitContainers, container)
		}

		for _, volume := range addition.Spec.Volumes {
			if _, exists := volumes[volume.Name]; exists {
				return errors.New("Duplicate volumes in the workspace definition: " + volume.Name)
			}
			volumes[volume.Name] = volume
			workspacePodTemplate.Spec.Volumes = append(workspacePodTemplate.Spec.Volumes, volume)
		}

		for _, pullSecret := range addition.Spec.ImagePullSecrets {
			if _, exists := pullSecrets[pullSecret.Name]; exists {
				continue
			}
			pullSecrets[pullSecret.Name] = pullSecret
			workspacePodTemplate.Spec.ImagePullSecrets = append(workspacePodTemplate.Spec.ImagePullSecrets, pullSecret)
		}
	}
	workspacePodTemplate.Labels[server.DEPLOYMENT_NAME_LABEL] = workspaceDeployment.Name
	for _, externalObject := range k8sObjects {
		service, isAService := externalObject.(*corev1.Service)
		if isAService {
			service.Spec.Selector[server.DEPLOYMENT_NAME_LABEL] = workspaceDeployment.Name
		}
	}
	return nil
}
