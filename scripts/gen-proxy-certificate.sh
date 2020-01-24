#!/bin/bash

openssl req -x509 -sha256 -nodes -days 365 \
        -newkey rsa:2048 \
        -subj "/CN=$(uuidgen).example.com/O=Exmaple Company Name LTD./C=US" \
        -keyout ./bin/key.pem \
        -out bin/cert.pem
