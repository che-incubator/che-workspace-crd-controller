#!/bin/bash
#
# Copyright (c) 2012-2020 Red Hat, Inc.
# This program and the accompanying materials are made
# available under the terms of the Eclipse Public License 2.0
# which is available at https://www.eclipse.org/legal/epl-2.0/
#
# SPDX-License-Identifier: EPL-2.0
#
set -ex

# ENV used by PROW ci
export CI="openshift"
export ARTIFACTS_DIR="/tmp/artifacts"

# Pod created by openshift ci don't have user. Using this envs should avoid errors with git user.
export GIT_COMMITTER_NAME="CI BOT"
export GIT_COMMITTER_EMAIL="ci_bot@notused.com"

# Check if operator-sdk is installed and if not install operator sdk in $GOPATH/bin dir
if ! hash operator-sdk 2>/dev/null; then
    mkdir -p $GOPATH/bin
    export PATH="$PATH:$(pwd):$GOPATH/bin"
    OPERATOR_SDK_VERSION=v0.17.0

    curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu

    chmod +x operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu && \
        cp operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu $GOPATH/bin/operator-sdk && \
        rm operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu
fi

# For some reason go on PROW force usage vendor folder
# This workaround is here until we don't figure out cause
go mod tidy
go mod vendor

make test_e2e
