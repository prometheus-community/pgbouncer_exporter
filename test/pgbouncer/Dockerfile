FROM alpine:3.7

ARG pgbversion

ENV PGBOUNCER_INI_PATH /vol/pgbouncer.ini

RUN apk --update add autoconf autoconf-doc automake c-ares c-ares-dev curl gcc libc-dev libevent libevent-dev libtool make man libressl-dev pkgconfig

RUN \
  # Download
  curl -o  /tmp/pgbouncer-$pgbversion.tar.gz -L https://pgbouncer.github.io/downloads/files/$pgbversion/pgbouncer-$pgbversion.tar.gz && \
  cd /tmp && \
  # Unpack, compile
  tar xvfz /tmp/pgbouncer-$pgbversion.tar.gz && \
  cd pgbouncer-$pgbversion && \
  ./configure --prefix=/usr --without-openssl && \
  make && \
  # Manual install
  cp pgbouncer /usr/bin && \
  mkdir -p /etc/pgbouncer /var/log/pgbouncer /var/run/pgbouncer && \
  chown -R postgres /var/run/pgbouncer /etc/pgbouncer && \
  # Cleanup
  cd /tmp && \
  rm -rf /tmp/pgbouncer*  && \
  apk del --purge autoconf autoconf-doc automake c-ares-dev curl gcc libc-dev libevent-dev libtool make man libressl-dev pkgconfig

CMD ["sh", "-c", "/usr/bin/pgbouncer $PGBOUNCER_INI_PATH -u nobody"]
