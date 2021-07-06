FROM golang:1.16.2 as build

ARG username
ARG token

ENV USERNAME=$username
ENV TOKEN=$token

ENV USER=appuser
ENV UID=10001 

RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --shell "/sbin/nologin" \    
    --no-create-home \    
    --uid "${UID}" \    
    "${USER}"

WORKDIR /go/src/bsh-backend

COPY ./src .
COPY go.mod .

#RUN GOCHACHE=OFF
RUN go env -w GOPRIVATE=gitlab.com/ms-ural
RUN git config --global url."https://${USERNAME}:${TOKEN}@gitlab.com".insteadOf "https://gitlab.com"
RUN go mod tidy -v
#RUN go install -v ./...
RUN CGO_ENABLED=1 go install -v ./...

# This results in a single layer image
FROM frolvlad/alpine-glibc:latest

RUN mkdir /usr/local/share/ca-certificates/
RUN wget https://letsencrypt.org/certs/isrgrootx1.pem -O /usr/local/share/ca-certificates/isrgrootx1.pem 
RUN wget https://letsencrypt.org/certs/isrg-root-x2.pem -O /usr/local/share/ca-certificates/isrg-root-x2.pem 
RUN wget https://letsencrypt.org/certs/letsencryptauthorityx3.pem -O /usr/local/share/ca-certificates/letsencryptauthorityx3.pem 
RUN wget https://letsencrypt.org/certs/lets-encrypt-x3-cross-signed.pem -O /usr/local/share/ca-certificates/lets-encrypt-x3-cross-signed.pem 
RUN wget https://letsencrypt.org/certs/lets-encrypt-r3.pem -O /usr/local/share/ca-certificates/lets-encrypt-r3.pem 
RUN wget https://letsencrypt.org/certs/lets-encrypt-e1.pem -O /usr/local/share/ca-certificates/lets-encrypt-e1.pem
RUN apk update && apk add ca-certificates && update-ca-certificates

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group

RUN mkdir /opt/certs && chown appuser:appuser /opt/certs

COPY --from=build /go/bin/backend /usr/bin/bsh-backend

EXPOSE 8080
EXPOSE 8443

USER appuser:appuser

CMD ["bsh-backend"]
