#!/bin/bash

WORKSPACE=${WORKSPACE:-$PWD}
DEPLOY_NAMESPACE=${DEPLOY_NAMESPACE:-demo}
PULL_IMAGE_SECRET=${PULL_IMAGE_SECRET:-pull-image-secret}

# TODO: 请设置配置参数
JWT_TOKEN=${JWT_TOKEN:-}
JWT_PRIVKEY=${JWT_PRIVKEY:-}
JDBC_URL=${JDBC_URL:-jdbc:postgresql://postgres:5432/demo}
JDBC_USER=${JDBC_USER:-postgres}
JDBC_PASSWORD=${JDBC_PASSWORD:-}
REDIS_SERVER=${REDIS_SERVER:-redis:6379}
REDIS_PASSWORD=${REDIS_PASSWORD:-snz1dp9527}
REDIS_DB=${REDIS_DB:-6}
CONFIG_URL=${CONFIG_URL:-http://confserv/appconfig}
XEAI_URL=${XEAI_URL:-http://xeai/xeai}
INITIAL_USERNAME=${INITIAL_USERNAME:-root}
INITIAL_PASSWORD=${INITIAL_PASSWORD:-123456}

cd $WORKSPACE
export SNZ1DP_HOME=$WORKSPACE/.snz1dp
export PATH=$PATH:$SNZ1DP_HOME/bin

DEPLOY_RELEASE_NAME=$(snz1dpctl make info)
CHART_NAME_VERSION=$(snz1dpctl make info --version)

${WORKSPACE}/.snz1dp/bin/snz1dpctl helm template \
  -n ${DEPLOY_NAMESPACE} \
  --skip-tests \
  --set "imagePullSecrets[0].name=${PULL_IMAGE_SECRET}" \
  --set "env.CONFIG_PROFILE=prod" \
  --set "env.JWT_TOKEN=${JWT_TOKEN}" \
  --set "env.JWT_PRIVKEY=" \
  --set "env.JDBC_URL=${JDBC_URL}" \
  --set "env.JDBC_USER=${JDBC_USER}" \
  --set "env.JDBC_PASSWORD=${JDBC_PASSWORD}" \
  --set "env.REDIS_SERVER=${REDIS_SERVER}" \
  --set "env.REDIS_PASSWORD=${REDIS_PASSWORD}" \
  --set "env.REDIS_DB=${REDIS_DB}" \
  --set "env.CONFIG_URL=${CONFIG_URL}" \
  --set "env.XEAI_URL=${XEAI_URL}" \
  --set "env.INITIAL_USERNAME=${INITIAL_USERNAME}" \
  --set "env.INITIAL_PASSWORD=${INITIAL_PASSWORD}" \
  ${DEPLOY_RELEASE_NAME} \
  out/${CHART_NAME_VERSION}.tgz>${WORKSPACE}/deploy.yaml

envsubst < ${WORKSPACE}/deploy.yaml | kubectl -n ${DEPLOY_NAMESPACE} apply -f -
