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

package workspace

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"

	"github.com/che-incubator/che-workspace-operator/pkg/apis/workspace/v1alpha1"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/provision"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclock "k8s.io/apimachinery/pkg/util/clock"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// clock is used to set status condition timestamps.
// This variable makes it easier to test conditions.
var clock kubeclock.Clock = &kubeclock.RealClock{}

// updateWorkspaceStatus updates the current workspace's status field with conditions and phase from the passed in status.
// Parameters for result and error are returned unmodified, unless error is nil and another error is encountered while
// updating the status.
func (r *ReconcileWorkspace) updateWorkspaceStatus(workspace *v1alpha1.Workspace, logger logr.Logger, status *currentStatus, reconcileResult reconcile.Result, reconcileError error) (reconcile.Result, error) {
	workspace.Status.Phase = status.Phase
	currTransitionTime := metav1.Time{Time: clock.Now()}
	for _, conditionType := range status.Conditions {
		conditionExists := false
		for idx, condition := range workspace.Status.Conditions {
			if condition.Type == conditionType && condition.LastTransitionTime.Before(&currTransitionTime) {
				workspace.Status.Conditions[idx].LastTransitionTime = currTransitionTime
				workspace.Status.Conditions[idx].Status = corev1.ConditionTrue
				conditionExists = true
				break
			}
		}
		if !conditionExists {
			workspace.Status.Conditions = append(workspace.Status.Conditions, v1alpha1.WorkspaceCondition{
				Type:               conditionType,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: currTransitionTime,
			})
		}
	}
	for idx, condition := range workspace.Status.Conditions {
		if condition.LastTransitionTime.Before(&currTransitionTime) {
			workspace.Status.Conditions[idx].LastTransitionTime = currTransitionTime
			workspace.Status.Conditions[idx].Status = corev1.ConditionUnknown
		}
	}
	sort.SliceStable(workspace.Status.Conditions, func(i, j int) bool {
		return strings.Compare(string(workspace.Status.Conditions[i].Type), string(workspace.Status.Conditions[j].Type)) > 0
	})

	err := r.client.Status().Update(context.TODO(), workspace)
	if err != nil {
		logger.Info(fmt.Sprintf("Error updating workspace status: %s", err))
		if reconcileError == nil {
			reconcileError = err
		}
	}
	return reconcileResult, reconcileError
}

func SyncWorkspaceIdeURL(workspace *v1alpha1.Workspace, exposedEndpoints map[string]v1alpha1.ExposedEndpointList, clusterAPI provision.ClusterAPI) (ok bool, err error) {
	ideUrl := getIdeUrl(exposedEndpoints)

	if workspace.Status.IdeUrl == ideUrl {
		return true, nil
	}
	workspace.Status.IdeUrl = ideUrl
	err = clusterAPI.Client.Status().Update(context.TODO(), workspace)
	return false, err
}

func getIdeUrl(exposedEndpoints map[string]v1alpha1.ExposedEndpointList) string {
	for _, endpoints := range exposedEndpoints {
		for _, endpoint := range endpoints {
			if endpoint.Attributes[v1alpha1.TYPE_ENDPOINT_ATTRIBUTE] == "ide" {
				return endpoint.Url
			}
		}
	}
	return ""
}
