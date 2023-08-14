FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.20 as builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app/
ADD . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build

FROM --platform=${TARGETPLATFORM:-linux/amd64} redgoat650/barnacle-net:scratch-base

COPY --from=builder /app/barnacle-net /bin/barnacle-net
COPY ./docker/scripts /scripts

ENTRYPOINT 	["/bin/barnacle-net"]
