FROM gcr.io/google-containers/debian-base-amd64:0.3.2

RUN clean-install ca-certificates

COPY event-exporter /