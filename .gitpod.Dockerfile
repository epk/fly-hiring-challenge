FROM gitpod/workspace-full

# install dependencies and tools
RUN DEBIAN_FRONTEND=noninteractive sudo install-packages\
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
  autoconf bison cmake dkms flex gawk gcc python3 rsync \
  libiberty-dev libncurses-dev libpci-dev libssl-dev libudev-dev

RUN cd /tmp && curl https://cdn.kernel.org/pub/linux/kernel/v5.x/linux-5.17.5.tar.gz | tar -xz
RUN sudo make -C /tmp/linux-5.17.5 headers_install INSTALL_HDR_PATH=/usr
RUN sudo make -C /tmp/linux-5.17.5/tools/lib/bpf install INSTALL_HDR_PATH=/usr
RUN sudo make -C /tmp/linux-5.17.5/tools/bpf/bpftool install
RUN sudo rm -rf /tmp/linux-5.17.5
