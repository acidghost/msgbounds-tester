FROM golang:1.16 AS tool-build
WORKDIR /work
COPY go.mod go.sum .
RUN go mod download
COPY *.go .
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
COPY --from=tool-build /work/msgbounds-tester tester
COPY messages/ftp msgs
COPY scripts/common.bash scripts/pure-ftpd.sh .
ENTRYPOINT ["./pure-ftpd.sh"]
CMD ["-init-sleep=0", "-init-read=false", "-sleep=0", "-read=false", "-send-all"]

FROM runtime AS lightftp
RUN apt-get update \
 && apt-get install -y libgnutls28-dev \
 && rm -rf /var/cache/apt/* \
 && groupadd fuzzing \
 && useradd -rm -d /home/fuzzing -s /bin/bash -g fuzzing -G sudo -u 1000 fuzzing \
        -p "$(openssl passwd -1 fuzzing)"
COPY patches/lightftp-gcov.patch gcov.patch
RUN git clone https://github.com/hfiref0x/LightFTP.git \
 && cd LightFTP \
 && git checkout --detach 5980ea1 \
 && patch -p1 < ../gcov.patch \
 && cd Source/Release \
 && CFLAGS="-fprofile-arcs -ftest-coverage" make -j
COPY --from=tool-build /work/msgbounds-tester tester
COPY messages/ftp msgs
COPY scripts/common.bash scripts/lightftp.sh assets/certificate/ assets/lightftp.conf .
ENTRYPOINT ["./lightftp.sh"]
CMD ["-init-sleep=0", "-init-read=false", "-sleep=25us", "-read=false"]

FROM runtime AS live555
COPY patches/live555-gcov.patch gcov.patch
RUN git clone https://github.com/rgaufman/live555.git \
 && cd live555 \
 && git checkout ceeb4f4 \
 && patch -p1 < ../gcov.patch \
 && ./genMakefiles linux \
 && CFLAGS="-fprofile-arcs -ftest-coverage" CXXFLAGS="-fprofile-arcs -ftest-coverage" \
        LDFLAGS="-fprofile-arcs -ftest-coverage" make -j all
COPY --from=tool-build /work/msgbounds-tester tester
COPY messages/rtsp msgs
COPY scripts/common.bash scripts/live555.sh assets/test.mp3 .
ENTRYPOINT ["./live555.sh"]
CMD ["-init-sleep=0", "-init-read=false", "-sleep=0", "-read=false", "-send-all"]
