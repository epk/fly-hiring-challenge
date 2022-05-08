FROM gitpod/workspace-full

# install dependencies and tools
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && \
  apt-get install --no-install-recommends -y \
  ca-certificates\
  clang \
  curl \
  git \
  llvm \
  libelf-dev \
  make \
  netcat \
  openssh-server \
  openssl\
  && rm -rf /var/lib/apt/lists/*

# install kernel headers, libbpf, and bpftool
# on Ubuntu 21.01 this can be replaced with: libbpf-dev linux-tools-5.16.0-30-generic linux-cloud-tools-5.16.0-30-generic
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && \
  apt-get install --no-install-recommends -y \
  autoconf bison cmake dkms flex gawk gcc python3 rsync \
  libiberty-dev libncurses-dev libpci-dev libssl-dev libudev-dev \
  && curl https://cdn.kernel.org/pub/linux/kernel/v5.x/linux-5.16.tar.gz | tar -xz \
  && make -C /linux-5.16 headers_install INSTALL_HDR_PATH=/usr \
  && make -C /linux-5.16/tools/lib/bpf install INSTALL_HDR_PATH=/usr \
  && make -C /linux-5.16/tools/bpf/bpftool install \
  && apt-get remove -y \
  autoconf bison cmake dkms flex gawk gcc python3 rsync \
  libiberty-dev libncurses-dev libpci-dev libssl-dev libudev-dev \
  && apt autoremove -y \
  && rm -rf /var/lib/apt/lists/* \
  && rm -rf /linux-5.16
