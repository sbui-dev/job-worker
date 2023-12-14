#!/bin/bash

  openssl ecparam -genkey -name prime256v1 -out ca.key

  openssl ecparam -out key.pem -name prime256v1 -genkey
  openssl req -new -sha256 -key key.pem -out server.csr
  openssl x509 -req -sha256 -days 365 -in server.csr -signkey key.pem -out cert.pem

# Create the server CA certs
openssl req -x509                                        \
  -newkey rsa:4096                                       \
  -nodes                                                 \
  -days 365                                              \
  -keyout server_ca_key.pem                              \
  -out server_ca_cert.pem                                \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-server_ca/   \
  -sha256

# Generate a server cert
openssl genrsa -out server_key.pem 4096
openssl req -new                                      \
  -key server_key.pem                                 \
  -days 365                                           \
  -out server_csr.pem                                 \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-server/

# Sign server cert
openssl x509 -req           \
  -in server_csr.pem        \
  -CAkey server_ca_key.pem  \
  -CA server_ca_cert.pem    \
  -days 365                 \
  -out server_cert.pem      \
  -set_serial 1             \
  -sha256

# Verify server cert
openssl verify -verbose -CAfile server_ca_cert.pem server_cert.pem

# Create the client CA certs
openssl req -x509                                         \
  -newkey rsa:4096                                        \
  -nodes                                                  \
  -days 365                                               \
  -keyout client_ca_key.pem                               \
  -out client_ca_cert.pem                                 \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-client_ca/    \
  -sha256

# Generate client certs
openssl genrsa -out alice_key.pem 4096
openssl req -new                                       \
  -key alice_key.pem                                   \
  -days 365                                            \
  -out alice_csr.pem                                   \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-client1/

openssl x509 -req           \
  -in alice_csr.pem        \
  -CAkey client_ca_key.pem  \
  -CA client_ca_cert.pem    \
  -days 365                 \
  -out alice_cert.pem      \
  -set_serial 1             \
  -sha256
openssl verify -verbose -CAfile alice_ca_cert.pem alice_cert.pem

rm *_csr.pem
