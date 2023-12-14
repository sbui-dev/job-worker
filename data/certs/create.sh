#!/bin/bash

# Create the server CA certs
openssl req -x509                                        \
  -newkey rsa:4096                                       \
  -nodes                                                 \
  -days 365                                              \
  -keyout server_ca_key.pem                              \
  -out server_ca_cert.pem                                \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-server_ca/   \
  -config ./config.cnf                                  \
  -extensions test_ca                                    \
  -sha256

# Generate a server cert
openssl genrsa -out server_key.pem 4096
openssl req -new                                      \
  -key server_key.pem                                 \
  -days 365                                           \
  -out server_csr.pem                                 \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=job-server/   \
  -config ./config.cnf                                \
  -reqexts job-server

# Sign server cert
openssl x509 -req           \
  -in server_csr.pem        \
  -CAkey server_ca_key.pem  \
  -CA server_ca_cert.pem    \
  -days 365                 \
  -out server_cert.pem      \
  -set_serial 1             \
  -extfile ./config.cnf     \
  -extensions job-server   \
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
  -config ./config.cnf                                    \
  -extensions test_ca                                     \
  -sha256

# Generate client certs
openssl genrsa -out alice_key.pem 4096
openssl req -new                                       \
  -key alice_key.pem                                   \
  -days 365                                            \
  -out alice_csr.pem                                   \
  -subj /C=US/ST=CA/L=SF/O=JobWorker/CN=alice/         \
  -config ./config.cnf                                 \
  -reqexts test_client

openssl x509 -req           \
  -in alice_csr.pem         \
  -CAkey client_ca_key.pem  \
  -CA client_ca_cert.pem    \
  -days 365                 \
  -out alice_cert.pem       \
  -set_serial 1             \
  -extfile ./config.cnf     \
  -extensions test_client   \
  -sha256
openssl verify -verbose -CAfile client_ca_cert.pem alice_cert.pem

rm *_csr.pem
