# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.15 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY console/ console/

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go build -a -o manager main.go


FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
MAINTAINER IBM Storage

ARG VCS_REF
ARG VCS_URL

LABEL vendor="IBM" \
  name="ibm-storage-odf-operator" \
  org.label-schema.vendor="IBM" \
  org.label-schema.name="ibm storage odf operator" \
  org.label-schema.vcs-ref=$VCS_REF \
  org.label-schema.vcs-url=$VCS_URL \
  org.label-schema.schema-version="1.0.1" \
  summary="IBM Storage ODF Operator" \
  description="operator and driver of ibm storage systems for openshift data foundation (ODF)"

WORKDIR /
COPY --from=builder /workspace/manager /manager
# COPY RULES
COPY /rules/*.yaml /prometheus-rules/
RUN mkdir /licenses
COPY /LICENSE /licenses/


ENV USER_UID=1001 \
    USER_NAME=ibm-storage-odf-operator

COPY hack/user_setup /usr/local/bin/user_setup
RUN chmod 777 /usr/local/bin/user_setup
RUN  /usr/local/bin/user_setup

USER ${USER_UID}

ENTRYPOINT ["/manager"]
