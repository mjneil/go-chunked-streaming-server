#!/usr/bin/env bash

# Create certs dir if it does not exists
mkdir -p ../certs

# Generate private key 
openssl genrsa -out ../certs/server.key 2048
openssl ecparam -genkey -name secp384r1 -out ../certs/server.key

# Generation of self-signed(x509) public key (PEM-encodings .pem|.crt) based on the private (.key)
openssl req -new -x509 -sha256 -key ../certs/server.key -out ../certs/server.crt -days 3650
