FROM ubuntu:bionic

ARG version

RUN apt-get update && apt-get install -y --no-install-recommends \
  curl \
  ca-certificates

RUN curl -o  /tmp/odyssey.tar.gz -L \
  https://github.com/yandex/odyssey/releases/download/$version/odyssey.linux-amd64.b7bcb86.tar.gz && \
  cd /tmp && \
  tar xvfz /tmp/odyssey.tar.gz && \
  cp odyssey /usr/bin && \
  mkdir -p /etc/odyssey && \
  rm -rf /tmp/odyssey*

CMD ["sh", "-c", "/usr/bin/odyssey /etc/odyssey/odyssey.conf"]
