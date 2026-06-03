FROM alpine:3.20

RUN echo "shared.Dockerfile is deprecated. Use Dockerfile / make build-linux for the WasmVM v3 static production build." && exit 1
