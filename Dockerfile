FROM alpine:latest

# goreleaser does the build
COPY ttnbridge /

EXPOSE 8080
ENV LISTEN_PORT=8080
ENV LISTEN_HOST=0.0.0.0

ENTRYPOINT ["/ttnbridge"]

