# FROM golang:1.16-alpine as builder
# ENV CGO_ENABLED=0
# WORKDIR /go/src/
# COPY go.mod go.sum ./
# RUN go mod download
# COPY . .
# RUN go build -ldflags '-w -s' -v -o /usr/local/bin/kubeconformvalidator ./

# FROM alpine:latest
# COPY --from=builder /usr/local/bin/kubeconformvalidator /usr/local/bin/kubeconformvalidator
# ENTRYPOINT ["kubeconformvalidator"]
