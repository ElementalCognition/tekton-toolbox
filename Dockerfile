# Whitesource scanning
FROM us-central1-docker.pkg.dev/elementalcognition-app-source/platform/dev/whitesource-agent-go:0acb0d6 as whitesource
RUN mkdir ${WSS_USER_HOME}/Data
ARG WS_APIKEY
ARG WS_PROJECTVERSION
ARG WS_PROJECTNAME="tekton-toolbox"
ARG WS_PRODUCTNAME="platform"
ARG WS_GO_RESOLVEDEPENDENCIES="false"
ARG WS_GO_MODULES_RESOLVEDEPENDENCIES="true"
ARG WS_GO_COLLECTDEPENDENCIESATRUNTIME="true"

COPY go.mod go.sum /home/wss-scanner/Data/
COPY cmd /home/wss-scanner/Data/cmd
COPY pkg /home/wss-scanner/Data/pkg
COPY internal /home/wss-scanner/Data/internal
RUN cd /home/wss-scanner/Data && go mod download -x
RUN java -jar ./wss/wss-unified-agent.jar -c ./wss/wss-unified-agent.config -d /home/wss-scanner/Data
