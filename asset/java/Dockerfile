# 引入openjdk8镜像
FROM {{ .DockerHub }}snz1dp/openjdk8-springboot-app:2.1

# 复制打包好的jar文件到/app目录
COPY target/{{ .Name }}.jar /app/app.jar
