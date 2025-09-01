FROM snz1.cn/bitnami/git:2.41.0-debian-11-r11 AS gitter

RUN cd / && \
  git clone -b master https://snz1.cn/gitrepo/dp/demo/vue-element-ui-demo.git && \
  cd vue-element-ui-demo && \
  rm -rf .git && \
  tar zcvf ../vue-element-ui-demo-1.0.0.tgz . && \
  cd ../ && \
  sha256sum vue-element-ui-demo-1.0.0.tgz>vue-element-ui-demo-1.0.0.tgz.sha256

FROM snz1.cn/dp/vueapp:2.0

ENV TZ=Asia/Shanghai

# 下载Maven
ARG MAVEN_VERSION=3.6.3
RUN curl -s \
  https://repo1.maven.org/maven2/org/apache/maven/apache-maven/${MAVEN_VERSION}/apache-maven-${MAVEN_VERSION}-bin.zip \
  -o /app/html/apache-maven-${MAVEN_VERSION}-bin.zip

# 复制Vue示例项目
COPY --from=gitter /vue-element-ui-demo-1.0.0.tgz* /app/html/

# 复制本地文件
ADD files/ /app/html/
COPY asset/version/* /app/html/
COPY nginx/default.conf /etc/nginx/conf.d/

COPY out/* /app/html/

RUN rm -rf /app/html/index.html && \
  echo -n '{"time":"'>/app/html/BUILD && \
  echo -n $(date +%FT%T.%3N%z)>>/app/html/BUILD && \
  echo -n '"}'>>/app/html/BUILD
