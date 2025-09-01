package action

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// HelmChart -
type HelmChart struct {
	Readme          string            `json:"readme"`
	Version         string            `json:"version"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Type            string            `json:"type"`
	ImageRepository string            `json:"image"`
	ImageTag        string            `json:"tag"`
	Service         *StandaloneConfig `json:"service"`
}

const (
	ReadmefileName = "README.md"
	// ChartfileName is the default Chart file name.
	ChartfileName = "Chart.yaml"
	// ValuesfileName is the default values file name.
	ValuesfileName = "values.yaml"
	// TemplatesDir is the relative directory name for templates.
	TemplatesDir = "templates"
	// ChartsDir is the relative directory name for charts dependencies.
	ChartsDir = "charts"
	// TemplatesTestsDir is the relative directory name for tests.
	TemplatesTestsDir = TemplatesDir + sep + "tests"
	// IgnorefileName is the name of the Helm ignore file.
	IgnorefileName = ".helmignore"
	// IngressSsoFileName is the name of the example ingress file.
	IngressSsoFileName = TemplatesDir + sep + "ingress-sso.yaml"
	// SsoauthFileName -
	SsoauthFileName = TemplatesDir + sep + "sso-plugin.yaml"
	// IngressJwtFileName is the name of the example ingress file.
	IngressJwtFileName = TemplatesDir + sep + "ingress-jwt.yaml"
	// JwtauthFileName -
	JwtauthFileName = TemplatesDir + sep + "jwt-plugin.yaml"
	// AuthACLFileName -
	AuthACLFileName = TemplatesDir + sep + "acl-plugin.yaml"
	// IngressAnonymousFileName is the name of the example ingress file.
	IngressAnonymousFileName = TemplatesDir + sep + "ingress-open.yaml"
	// DeploymentName is the name of the example deployment file.
	DeploymentName = TemplatesDir + sep + "deployment.yaml"
	// StatefulSet is the name of the example StatefulSet file.
	StatefulSetName = TemplatesDir + sep + "satefulset.yaml"
	// ServiceName is the name of the example service file.
	ServiceName = TemplatesDir + sep + "service.yaml"
	// ServiceAccountName is the name of the example serviceaccount file.
	ServiceAccountName = TemplatesDir + sep + "serviceaccount.yaml"
	// HorizontalPodAutoscalerName is the name of the example hpa file.
	HorizontalPodAutoscalerName = TemplatesDir + sep + "hpa.yaml"
	// NotesName is the name of the example NOTES.txt file.
	NotesName = TemplatesDir + sep + "NOTES.txt"
	// HelpersName is the name of the example helpers file.
	HelpersName = TemplatesDir + sep + "_helpers.tpl"
	// TestConnectionName is the name of the example test file.
	TestConnectionName = TemplatesTestsDir + sep + "test-connection.yaml"
	// ConfigmapFileName -
	ConfigmapFileName = TemplatesDir + sep + "configmap.yaml"
	// RunFilesFileName -
	RunFilesFileName = TemplatesDir + sep + "runfiles.yaml"
	// PvcFileName -
	PvcFileName = TemplatesDir + sep + "dataclaim.yaml"
	// InitJobFielName -
	InitJobFielName = TemplatesDir + sep + "initjob.yaml"
)

const sep = string(filepath.Separator)

const defaultIngressConfig = `ingress:
  enabled: false
  sso:
    enabled: false
    hosts:
      paths: []
  jwt:
    enabled: false
    group:
      enabled: false
    hosts:
     paths: []
  anonymous:
    enabled: false
    hosts:
      paths: []

`

const defaultSsoIngressConfig = `
  sso:
    enabled: false
    hosts:
      paths: []

`

const defaultJwtIngressConfig = `
  jwt:
    enabled: false
    group:
      enabled: false
    hosts:
      paths: []

`

const defaultAnonymousIngressConfig = `
  anonymous:
    enabled: false
    hosts:
      paths: []

`

const defaultChartfile = `apiVersion: v2
name: <CHARTNAME>
description: <DESCRIPTION>

type: <APPTYPE>
version: <VERSION>
appVersion: <APPVERSION>
`

const defaultConfigmap = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ template "<CHARTNAME>.fullname" . }}-env
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
data:
{{- if .Values.env }}
{{- range $key, $value := .Values.env }}
  {{ $key }}: {{ $value | quote }}
{{- end }}
{{- end}}

`

const defaultValues = `
# 实例数量
replicaCount: 1

# 镜像配置
image:
  repository: <IMAGEREPOSITORY>
  tag: "<IMAGETAG>"
  pullPolicy: IfNotPresent

# 镜像拉取密钥
imagePullSecrets: []
# - name: snz1dp-docker-repo

nameOverride: ""
fullnameOverride: ""

# K8s服务帐号
serviceAccount:
  create: true
  annotations: {}
  name: ""

# Pod注解
podAnnotations: {}

# Pod安全上下文上下文
podSecurityContext: {}
  # fsGroup: 2000

# 安全上下文
securityContext: {}
  # capabilities:
  #   drop:
  #   - ALL
  # readOnlyRootFilesystem: true
  # runAsNonRoot: true
  # runAsUser: 1000

# 服务定义配置
service:
  type: ClusterIP
  port: <BACKENDPORT_VALUE>

# 运行环境变量配置
env:
<ENVIRONMENT>

# 扩展主机名
hosts: []

# 对外服务配置
<INGRESSCONFIG>

# 资源配额定义
resources: {}
  # limits:
  #   cpu: 100m
  #   memory: 128Mi
  # requests:
  #   cpu: 100m
  #   memory: 128Mi

# 自动缩放配置
autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 100
  targetCPUUtilizationPercentage: 80
  # targetMemoryUtilizationPercentage: 80

<DATACLAIM>
<RUNCOMMANDS>
<INITCOMMANDS>

# 其他配置
nodeSelector: {}

tolerations: []

affinity: {}
`

const defaultIgnore = `# Patterns to ignore when building packages.
# This supports shell glob matching, relative path matching, and
# negation (prefixed with !). Only one pattern per line.
.DS_Store
# Common VCS dirs
.git/
.gitignore
.bzr/
.bzrignore
.hg/
.hgignore
.svn/
# Common backup files
*.swp
*.bak
*.tmp
*.orig
*~
# Various IDEs
.project
.idea/
*.tmproj
.vscode/
*.DS_Store
`

const defaultIngressSso = `{{- if and .Values.ingress.enabled .Values.ingress.sso.enabled }}
{{- $fullName := include "<CHARTNAME>.fullname" . -}}
{{- $svcPort := .Values.service.port -}}
{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion }}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" .Capabilities.KubeVersion.GitVersion }}
apiVersion: networking.k8s.io/v1beta1
{{- else }}
apiVersion: extensions/v1beta1
{{- end }}
kind: Ingress
metadata:
  name: {{ $fullName }}-sso
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
  annotations:
    kubernetes.io/ingress.class: appgateway
    appgateway.snz1.cn/preserve-host: "true"
    appgateway.snz1.cn/strip-path: "false"
    appgateway.snz1.cn/plugins: {{ template "<CHARTNAME>.fullname" . }}-ssoauth
  {{- with .Values.ingress.sso.annotations }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.ingress.sso.tls }}
  tls:
    {{- range .Values.ingress.sso.tls }}
    - hosts:
        {{- range .hosts }}
        - {{ . | quote }}
        {{- end }}
      secretName: {{ .secretName }}
    {{- end }}
  {{- end }}
  rules:
    {{- range .Values.ingress.sso.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
        {{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
        {{- range .paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            service:
              name: {{ $fullName }}
              port:
                number: {{ $svcPort }}
        {{- end }}
        {{- else }}
        {{- range .paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            serviceName: {{ $fullName }}
            servicePort: {{ $svcPort }}
        {{- end }}
        {{- end }}
    {{- end }}
    {{- if .Values.ingress.sso.paths }}
    - http:
        paths:
        {{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
        {{- range .Values.ingress.sso.paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            service:
              name: {{ $fullName }}
              port:
                number: {{ $svcPort }}
        {{- end }}
        {{- else }}
        {{- range .Values.ingress.sso.paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            serviceName: {{ $fullName }}
            servicePort: {{ $svcPort }}
        {{- end }}
        {{- end }}
    {{- end }}
{{- end }}
`

const defaultSsoauth = `{{- if and .Values.ingress.enabled .Values.ingress.sso.enabled }}
apiVersion: appgateway.snz1.cn/v1
kind: Plugin
metadata:
  name: {{ template "<CHARTNAME>.fullname" . }}-ssoauth
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
config:
  anonymous: {{ .Values.ingress.sso.anonymous }}
plugin: ssoauth
{{- end }}
`

const defaultIngressJwt = `{{- if and .Values.ingress.enabled .Values.ingress.jwt.enabled }}
{{- $fullName := include "<CHARTNAME>.fullname" . -}}
{{- $svcPort := .Values.service.port -}}
{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion }}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" .Capabilities.KubeVersion.GitVersion }}
apiVersion: networking.k8s.io/v1beta1
{{- else }}
apiVersion: extensions/v1beta1
{{- end }}
kind: Ingress
metadata:
  name: {{ $fullName }}-jwt
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
  annotations:
    kubernetes.io/ingress.class: appgateway
    appgateway.snz1.cn/preserve-host: "true"
    appgateway.snz1.cn/strip-path: "false"
    appgateway.snz1.cn/plugins: {{ template "<CHARTNAME>.fullname" . }}-jwtauth {{- if .Values.ingress.jwt.group.enabled }} {{ template "<CHARTNAME>.fullname" . }}-authacl {{- end }}
  {{- with .Values.ingress.jwt.annotations }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.ingress.jwt.tls }}
  tls:
    {{- range .Values.ingress.jwt.tls }}
    - hosts:
        {{- range .hosts }}
        - {{ . | quote }}
        {{- end }}
      secretName: {{ .secretName }}
    {{- end }}
  {{- end }}
  rules:
    {{- range .Values.ingress.jwt.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
        {{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
        {{- range .paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            service:
              name: {{ $fullName }}
              port:
                number: {{ $svcPort }}
        {{- end }}
        {{- else }}
        {{- range .paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            serviceName: {{ $fullName }}
            servicePort: {{ $svcPort }}
        {{- end }}
        {{- end }}
    {{- end }}
    {{- if .Values.ingress.jwt.paths }}
    - http:
        paths:
        {{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
        {{- range .Values.ingress.jwt.paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            service:
              name: {{ $fullName }}
              port:
                number: {{ $svcPort }}
        {{- end }}
        {{- else }}
        {{- range .Values.ingress.jwt.paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            serviceName: {{ $fullName }}
            servicePort: {{ $svcPort }}
        {{- end }}
        {{- end }}
    {{- end }}
{{- end }}
`

const defaultJwtauth = `{{- if and .Values.ingress.enabled .Values.ingress.jwt.enabled }}
apiVersion: appgateway.snz1.cn/v1
kind: Plugin
metadata:
  name: {{ template "<CHARTNAME>.fullname" . }}-jwtauth
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
plugin: jwtauth
{{- end }}
`

const defaultAclauth = `{{- if and .Values.ingress.enabled .Values.ingress.jwt.group.enabled }}
apiVersion: appgateway.snz1.cn/v1
kind: Plugin
metadata:
  name: {{ template "<CHARTNAME>.fullname" . }}-authacl
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
config:
  {{- with .Values.ingress.jwt.group.whitelist }}
  whitelist:
  {{- with .Values.ingress.jwt.group.whitelist }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- end }}
  {{- if .Values.ingress.jwt.group.blacklist }}
  blacklist:
  {{- with .Values.ingress.jwt.group.blacklist }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- end }}
plugin: authacl
{{- end }}
`

const defaultIngressAnonymous = `{{- if and .Values.ingress.enabled .Values.ingress.anonymous.enabled }}
{{- $fullName := include "<CHARTNAME>.fullname" . -}}
{{- $svcPort := .Values.service.port -}}
{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion }}
apiVersion: networking.k8s.io/v1
{{- else if semverCompare ">=1.14-0" .Capabilities.KubeVersion.GitVersion }}
apiVersion: networking.k8s.io/v1beta1
{{- else }}
apiVersion: extensions/v1beta1
{{- end }}
kind: Ingress
metadata:
  name: {{ $fullName }}-anonymous
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
  annotations:
    kubernetes.io/ingress.class: appgateway
    appgateway.snz1.cn/preserve-host: "true"
    appgateway.snz1.cn/strip-path: "false"
  {{- with .Values.ingress.anonymous.annotations }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.ingress.anonymous.tls }}
  tls:
    {{- range .Values.ingress.anonymous.tls }}
    - hosts:
        {{- range .hosts }}
        - {{ . | quote }}
        {{- end }}
      secretName: {{ .secretName }}
    {{- end }}
  {{- end }}
  rules:
    {{- range .Values.ingress.anonymous.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
        {{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
        {{- range .paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            service:
              name: {{ $fullName }}
              port:
                number: {{ $svcPort }}
        {{- end }}
        {{- else }}
        {{- range .paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            serviceName: {{ $fullName }}
            servicePort: {{ $svcPort }}
        {{- end }}
        {{- end }}
    {{- end }}
    {{- if .Values.ingress.anonymous.paths }}
    - http:
        paths:
        {{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
        {{- range .Values.ingress.anonymous.paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            service:
              name: {{ $fullName }}
              port:
                number: {{ $svcPort }}
        {{- end }}
        {{- else }}
        {{- range .Values.ingress.anonymous.paths }}
        - path: {{ . }}
          pathType: Prefix
          backend:
            serviceName: {{ $fullName }}
            servicePort: {{ $svcPort }}
        {{- end }}
        {{- end }}
    {{- end }}
{{- end }}
`

const defaultDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "<CHARTNAME>.fullname" . }}
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
spec:
{{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
{{- end }}
  selector:
    matchLabels:
      {{- include "<CHARTNAME>.selectorLabels" . | nindent 6 }}
  template:
    metadata:
    {{- with .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
    {{- end }}
      labels:
        {{- include "<CHARTNAME>.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "<CHARTNAME>.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      {{- if .Values.init }}
      initContainers:
        - name: {{ .Chart.Name }}-init
          {{- if .Values.init.image }}
          image: {{ .Values.init.image }}
          {{- else }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          {{- end }}
          command:
            {{- toYaml .Values.init.command | nindent 12 }}
<INITJOB_MOUNTVOLUMES>
      {{- end }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          {{- with .Values.command }}
          command:
          {{- toYaml . | nindent 10 }}
          {{- end }}
          envFrom:
          - configMapRef:
              name: {{ template "<CHARTNAME>.fullname" . }}-env
          ports:
            - name: <BACKENDPORT_NAME>
              containerPort: <BACKENDPORT_VALUE>
              protocol: <BACKENDPORT_PROTOCOL>
          livenessProbe:
            <HEALTHACTION>
            <HEALTHCHECK>

          readinessProbe:
            <HEALTHACTION>
            <HEALTHCHECK>

          resources:
            {{- toYaml .Values.resources | nindent 12 }}
<MOUNTVOLUMES>
<VOLUMEDEFS>
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
`

const defaultStatefulSet = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include "<CHARTNAME>.fullname" . }}
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
spec:
  serviceName: {{ include "<CHARTNAME>.fullname" . }}-headless
{{- if not .Values.autoscaling.enabled }}
  replicas: {{ .Values.replicaCount }}
{{- else }}
  replicas: 1
{{- end }}
  selector:
    matchLabels:
      {{- include "<CHARTNAME>.selectorLabels" . | nindent 6 }}
  template:
    metadata:
    {{- with .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
    {{- end }}
      labels:
        {{- include "<CHARTNAME>.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "<CHARTNAME>.serviceAccountName" . }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      {{- if .Values.init }}
      initContainers:
        - name: {{ .Chart.Name }}-init
          {{- if .Values.init.image }}
          image: {{ .Values.init.image }}
          {{- else }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          {{- end }}
          command:
            {{- toYaml .Values.init.command | nindent 12 }}
<INITJOB_MOUNTVOLUMES>
      {{- end }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          {{- with .Values.command }}
          command:
          {{- toYaml . | nindent 10 }}
          {{- end }}
          envFrom:
          - configMapRef:
              name: {{ template "<CHARTNAME>.fullname" . }}-env
          ports:
            - name: <BACKENDPORT_NAME>
              containerPort: <BACKENDPORT_VALUE>
              protocol: <BACKENDPORT_PROTOCOL>
          livenessProbe:
            <HEALTHACTION>
            <HEALTHCHECK>

          readinessProbe:
            <HEALTHACTION>
            <HEALTHCHECK>

          resources:
            {{- toYaml .Values.resources | nindent 12 }}
<MOUNTVOLUMES>
<VOLUMEDEFS>
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
`

const defaultService = `apiVersion: v1
kind: Service
metadata:
  name: {{ include "<CHARTNAME>.fullname" . }}
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: <BACKENDPORT_NAME>
      protocol: <BACKENDPORT_PROTOCOL>
      name:  <BACKENDPORT_NAME>
  selector:
    {{- include "<CHARTNAME>.selectorLabels" . | nindent 4 }}
`

const defaultServiceAccount = `{{- if .Values.serviceAccount.create -}}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "<CHARTNAME>.serviceAccountName" . }}
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
`

const defaultHorizontalPodAutoscaler = `{{- if .Values.autoscaling.enabled }}
apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "<CHARTNAME>.fullname" . }}
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "<CHARTNAME>.fullname" . }}
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  metrics:
  {{- if .Values.autoscaling.targetCPUUtilizationPercentage }}
    - type: Resource
      resource:
        name: cpu
        targetAverageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
  {{- end }}
  {{- if .Values.autoscaling.targetMemoryUtilizationPercentage }}
    - type: Resource
      resource:
        name: memory
        targetAverageUtilization: {{ .Values.autoscaling.targetMemoryUtilizationPercentage }}
  {{- end }}
{{- end }}
`

const defaultNotes = `1. Get the application URL by running these commands:
{{- if .Values.ingress.sso.enabled }}
{{- range $host := .Values.ingress.sso.hosts }}
  {{- range .paths }}
  http{{ if $.Values.ingress.sso.tls }}s{{ end }}://{{ $host.host }}{{ . }}
  {{- end }}
{{- end }}
{{- else if contains "NodePort" .Values.service.type }}
  export NODE_PORT=$(kubectl get --namespace {{ .Release.Namespace }} -o jsonpath="{.spec.ports[0].nodePort}" services {{ include "<CHARTNAME>.fullname" . }})
  export NODE_IP=$(kubectl get nodes --namespace {{ .Release.Namespace }} -o jsonpath="{.items[0].status.addresses[0].address}")
  echo http://$NODE_IP:$NODE_PORT
{{- else if contains "LoadBalancer" .Values.service.type }}
     NOTE: It may take a few minutes for the LoadBalancer IP to be available.
           You can watch the status of by running 'kubectl get --namespace {{ .Release.Namespace }} svc -w {{ include "<CHARTNAME>.fullname" . }}'
  export SERVICE_IP=$(kubectl get svc --namespace {{ .Release.Namespace }} {{ include "<CHARTNAME>.fullname" . }} --template "{{"{{ range (index .status.loadBalancer.ingress 0) }}{{.}}{{ end }}"}}")
  echo http://$SERVICE_IP:{{ .Values.service.port }}
{{- else if contains "ClusterIP" .Values.service.type }}
  export POD_NAME=$(kubectl get pods --namespace {{ .Release.Namespace }} -l "app.kubernetes.io/name={{ include "<CHARTNAME>.name" . }},app.kubernetes.io/instance={{ .Release.Name }}" -o jsonpath="{.items[0].metadata.name}")
  echo "Visit http://127.0.0.1:8080 to use your application"
  kubectl --namespace {{ .Release.Namespace }} port-forward $POD_NAME 8080:80
{{- end }}
`

const defaultHelpers = `{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "<CHARTNAME>.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "<CHARTNAME>.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "<CHARTNAME>.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "<CHARTNAME>.labels" -}}
helm.sh/chart: {{ include "<CHARTNAME>.chart" . }}
{{ include "<CHARTNAME>.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "<CHARTNAME>.selectorLabels" -}}
app.kubernetes.io/name: {{ include "<CHARTNAME>.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "<CHARTNAME>.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "<CHARTNAME>.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
`

const defaultTestConnection = `apiVersion: v1
kind: Pod
metadata:
  name: "{{ include "<CHARTNAME>.fullname" . }}-test-connection"
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": test-success
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args: ['{{ include "<CHARTNAME>.fullname" . }}:{{ .Values.service.port }}']
  restartPolicy: Never
`

const defaultRunFiles = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ template "<CHARTNAME>.fullname" .  }}-runfiles
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
<BINARYFILES>
data:
<CONFIGFILES>
`

const defaultPVC = `{{- if .Values.persistence.create -}}
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: {{ .Values.persistence.claimName }}
  labels:
    {{- include "<CHARTNAME>.labels" . | nindent 4 }}
spec:
  storageClassName: {{ .Values.persistence.storageClassName }}
  accessModes:
    - {{ .Values.persistence.accessMode }}
  resources:
    requests:
      storage: {{ .Values.persistence.storageSize }}
{{- end -}}
`

// const defaultInitJob = `apiVersion: batch/v1
// kind: Job
// metadata:
//   name: {{ include "<CHARTNAME>.fullname" . }}-initializer
//   labels:
//     {{- include "<CHARTNAME>.labels" . | nindent 4 }}
// spec:
//   template:
//     metadata:
//       name: {{ .Chart.Name }}
//       annotations:
//         sidecar.istio.io/inject: "false"
//     spec:
//     {{- with .Values.imagePullSecrets }}
//       imagePullSecrets:
//         {{- toYaml . | nindent 8 }}
//     {{- end }}
//     {{- with .Values.extrasHosts }}
//       hostAliases:
//         {{- toYaml . | nindent 8 }}
//     {{- end }}
//       restartPolicy: OnFailure
//       containers:
//         - name: {{ .Chart.Name }}-initializer
//           image: "{{ .Values.init.image }}"
//           imagePullPolicy: {{ .Values.image.pullPolicy }}
//           envFrom:
//           - configMapRef:
//               name: {{ template "<CHARTNAME>.fullname" . }}-env
//           command:
//           {{- with .Values.init.command }}
//             {{- toYaml . | nindent 10 }}
//           {{- end }}
// <INITJOB_MOUNTVOLUMES>
// <INITJOB_VOLUMEDEFS>
// `

func (h *HelmChart) getMountVolumes() (val string) {
	buf := bytes.NewBuffer(nil)
	runfiles := map[string]bool{}

	if h.Service != nil {
		for k := range h.Service.RunFiles {
			runfiles[k] = true
		}
	}

	if h.Service != nil && len(h.Service.Volumes) > 0 {
		fmt.Fprintln(buf, "          volumeMounts:")
		for _, v := range h.Service.Volumes {
			fst := strings.Index(v, ":")
			if fst < 0 {
				continue
			}

			firstName := v[0:fst]
			if firstName == "" {
				continue
			}

			lastName := v[fst+1:]
			if lastName == "" {
				continue
			}

			fst = strings.Index(lastName, ":")
			if fst >= 0 {
				lastName = lastName[0:fst]
			}

			if lastName == "" {
				continue
			}

			fmt.Fprintln(buf, "            - mountPath: "+lastName)
			if runfiles[firstName] {
				fmt.Fprintln(buf, "              name: runfiles")
			} else if h.isDeployment() {
				fmt.Fprintln(buf, "              name: dataclaim")
			} else {
				fmt.Fprintln(buf, "              name: {{ .Values.persistence.claimName }}")
			}
			fmt.Fprintln(buf, "              subPath: "+firstName)

		}
	}

	val = buf.String()

	return
}

func (h *HelmChart) getInitJobMountVolumes() (val string) {
	buf := bytes.NewBuffer(nil)
	runfiles := map[string]bool{}

	if h.Service != nil {
		for k := range h.Service.RunFiles {
			runfiles[k] = true
		}
	}

	if h.Service != nil && len(h.Service.Volumes) > 0 {
		fmt.Fprintln(buf, "          volumeMounts:")
		for _, v := range h.Service.Volumes {
			fst := strings.Index(v, ":")
			if fst < 0 {
				continue
			}

			firstName := v[0:fst]
			if firstName == "" {
				continue
			}

			lastName := v[fst+1:]
			if lastName == "" {
				continue
			}

			fst = strings.Index(lastName, ":")
			if fst >= 0 {
				lastName = lastName[0:fst]
			}

			if lastName == "" {
				continue
			}

			fmt.Fprintln(buf, "            - mountPath: "+lastName)
			if runfiles[firstName] {
				fmt.Fprintln(buf, "              name: runfiles")
				fmt.Fprintln(buf, "              subPath: "+firstName)
			} else {
				fmt.Fprintln(buf, "              name: dataclaim")
			}
		}
	}

	val = buf.String()

	return
}

func (h *HelmChart) isDeployment() (val bool) {
	val = true
	if h.Service.Kind == "StatefulSet" {
		val = false
	}
	return val
}

func (h *HelmChart) hasDataClaim() (dataclaim bool) {
	if h.Service == nil || (len(h.Service.RunFiles) == 0 && len(h.Service.Volumes) == 0) {
		return
	}
	runfiles := map[string]bool{}
	if len(h.Service.RunFiles) > 0 {
		for k := range h.Service.RunFiles {
			runfiles[k] = true
		}
	}
	if len(h.Service.Volumes) > 0 {
		for _, v := range h.Service.Volumes {
			fst := strings.Index(v, ":")
			if fst < 0 {
				continue
			}

			firstName := v[0:fst]
			if firstName == "" {
				continue
			}

			lastName := v[fst+1:]
			if lastName == "" {
				continue
			}

			fst = strings.Index(lastName, ":")
			if fst >= 0 {
				lastName = lastName[fst+1:]
			}

			if lastName == "" {
				continue
			}

			if !runfiles[firstName] {
				dataclaim = true
				break
			}
		}
	}
	return
}

func (h *HelmChart) getDataClaim() (val string) {
	if h.hasDataClaim() {
		val = `# 持久化卷定义,请根据需要修改persistence.storageClassName值
persistence:
  create: true
  claimName: <CHARTNAME>-data
  storageClassName: hostpath
  accessMode: ReadWriteOnce
  storageSize: 5Gi
`
		val = strings.ReplaceAll(val, "<CHARTNAME>", h.Name)
	}
	return
}

func (h *HelmChart) getRunCommand() (val string) {
	if h.Service == nil || len(h.Service.Cmd) == 0 {
		return
	}
	buf := bytes.NewBuffer(nil)
	fmt.Fprintln(buf, "# 定义容器运行命令")
	fmt.Fprintln(buf, "command:")
	for _, v := range h.Service.InitCmd {
		v = strings.ReplaceAll(v, "{{", "[[{")
		v = strings.ReplaceAll(v, "}}", "}]]")
		fmt.Fprintln(buf, "- "+v)
	}
	val = buf.String()
	return
}

func (h *HelmChart) getInitCommand() (val string) {
	if h.Service == nil {
		return
	}
	var (
		initImage    string
		initCommands []string
	)
	if len(h.Service.InitCmd) > 0 {
		initCommands = h.Service.InitCmd
		initImage = fmt.Sprintf("%s:%s", h.ImageRepository, h.ImageTag)
	} else {
		if h.Service.InitJob == nil || len(h.Service.InitJob.Command) == 0 {
			return
		}
		initCommands = h.Service.InitJob.Command
		if h.Service.InitJob.DockerImage == "" {
			initImage = fmt.Sprintf("%s:%s", h.ImageRepository, h.ImageTag)
		} else {
			initImage = h.Service.InitJob.DockerImage
		}
	}
	buf := bytes.NewBuffer(nil)
	fmt.Fprintln(buf, "# 初始化脚本定义")
	fmt.Fprintln(buf, "init:")
	fmt.Fprintf(buf, "  image: %s\n", initImage)
	fmt.Fprintln(buf, "  command:")
	for _, v := range initCommands {
		v = strings.ReplaceAll(v, "{{", "[[{")
		v = strings.ReplaceAll(v, "}}", "}]]")
		fmt.Fprintln(buf, "  - "+v)
	}
	val = buf.String()
	return
}

func (h *HelmChart) getEnvironment() (val string) {
	if h.Service == nil || len(h.Service.Envs) == 0 {
		return
	}

	buf := bytes.NewBuffer(nil)

	for _, v := range h.Service.Envs {
		fst := strings.Index(v, "=")
		if fst < 0 {
			continue
		}
		tmpv := v[fst+1:]
		tmpv = strings.ReplaceAll(tmpv, "{{", "[[{")
		tmpv = strings.ReplaceAll(tmpv, "}}", "}]]")
		fmt.Fprintln(buf, "  "+v[0:fst]+": "+tmpv)
	}

	val = buf.String()

	return
}

func (h *HelmChart) getVolumeDefinitions() (val string) {
	if h.Service == nil || (len(h.Service.RunFiles) == 0 && len(h.Service.Volumes) == 0) {
		return
	}

	buf := bytes.NewBuffer(nil)

	fmt.Fprintln(buf, "      volumes:")

	if len(h.Service.RunFiles) > 0 {
		fmt.Fprintln(buf, "        - name: runfiles")
		fmt.Fprintln(buf, "          configMap:")
		fmt.Fprintln(buf, "            name: {{ template \""+h.Name+".fullname\" . }}-runfiles")
	}

	if h.isDeployment() && h.hasDataClaim() {
		fmt.Fprintln(buf, "        - name: dataclaim")
		fmt.Fprintln(buf, "          persistentVolumeClaim:")
		fmt.Fprintln(buf, "            claimName: {{ .Values.persistence.claimName }}")
	} else if !h.isDeployment() {
		fmt.Fprintln(buf, "  volumeClaimTemplates:")
		fmt.Fprintln(buf, "    - kind: PersistentVolumeClaim")
		fmt.Fprintln(buf, "      apiVersion: v1")
		fmt.Fprintln(buf, "      metadata:")
		fmt.Fprintln(buf, "        name: {{ .Values.persistence.claimName }}")
		fmt.Fprintln(buf, "      spec:")
		fmt.Fprintln(buf, "        accessModes:")
		fmt.Fprintln(buf, "          - {{ .Values.persistence.accessMode }}")
		fmt.Fprintln(buf, "        resources:")
		fmt.Fprintln(buf, "          requests:")
		fmt.Fprintln(buf, "            storage: {{ .Values.persistence.storageSize }}")
		fmt.Fprintln(buf, "        storageClassName: {{ .Values.persistence.storageClassName }}")
	}

	val = buf.String()
	return
}

func (h *HelmChart) getInitJobVolumeDefinitions() (val string) {
	if h.Service == nil || (len(h.Service.RunFiles) == 0 && len(h.Service.Volumes) == 0) {
		return
	}

	buf := bytes.NewBuffer(nil)

	fmt.Fprintln(buf, "      volumes:")

	if len(h.Service.RunFiles) > 0 {
		fmt.Fprintln(buf, "        - name: runfiles")
		fmt.Fprintln(buf, "          configMap:")
		fmt.Fprintln(buf, "            name: {{ template \""+h.Name+".fullname\" . }}-runfiles")
	}

	if h.isDeployment() && h.hasDataClaim() {
		fmt.Fprintln(buf, "        - name: dataclaim")
		fmt.Fprintln(buf, "          persistentVolumeClaim:")
		fmt.Fprintln(buf, "            claimName: {{ .Values.persistence.claimName }}")
	}

	val = buf.String()
	return
}

func (h *HelmChart) getRunFiles() (val string) {
	buf := bytes.NewBuffer(nil)

	for k, v := range h.Service.RunFiles {

		if strings.HasPrefix(v, "base64://") {
			continue
		}

		fmt.Fprintln(buf, "  "+k+": |")

		v = strings.ReplaceAll(v, "\n", "\n    ")
		v = strings.ReplaceAll(v, "{{", "[[{")
		v = strings.ReplaceAll(v, "}}", "}]]")

		fmt.Fprintln(buf, "    "+v)
		fmt.Fprintln(buf, "")
	}

	val = buf.String()

	return
}

func (h *HelmChart) getBinaryFiles() (val string) {
	buf := bytes.NewBuffer(nil)

	fmt.Fprintln(buf, "binaryData:")

	for k, v := range h.Service.RunFiles {

		if !strings.HasPrefix(v, "base64://") {
			continue
		}

		fmt.Fprintln(buf, "  "+k+": |")

		v = v[9:]
		v = strings.ReplaceAll(v, "\n", "\n    ")
		v = strings.ReplaceAll(v, "{{", "[[{")
		v = strings.ReplaceAll(v, "}}", "}]]")

		fmt.Fprintln(buf, "    "+v)
		fmt.Fprintln(buf, "")
	}

	val = buf.String()

	return
}

func (h *HelmChart) hasBinaryFiles() bool {
	for _, v := range h.Service.RunFiles {
		if strings.HasPrefix(v, "base64://") {
			return true
		}
	}
	return false
}

func (h *HelmChart) getIngressConfig() (val string) {
	buf := bytes.NewBuffer(nil)

	if h.Service == nil || len(h.Service.Ingress) == 0 {
		fmt.Fprint(buf, defaultIngressConfig)
		val = buf.String()
		return
	}

	fmt.Fprintln(buf, "ingress:")
	fmt.Fprintln(buf, "  enabled: false")

	ingressConfig := h.Service.Ingress[0]

	if len(ingressConfig.SSOAuth) == 0 {
		fmt.Fprint(buf, defaultSsoIngressConfig)
	} else {
		fmt.Fprintln(buf, "  # 定义用户身份验证接口")
		fmt.Fprintln(buf, "  sso:")
		fmt.Fprintln(buf, "    enabled: true")
		fmt.Fprintln(buf, "    webroot: "+ingressConfig.WebRoot)
		if ingressConfig.MiscAuth {
			fmt.Fprintln(buf, "    anonymous: true")
		} else {
			fmt.Fprintln(buf, "    anonymous: false")
		}
		if len(ingressConfig.Host) == 0 {
			fmt.Fprintln(buf, "    hosts: []")
		} else {
			fmt.Fprintln(buf, "    hosts:")
			for _, v := range ingressConfig.Host {
				fmt.Fprintln(buf, "    -"+v)
			}
		}
		fmt.Fprintln(buf, "    paths:")
		for _, v := range ingressConfig.SSOAuth {
			fmt.Fprintln(buf, "    - "+v)
		}
	}

	if len(ingressConfig.JWTAuth) == 0 {
		fmt.Fprint(buf, defaultJwtIngressConfig)
	} else {
		fmt.Fprintln(buf, "  # 定义程序身份验证接口")
		fmt.Fprintln(buf, "  jwt:")
		fmt.Fprintln(buf, "    enabled: true")
		fmt.Fprintln(buf, "    webroot: "+ingressConfig.WebRoot)
		if len(ingressConfig.Host) == 0 {
			fmt.Fprintln(buf, "    hosts: []")
		} else {
			fmt.Fprintln(buf, "    hosts:")
			for _, v := range ingressConfig.Host {
				fmt.Fprintln(buf, "    -"+v)
			}
		}
		fmt.Fprintln(buf, "    paths:")
		for _, v := range ingressConfig.JWTAuth {
			fmt.Fprintln(buf, "    - "+v)
		}

		if len(ingressConfig.Whitelist) > 0 {
			fmt.Fprintln(buf, "    group:")
			fmt.Fprintln(buf, "      enabled: true")
			fmt.Fprintln(buf, "      whitelist:")
			for _, k := range ingressConfig.Whitelist {
				fmt.Fprintln(buf, "      - "+k)
			}
		} else if len(ingressConfig.Blacklist) > 0 {
			fmt.Fprintln(buf, "    group:")
			fmt.Fprintln(buf, "      enabled: true")
			fmt.Fprintln(buf, "      blacklist:")
			for _, k := range ingressConfig.Blacklist {
				fmt.Fprintln(buf, "      - "+k)
			}
		} else {
			fmt.Fprintln(buf, "    group:")
			fmt.Fprintln(buf, "      enabled: false")
		}
	}

	if len(ingressConfig.Anonymous) == 0 {
		fmt.Fprint(buf, defaultAnonymousIngressConfig)
	} else {
		fmt.Fprintln(buf, "  # 定义匿名开放接口")
		fmt.Fprintln(buf, "  anonymous:")
		fmt.Fprintln(buf, "    enabled: false")
		fmt.Fprintln(buf, "    webroot: "+ingressConfig.WebRoot)
		if len(ingressConfig.Host) == 0 {
			fmt.Fprintln(buf, "    hosts: []")
		} else {
			fmt.Fprintln(buf, "    hosts:")
			for _, v := range ingressConfig.Host {
				fmt.Fprintln(buf, "    -"+v)
			}
		}
		fmt.Fprintln(buf, "    paths:")
		for _, v := range ingressConfig.Anonymous {
			fmt.Fprintln(buf, "    - "+v)
		}
	}

	val = buf.String()

	return
}

func (h *HelmChart) getBackendProtocol() (val string) {
	val = "tcp"
	if h.Service == nil || len(h.Service.Ingress) == 0 {
		if h.Service == nil || len(h.Service.Ports) == 0 {
			return
		}
		portFirst := h.Service.Ports[0]
		lstPos := strings.LastIndex(portFirst, "/")
		if lstPos < 0 {
			return
		}
		val = portFirst[lstPos+1:]
	} else {
		ingressConfig := h.Service.Ingress[0]
		if ingressConfig.Protocol != "" && ingressConfig.Protocol != "http" && ingressConfig.Protocol != "https" {
			val = ingressConfig.Protocol
		}
	}
	return
}

func (h *HelmChart) getBackendPort() (val string) {
	val = "80"

	if h.Service == nil || len(h.Service.Ingress) == 0 {
		if h.Service == nil || len(h.Service.Ports) == 0 {
			return
		}

		portFirst := h.Service.Ports[0]
		lstPos := strings.LastIndex(portFirst, ":")
		if lstPos < 0 {
			lstPos = strings.Index(portFirst, "/")
			if lstPos < 0 {
				val = portFirst
				return
			}
			val = portFirst[0:lstPos]
			return
		}

		val = portFirst[lstPos+1:]
		lstPos = strings.Index(val, "/")
		if lstPos < 0 {
			return
		}
		val = val[0:lstPos]
	} else {
		ingressConfig := h.Service.Ingress[0]
		val = fmt.Sprintf("%d", ingressConfig.BackendPort)
	}
	return
}

func (h *HelmChart) getHealthURL() (val string) {

	val = "/health"

	if h.Service == nil || h.Service.HealthCheck == nil || h.Service.HealthCheck.URL == "" {
		return
	}

	val = h.Service.HealthCheck.URL

	return
}

func (h *HelmChart) getHealthAction() (val string) {
	// httpGet:
	// path: <HEALTHURL>
	// port: http
	buf := bytes.NewBuffer(nil)
	if len(h.Service.HealthCheck.Test) <= 1 {
		backendPortName := fmt.Sprintf("%s%s", h.getBackendProtocol(), h.getBackendPort())
		fmt.Fprintln(buf, "httpGet:")
		fmt.Fprintln(buf, "              path: "+h.getHealthURL())
		fmt.Fprintf(buf, "              port: %s\n", backendPortName)
	} else {
		fmt.Fprintln(buf, "exec:")
		fmt.Fprintln(buf, "              command:")
		for _, v := range h.Service.HealthCheck.Test[1:] {
			fmt.Fprintln(buf, "              - "+v)
		}
	}
	val = buf.String()
	return
}

// CreateChart creates a new chart in a directory.
func CreateChart(dir string, chart *HelmChart) (string, error) {
	cdir, err := filepath.Abs(dir)
	if err != nil {
		return cdir, err
	}

	if fi, err := os.Stat(cdir); err == nil {
		if fi.IsDir() {
			return cdir, errors.Errorf("%s already exists", cdir)
		}
		return cdir, errors.Errorf("%s already exists and is not a directory", cdir)
	}

	files := []struct {
		path    string
		content []byte
	}{
		{
			// Chart.yaml
			path:    filepath.Join(cdir, ChartfileName),
			content: transformByHelmChart(defaultChartfile, chart),
		},
		{
			// values.yaml
			path:    filepath.Join(cdir, ValuesfileName),
			content: transformBackend(transformIngressConfig(transformByHelmChart(defaultValues, chart), chart), chart),
		},
		{
			// .helmignore
			path:    filepath.Join(cdir, IgnorefileName),
			content: []byte(defaultIgnore),
		},
		{
			// configmap.yaml
			path:    filepath.Join(cdir, ConfigmapFileName),
			content: transformByHelmChart(defaultConfigmap, chart),
		},
		{
			// ingress-sso.yaml
			path:    filepath.Join(cdir, IngressSsoFileName),
			content: transformByHelmChart(defaultIngressSso, chart),
		},
		{
			// ssoauth.yaml
			path:    filepath.Join(cdir, SsoauthFileName),
			content: transformByHelmChart(defaultSsoauth, chart),
		},
		{
			// ingress-jwt.yaml
			path:    filepath.Join(cdir, IngressJwtFileName),
			content: transformByHelmChart(defaultIngressJwt, chart),
		},
		{
			// jwtauth.yaml
			path:    filepath.Join(cdir, JwtauthFileName),
			content: transformByHelmChart(defaultJwtauth, chart),
		},
		{
			// authacl.yaml
			path:    filepath.Join(cdir, AuthACLFileName),
			content: transformByHelmChart(defaultAclauth, chart),
		},
		{
			// ingress-anonymous.yaml
			path:    filepath.Join(cdir, IngressAnonymousFileName),
			content: transformByHelmChart(defaultIngressAnonymous, chart),
		},
		{
			// service.yaml
			path:    filepath.Join(cdir, ServiceName),
			content: transformBackend(transformByHelmChart(defaultService, chart), chart),
		},
		{
			// serviceaccount.yaml
			path:    filepath.Join(cdir, ServiceAccountName),
			content: transformByHelmChart(defaultServiceAccount, chart),
		},
		{
			// hpa.yaml
			path:    filepath.Join(cdir, HorizontalPodAutoscalerName),
			content: transformByHelmChart(defaultHorizontalPodAutoscaler, chart),
		},
		{
			// NOTES.txt
			path:    filepath.Join(cdir, NotesName),
			content: transformByHelmChart(defaultNotes, chart),
		},
		{
			// _helpers.tpl
			path:    filepath.Join(cdir, HelpersName),
			content: transformByHelmChart(defaultHelpers, chart),
		},
		{
			// test-connection.yaml
			path:    filepath.Join(cdir, TestConnectionName),
			content: transformByHelmChart(defaultTestConnection, chart),
		},
	}

	if chart.Service != nil && len(chart.Service.RunFiles) > 0 {
		// runfiles.yaml
		files = append(files, struct {
			path    string
			content []byte
		}{
			path:    filepath.Join(cdir, RunFilesFileName),
			content: transformRunFiles(transformByHelmChart(defaultRunFiles, chart), chart),
		})
	}

	if chart.isDeployment() {
		files = append(files, struct {
			path    string
			content []byte
		}{
			// deployment.yaml
			path:    filepath.Join(cdir, DeploymentName),
			content: transformBackend(transformByHelmChart(defaultDeployment, chart), chart),
		})

		if chart.hasDataClaim() {
			// dataclaim.yaml
			files = append(files, struct {
				path    string
				content []byte
			}{
				path:    filepath.Join(cdir, PvcFileName),
				content: transformByHelmChart(defaultPVC, chart),
			})
		}

	} else {
		files = append(files, struct {
			path    string
			content []byte
		}{
			// statefulset.yaml
			path:    filepath.Join(cdir, StatefulSetName),
			content: transformBackend(transformByHelmChart(defaultStatefulSet, chart), chart),
		})
	}

	/*
		if chart.Service != nil && (len(chart.Service.InitCmd) > 0 || (chart.Service.InitJob != nil && len(chart.Service.InitJob.Command) > 0)) {
			// initjob.yaml
			files = append(files, struct {
				path    string
				content []byte
			}{
				path:    filepath.Join(cdir, InitJobFielName),
				content: transformBackend(transformByHelmChart(defaultInitJob, chart), chart),
			})
		}
	*/

	if chart.Readme != "" {
		files = append(files, struct {
			path    string
			content []byte
		}{
			// README.md
			path:    filepath.Join(cdir, ReadmefileName),
			content: transformByHelmChart(chart.Readme, chart),
		})
	}

	for _, file := range files {
		if _, err := os.Stat(file.path); err == nil {
			// File exists and is okay. Skip it.
			continue
		}
		if err := writeFile(file.path, file.content); err != nil {
			return cdir, err
		}
	}

	// Need to add the ChartsDir explicitly as it does not contain any file OOTB
	if err := os.MkdirAll(filepath.Join(cdir, ChartsDir), 0755); err != nil {
		return cdir, err
	}
	return cdir, nil
}

func transformByHelmChart(src string, chart *HelmChart) []byte {
	src = strings.ReplaceAll(src, "<CHARTNAME>", chart.Name)
	src = strings.ReplaceAll(src, "<APPTYPE>", chart.Type)
	src = strings.ReplaceAll(src, "<VERSION>", chart.Version)
	src = strings.ReplaceAll(src, "<APPVERSION>", chart.Version)
	src = strings.ReplaceAll(src, "<DESCRIPTION>", chart.Description)
	src = strings.ReplaceAll(src, "<IMAGEREPOSITORY>", chart.ImageRepository)
	src = strings.ReplaceAll(src, "<IMAGETAG>", chart.ImageTag)
	return []byte(src)
}

func transformIngressConfig(src []byte, chart *HelmChart) []byte {
	val := strings.ReplaceAll(string(src), "<INGRESSCONFIG>", chart.getIngressConfig())
	val = strings.ReplaceAll(val, "<DATACLAIM>", chart.getDataClaim())
	val = strings.ReplaceAll(val, "<ENVIRONMENT>", chart.getEnvironment())
	val = strings.ReplaceAll(val, "<INITCOMMANDS>", chart.getInitCommand())
	val = strings.ReplaceAll(val, "<RUNCOMMANDS>", chart.getRunCommand())
	return []byte(val)
}

func transformRunFiles(src []byte, chart *HelmChart) []byte {
	var fileContext = strings.ReplaceAll(string(src), "<CONFIGFILES>", chart.getRunFiles())
	if chart.hasBinaryFiles() {
		fileContext = strings.ReplaceAll((string)(fileContext), "<BINARYFILES>", chart.getBinaryFiles())
	} else {
		fileContext = strings.ReplaceAll((string)(fileContext), "<BINARYFILES>", "")
	}
	return ([]byte)(fileContext)
}

const healthParam = `initialDelaySeconds: %d
            periodSeconds: %d
            timeoutSeconds: %d
            failureThreshold: %d
            successThreshold: 1`

func transformBackend(src []byte, chart *HelmChart) []byte {
	var (
		val                       string
		healthCheck               string
		backendPortName           string
		interval, timeout, period time.Duration
	)

	backendPortName = fmt.Sprintf("%s%s", chart.getBackendProtocol(), chart.getBackendPort())

	val = strings.ReplaceAll(string(src), "<BACKENDPORT_NAME>", backendPortName)
	val = strings.ReplaceAll(val, "<BACKENDPORT_PROTOCOL>", strings.ToUpper(chart.getBackendProtocol()))
	val = strings.ReplaceAll(val, "<BACKENDPORT_VALUE>", chart.getBackendPort())

	val = strings.ReplaceAll(val, "<HEALTHURL>", chart.getHealthURL())
	val = strings.ReplaceAll(val, "<HEALTHACTION>", chart.getHealthAction())
	val = strings.ReplaceAll(val, "<MOUNTVOLUMES>", chart.getMountVolumes())
	val = strings.ReplaceAll(val, "<VOLUMEDEFS>", chart.getVolumeDefinitions())

	val = strings.ReplaceAll(val, "<INITJOB_MOUNTVOLUMES>", chart.getInitJobMountVolumes())
	val = strings.ReplaceAll(val, "<INITJOB_VOLUMEDEFS>", chart.getInitJobVolumeDefinitions())

	if chart.Service != nil && chart.Service.HealthCheck != nil {
		interval, _ = time.ParseDuration(chart.Service.HealthCheck.Interval)
		timeout, _ = time.ParseDuration(chart.Service.HealthCheck.Timeout)
		period, _ = time.ParseDuration(chart.Service.HealthCheck.StartPeriod)
		healthCheck = fmt.Sprintf(healthParam,
			int64(period.Seconds()),
			int64(interval.Seconds()),
			int64(timeout.Seconds()),
			chart.Service.HealthCheck.Retries)
	}
	val = strings.ReplaceAll(val, "<HEALTHCHECK>", healthCheck)

	return []byte(val)
}

func writeFile(name string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	return os.WriteFile(name, content, 0644)
}
