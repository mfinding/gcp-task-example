############################
# STEP 1 build main binary
############################x
FROM golang:1.17-alpine AS builder
LABEL stage=build

# Create appuser.
ENV USER=app
ENV UID=1001
# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser -D -H -s /sbin/nologin -g "" -h "/nonexistent" -u "${UID}" "${USER}"

WORKDIR /src

# COPY module config and download
COPY go.* ./


RUN go mod download

# COPY the source code as the last step
COPY . .

# Build the binaries.
RUN CGO_ENABLED=0 go build -o /go/bin/server .

##################################
# STEP 2 scratch image for runtime
##################################
FROM scratch as buildoutput

# Add user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy builder
COPY --from=builder /go/bin/* /go/bin/

# Add ssl certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080/tcp

USER app:app

# Run the webservice binary.
ENTRYPOINT ["/go/bin/server"]