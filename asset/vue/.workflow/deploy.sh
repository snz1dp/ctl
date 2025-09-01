#!/bin/bash

WORKSPACE=${WORKSPACE:-$PWD}
DEPLOY_NAMESPACE=${DEPLOY_NAMESPACE:-demo}
PULL_IMAGE_SECRET=${PULL_IMAGE_SECRET:-pull-image-secret}

cd $WORKSPACE
export SNZ1DP_HOME=$WORKSPACE/.snz1dp
export PATH=$PATH:$SNZ1DP_HOME/bin

DEPLOY_RELEASE_NAME=$(snz1dpctl make info)
CHART_NAME_VERSION=$(snz1dpctl make info --version)

${WORKSPACE}/.snz1dp/bin/snz1dpctl helm template \
  -n ${DEPLOY_NAMESPACE} \
  --skip-tests \
  --set "imagePullSecrets[0].name=${PULL_IMAGE_SECRET}" \
  ${DEPLOY_RELEASE_NAME} \
  out/${CHART_NAME_VERSION}.tgz>${WORKSPACE}/deploy.yaml

envsubst < ${WORKSPACE}/deploy.yaml | kubectl -n ${DEPLOY_NAMESPACE} apply -f -
