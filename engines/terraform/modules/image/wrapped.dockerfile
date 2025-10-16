ARG BASE_IMAGE

# TODO: Need to make sure the architecture for the build matches the base image
FROM golang:1.25.0 AS base

ARG PLUGIN_DEFINITION
ENV PLUGIN_DEFINITION=${PLUGIN_DEFINITION}

# Need to install make
RUN apt-get update && apt-get install -y make

# Checkout the suga github repo
RUN git clone --depth=1 -b feat/suga-plugin-serve https://github.com/nitrictech/suga /suga
WORKDIR /suga

RUN go work sync

WORKDIR /suga/server

RUN make

FROM $BASE_IMAGE

ARG ORIGINAL_COMMAND
ENV ORIGINAL_COMMAND=${ORIGINAL_COMMAND}

COPY --from=base /suga/server/bin/host /usr/local/bin/suga

# CMD ["-c", "$ORIGINAL_COMMAND"]
ENTRYPOINT /usr/local/bin/suga -c "$ORIGINAL_COMMAND"
