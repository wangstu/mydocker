FROM golang:1.21.7

WORKDIR /root/

COPY ./output/mydocker .

RUN apt update && apt install -y fuse-overlayfs