FROM ubuntu:20.04

# install dependencies and tools
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && apt-get install -y \
  build-essential \
  ca-certificates\
  clang \
  curl \
  git \
  llvm \
  libelf-dev \
  make \
  netcat \
  openssl\
  binutils-dev\
  libcap-dev\
  openssh-server\
  autoconf bison cmake dkms flex gawk gcc python3 rsync \
  libiberty-dev libncurses-dev libpci-dev libssl-dev libudev-dev\
  zsh

RUN cd /tmp && curl https://cdn.kernel.org/pub/linux/kernel/v5.x/linux-5.17.5.tar.gz | tar -xz
RUN make -C /tmp/linux-5.17.5 headers_install INSTALL_HDR_PATH=/usr
RUN make -C /tmp/linux-5.17.5/tools/lib/bpf install INSTALL_HDR_PATH=/usr
RUN make -C /tmp/linux-5.17.5/tools/bpf/bpftool install
RUN rm -rf /tmp/linux-5.17.5

# install go
ENV GOPATH="/root"
ENV PATH="/usr/local/go/bin:$GOPATH/bin:$PATH"

RUN curl -L https://go.dev/dl/go1.18.1.linux-amd64.tar.gz | tar -xz -C /usr/local;
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" "$GOPATH/pkg" && chmod -R 777 "$GOPATH"
# configure sshd
RUN mkdir /run/sshd; \
  sed -i 's/^#\(PermitRootLogin\) .*/\1 yes/' /etc/ssh/sshd_config; \
  sed -i 's/^\(UsePAM yes\)/# \1/' /etc/ssh/sshd_config;

# entrypoint
RUN { \
  echo '#!/bin/bash -eu'; \
  echo 'ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime'; \
  echo 'echo "root:${ROOT_PASSWORD}" | chpasswd'; \
  echo 'exec "$@"'; \
  } > /usr/local/bin/entry_point.sh; \
  chmod +x /usr/local/bin/entry_point.sh;

ENV TZ=UTC

ENV ROOT_PASSWORD root

EXPOSE 22

ENTRYPOINT ["entry_point.sh"]
CMD    ["/usr/sbin/sshd", "-D", "-e", "-p", "2222"]
