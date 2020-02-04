## Lighthouse
FROM ubuntu:18.04
LABEL MAINTAINER="beamer"

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get -yq install apt-utils tzdata wait-for-it ca-certificates && \
    apt-get autoclean -y && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /usr/bin

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

COPY ./bin/lighthouse lighthouse

RUN chmod +x ./lighthouse

ENV ELASTICSEARCH="http://localhost:9300"
ENV CHAINQUERY_DSN="lbry:lbry@tcp(localhost:3306)/chainquery"
ENV LIGHTHOUSE_API="http://localhost:50005"

EXPOSE 50005
STOPSIGNAL SIGINT
CMD ./lighthouse serve