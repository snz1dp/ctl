snz1dp version     : {{ .Snz1dp.Version }}
appgateway version : {{ .Appgateway.Version }}
xeai version       : {{ .Xeai.Version }}
confserv version   : {{ .Confserv.Version }}
{{- if .Istio.Install }}
istio version      : {{ .Istio.Version }}
{{- end }}
{{- if .Redis.Install }}
redis version      : {{ .Redis.Version }}
{{- end }}
{{- if .ActiveMQ.Install }}
activemq version   : {{ .ActiveMQ.Version }}
{{- end }}
{{- if .InfluxDB.Install }}
influxdb version   : {{ .InfluxDB.Version }}
{{- end }}
{{- if .Postgres.Install }}
postgres version   : {{ .Postgres.Version }}
{{- end }}
{{- if .Openldap.Install }}
openldap version   : {{ .Openldap.Version }}
{{- end }}
{{- if .Gitlab.Install  }}
gitlab version     : {{ .Gitlab.Version }}
{{- end }}
{{- if .Jenkins.Install }}
jenkins version    : {{ .Jenkins.Version }}
{{- end }}
{{- if .Nexus.Install }}
nexus version      : {{ .Nexus.Version }}
{{- end }}
{{- if .Filerepo.Install }}
filerepo version   : {{ .Filerepo.Version }}
{{- end}}
{{- if .Jobmgr.Install }}
jobmgr version     : {{ .Jobmgr.Version }}
{{- end}}
{{- if .OldGateway.Install }}
oldgateway version : {{ .OldGateway.Version }}
{{- end}}
{{- if .Logserv.Install }}
logserv version    : {{ .Logserv.Version }}
{{- end}}
{{- if .Monitor.Install  }}
monitor version    : {{ .Monitor.Version }}
{{- end }}
