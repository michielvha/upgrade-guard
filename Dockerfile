FROM alpine:3.23

ARG IMAGE_NAME=upgrade-guard
ARG TARGETARCH
ARG TARGETOS

ENV IMAGE_NAME=${IMAGE_NAME}

RUN apk add --no-cache git ca-certificates wget

RUN addgroup -g 1001 ${IMAGE_NAME} && \
    adduser -D -u 1001 -G ${IMAGE_NAME} ${IMAGE_NAME}

RUN git config --system --add safe.directory '*'

COPY dist/${IMAGE_NAME}_${TARGETOS}_${TARGETARCH}*/${IMAGE_NAME} /usr/local/bin/${IMAGE_NAME}

RUN chmod +x /usr/local/bin/${IMAGE_NAME} && \
    chown ${IMAGE_NAME}:${IMAGE_NAME} /usr/local/bin/${IMAGE_NAME}

USER ${IMAGE_NAME}
WORKDIR /home/${IMAGE_NAME}

ENTRYPOINT ["/bin/sh", "-c", "/usr/local/bin/${IMAGE_NAME}"]
