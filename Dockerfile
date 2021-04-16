FROM golang:1.16 AS tool-build
WORKDIR /work
COPY *.go go.mod ./
RUN go build

FROM ubuntu:20.04 AS runtime
WORKDIR /work
RUN apt-get update -y \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        automake \
        build-essential \
        gcovr \
        git \
        jq \
        less \
        strace \
 && rm -rf /var/cache/apt/*

FROM runtime AS pure-ftpd
RUN groupadd fuzzing \
 && useradd -rm -d /home/fuzzing -s /bin/bash -g fuzzing -G sudo -u 1000 fuzzing \
        -p "$(openssl passwd -1 fuzzing)"
COPY patches/pure-ftpd-gcov.patch gcov.patch
RUN git clone --depth 1 https://github.com/jedisct1/pure-ftpd.git --branch 1.0.49 --single-branch \
 && cd pure-ftpd \
 && patch -p1 < ../gcov.patch \
 && ./autogen.sh \
 && CFLAGS="-fprofile-arcs -ftest-coverage" ./configure --without-privsep -without-capabilities \
 && make -j
COPY --from=tool-build /work/msgbounds .
COPY scripts/pure-ftpd.sh .
ENTRYPOINT ./pure-ftpd.sh
