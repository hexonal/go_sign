FROM mcr.microsoft.com/playwright:v1.43.1-focal

WORKDIR /app

# 安装最新版 Go（1.21.0），并设置 PATH
ENV GOLANG_VERSION=1.21.0
RUN apt-get update && \
    apt-get install -y --no-install-recommends gcc libc6-dev curl && \
    curl -fsSL https://go.dev/dl/go${GOLANG_VERSION}.linux-$(dpkg --print-architecture).tar.gz -o golang.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf golang.tar.gz && \
    rm golang.tar.gz && \
    ln -sf /usr/local/go/bin/go /usr/bin/go && \
    rm -rf /var/lib/apt/lists/*

ENV PATH="/usr/local/go/bin:${PATH}"

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal ./internal

RUN CGO_ENABLED=1 go build -o go_sign main.go

RUN curl -fsSL -o stealth.min.js "https://raw.githubusercontent.com/requireCool/stealth.min.js/main/stealth.min.js"

EXPOSE 5005

CMD ["./go_sign", "--stealth=./stealth.min.js", "--addr=:5005"] 