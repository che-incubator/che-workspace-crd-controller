//
// Copyright (c) 2019-2021 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

// package constants defines constant values used throughout the DevWorkspace Operator
package constants

// Labels which should be used for controller related objects
var ControllerAppLabels = func() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":    "devworkspace-controller",
		"app.kubernetes.io/part-of": "devworkspace-operator",
	}
}

// Internal constants
const (
	DefaultProjectsSourcesRoot = "/projects"

	AuthEnabled = "false"

	ServiceAccount = "devworkspace"

	SidecarDefaultMemoryLimit = "128M"
	PVCStorageSize            = "1Gi"

	// DevWorkspaceIDLabel is label key to store workspace identifier
	DevWorkspaceIDLabel = "controller.devfile.io/devworkspace_id"

	// DevWorkspaceIDLoggerKey is the key used to log workspace ID in the reconcile
	DevWorkspaceIDLoggerKey = "devworkspace_id"

	// WorkspaceEndpointNameAnnotation is the annotation key for storing an endpoint's name from the devfile representation
	DevWorkspaceEndpointNameAnnotation = "controller.devfile.io/endpoint_name"

	// DevWorkspaceNameLabel is label key to store workspace name
	DevWorkspaceNameLabel = "controller.devfile.io/devworkspace_name"

	// PullSecretLabel marks the intention that secret should be used as pull secret for devworkspaces withing namespace
	// Only secrets with 'true' value will be mount as pull secret
	// Should be assigned to secrets with type docker config types (kubernetes.io/dockercfg and kubernetes.io/dockerconfigjson)
	DevWorkspacePullSecretLabel = "controller.devfile.io/devworkspace_pullsecret"

	// DevWorkspaceCreatorLabel is the label key for storing the UID of the user who created the workspace
	DevWorkspaceCreatorLabel = "controller.devfile.io/creator"

	// DevWorkspaceStorageTypeLabel defines the strategy used for provisioning storage for the workspace.
	// If empty, the common PVC strategy is used.
	// Supported options:
	// - "common": Create one PVC per namespace, and store data for all workspaces in that namespace in that PVC
	// - "async" : Create one PVC per namespace, and create a remote server that syncs data from workspaces to the PVC.
	//             All volumeMounts used for devworkspaces are emptyDir
	DevWorkspaceStorageTypeLabel = "controller.devfile.io/storage-type"

	// DevWorkspaceRestrictedAccessAnnotation marks the intention that devworkspace access is restricted to only the creator; setting this
	// annotation will cause devworkspace start to fail if webhooks are disabled.
	// Operator also propagates it to the devworkspace-related objects to perform authorization.
	DevWorkspaceRestrictedAccessAnnotation = "controller.devfile.io/restricted-access"

	// DevWorkspaceDiscoverableServiceAnnotation marks a service in a devworkspace as created for a discoverable endpoint,
	// as opposed to a service created to support the devworkspace itself.
	DevWorkspaceDiscoverableServiceAnnotation = "controller.devfile.io/discoverable-service"

	// ControllerServiceAccountNameEnvVar stores the name of the serviceaccount used in the controller.
	ControllerServiceAccountNameEnvVar = "CONTROLLER_SERVICE_ACCOUNT_NAME"

	// DevWorkspaceStopReasonAnnotation marks the reason why the devworkspace was stopped; when a devworkspace is restarted
	// this annotation will be cleared
	DevWorkspaceStopReasonAnnotation = "controller.devfile.io/stopped-by"

	// WebhookRestartedAtAnnotation holds the the time (unixnano) of when the webhook server was forced to restart by controller
	WebhookRestartedAtAnnotation = "controller.devfile.io/restarted-at"

	// PVCCleanupPodMemoryLimit is the memory limit used for PVC clean up pods
	PVCCleanupPodMemoryLimit = "100Mi"

	// PVCCleanupPodMemoryRequest is the memory request used for PVC clean up pods
	PVCCleanupPodMemoryRequest = "32Mi"

	// PVCCleanupPodCPULimit is the cpu limit used for PVC clean up pods
	PVCCleanupPodCPULimit = "50m"

	// PVCCleanupPodCPURequest is the cpu request used for PVC clean up pods
	PVCCleanupPodCPURequest = "5m"

	// RoutingAnnotationInfix is the infix of the annotations of DevWorkspace that are passed down as annotation to the DevWorkspaceRouting objects.
	// The full annotation name is supposed to be "<routingClass>.routing.controller.devfile.io/<anything>"
	RoutingAnnotationInfix = ".routing.controller.devfile.io/"

	// Constants describing storage classes supported by the controller
	// CommonStorageClassType defines the 'common' storage policy -- one PVC is provisioned per namespace and all devworkspace storage
	// is mounted in it on subpaths according to devworkspace ID.
	CommonStorageClassType = "common"
	// AsyncStorageClassType defines the 'asynchronous' storage policy. An rsync sidecar is added to devworkspaces that uses SSH to connect
	// to a storage deployment that mounts a common PVC for the namespace.
	AsyncStorageClassType = "async"
)
