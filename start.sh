#!/bin/bash

CTSTATUS="dev"
CTORG="b"
CTPRODUCT="bsh"
CTCOMPONENT="backend"
CTPROGRAM="bsh-backend"

set -e

docker pull skipufo/bsh-backend

if docker stop bsh-backend; then
  echo "container not started"
fi

if docker rm bsh-backend; then
  echo "container not existed"
fi

docker run \
        --restart=always \
        -m 1000M \
        -dit \
        -p 443:8443 \
        -p 80:8080 \
        --name bsh-backend \
        -v ~/ssl:/opt/certs \
        --log-opt max-size=300m \
	--log-opt max-file=10 \
        --label name=bsh-backend \
        --label status=${CTSTATUS} \
        --label org=${CTORG} \
        --label product=${CTPRODUCT} \
        --label component=${CTCOMPONENT} \
        --label program=${CTPROGRAM} \
        --log-driver json-file \
        --log-opt labels=name,status,org,product,component,program \
        --log-opt tag="skipufo/bsh-backend,${CTSTATUS},${CTORG},${CTPRODUCT},${CTCOMPONENT},${CTPROGRAM}" \
skipufo/bsh-backend
