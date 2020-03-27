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
package handler

import (
	"context"
	"github.com/che-incubator/che-workspace-operator/pkg/controller/workspace/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var V1PodExecOptionKind = metav1.GroupVersionKind{Kind: "PodExecOptions", Group: "", Version: "v1"}

func (h *WebhookHandler) ValidateExecOnConnect(ctx context.Context, req admission.Request) admission.Response {
	p := corev1.Pod{}
	err := h.Client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, &p)

	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	_, ok := p.Labels[model.WorkspaceIDLabel]
	if !ok {
		return admission.Allowed("It's not workspace related pod")
	}

	creator, ok := p.Annotations[model.WorkspaceCreatorAnnotation]
	if !ok {
		return admission.Denied("The workspace info is missing in the workspace-related pod")
	}

	if creator != req.UserInfo.UID {
		return admission.Denied("The only workspace workspace has exec access")
	}

	return admission.Allowed("The current user and workspace are matched")
}
