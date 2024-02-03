#!/bin/bash

if [[ "$1" != "" ]]; then
    DOMAIN="$1"
else
    DOMAIN="localhost"
fi


openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout $DOMAIN.key -out $DOMAIN.crt \
    -subj "/C=US/ST=Seattle/L=Seattle/O=Krydus/OU=Development/CN=$DOMAIN/emailAddress=it@krydus.com"
