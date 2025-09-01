
======================================================================

snz1dp version       : {{ .Snz1dp.Version }}
snz1dp namespace     : {{ .Snz1dp.Namespace }}
snz1dp web protocol  : {{ .Snz1dp.Ingress.Protocol }}
snz1dp web host      : {{ .Snz1dp.Ingress.Host }}
snz1dp web port      : {{ .Snz1dp.Ingress.Port }}

snz1dp repo protocol : {{ .Snz1dp.Ingress.Docker.Repo.Protocol }}
snz1dp repo host     : {{ .Snz1dp.Ingress.Docker.Repo.Host }}
snz1dp repo port     : {{ .Snz1dp.Ingress.Docker.Repo.Port }}

appgateway version   : {{ .Appgateway.Version }}
xeai version         : {{ .Xeai.Version }}
confserv version     : {{ .Confserv.Version }}

xeai  web root       : /xeai
confserv web root    : /appconfig

admin username       : {{ .Snz1dp.Admin.Username }}
admin password       : {{ .Snz1dp.Admin.Password }}

======================================================================

istio installed      : {{ .Istio.Install }}
istio version        : {{ .Istio.Version }}
istio namespace      : {{ .Istio.Namespace }}

======================================================================

redis installed      : {{ .Redis.Install }}
redis version        : {{ .Redis.Version }}
{{- if not .Redis.Install }}
redis host           : {{ .Redis.Host }}
redis port           : {{ .Redis.Port }}
{{- end }}
redis password       : {{ .Redis.Password }}

postgres installed   : {{ .Postgres.Install }}
postgres version     : {{ .Postgres.Version }}
{{- if not .Postgres.Install }}
postgres host        : {{ .Postgres.Host }}
postgres port        : {{ .Postgres.Port }}
{{- end }}
postgres username    : {{ .Postgres.Admin.Username }}
postgres password    : {{ .Postgres.Admin.Password }}

activemq installed   : {{ .ActiveMQ.Install }}
activemq version     : {{ .ActiveMQ.Version }}
{{- if not .ActiveMQ.Install }}
activemq host        : {{ .ActiveMQ.Host }}
{{- end }}
openwire port        : {{ .ActiveMQ.WIRE.Port }}
mqtt port            : {{ .ActiveMQ.MQTT.Port }}
stomp port           : {{ .ActiveMQ.STOMP.Port }}
amqp port            : {{ .ActiveMQ.AMQP.Port }}
console port         : {{ .ActiveMQ.Console.Port }}
console web root     : {{ .ActiveMQ.Console.Webroot }}
admin username       : {{ .ActiveMQ.Admin.Username }}
admin password       : {{ .ActiveMQ.Admin.Password }}

influxdb installed   : {{ .InfluxDB.Install }}
influxdb version     : {{ .InfluxDB.Version }}
{{- if not .InfluxDB.Install }}
influxdb host        : {{ .InfluxDB.Host }}
influxdb port        : {{ .InfluxDB.Port }}
{{- end }}
database name        : {{ .InfluxDB.DatabaseName }}
======================================================================

openldap version     : {{ .Openldap.Version }}
openldap installed   : {{ .Openldap.Install }}
{{- if not .Openldap.Install }}
openldap protocol    : {{ .Openldap.Protocol }}
openldap host        : {{ .Openldap.Host }}
openldap port        : {{ .Openldap.Port }}
{{- end }}
openldap domain      : {{ .Openldap.Domain }}
openldap admin dn    : {{ .Openldap.Admin.Username }}
admin password       : {{ .Openldap.Admin.Password }}
openldap config dn   : {{ .Openldap.Config.Username }}
config password      : {{ .Openldap.Config.Password }}

======================================================================

gitlab version       : {{ .Gitlab.Version }}
gitlab installed     : {{ .Gitlab.Install }}
{{- if not .Gitlab.Install  }}
gitlab protocol      : {{ .Gitlab.Web.Protocol }}
gitlab host          : {{ .Gitlab.Web.Host }}
gitlab port          : {{ .Gitlab.Web.Port }}
{{- end }}
gitlab web root      : {{ .Gitlab.Web.Webroot }}

jenkins version      : {{ .Jenkins.Version }}
jenkins installed    : {{ .Jenkins.Install }}
{{- if not .Jenkins.Install }}
jenkins protocol     : {{ .Jenkins.Web.Protocol }}
jenkins host         : {{ .Jenkins.Web.Host }}
jenkins port         : {{ .Jenkins.Web.Port }}
{{- end }}
jenkins web root     : {{ .Jenkins.Web.Webroot }}

nexus version        : {{ .Nexus.Version }}
nexus installed      : {{ .Nexus.Install }}
{{- if not .Nexus.Install }}
nexus web protocol   : {{ .Nexus.Web.Protocol }}
nexus web host       : {{ .Nexus.Web.Host }}
nexus web port       : {{ .Nexus.Web.Port }}
{{- end }}
{{- if not .Nexus.Install }}
nexus web root       : {{ .Nexus.Web.Webroot }}
docker repo protocol : {{ .Nexus.Repo.Protocol }}
docker repo host     : {{ .Nexus.Repo.Host }}
docker repo port     : {{ .Nexus.Repo.Port }}
{{- end }}

======================================================================

filerepo version     : {{ .Filerepo.Version }}
filerepo install     : {{ .Filerepo.Install }}
{{- if .Filerepo.Install }}
filerepo web root    : /filerepo
{{- end}}

jobmgr version       : {{ .Jobmgr.Version }}
jobmgr install       : {{ .Jobmgr.Install }}
{{- if .Jobmgr.Install }}
jobmgr web root      : /jobmgr
{{- end}}

======================================================================


{{- if .OldGateway.Install  }}
oldgateway version   : {{ .OldGateway.Version }}
{{- end }}
{{- if .Logserv.Install  }}
logserv version      : {{ .Logserv.Version }}
{{- end }}
{{- if .Monitor.Install  }}
monitor version      : {{ .Monitor.Version }}
{{- end }}
