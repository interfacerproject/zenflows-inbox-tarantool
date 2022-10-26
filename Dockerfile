FROM debian:bullseye

FROM golang:1.19-bullseye AS builder
RUN apt update && apt install -y build-essential git cmake vim python3 python3-pip zsh \
        && pip3 install meson ninja \
        && git clone https://github.com/dyne/Zenroom.git /zenroom
RUN cd /zenroom && make linux-go
ADD . /app
WORKDIR /app
RUN go build inbox.go zenflows-auth.go

FROM debian:bullseye
WORKDIR /root
ENV HOST=0.0.0.0
ENV PORT=80
EXPOSE 80
COPY --from=builder /app/inbox /root/
COPY --from=builder /zenroom/meson/libzenroom.so /usr/lib/
CMD ["/root/inbox"]
