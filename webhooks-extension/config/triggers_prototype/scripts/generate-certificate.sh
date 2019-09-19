#!/bin/bash -e

CertificateKeyPassphrase=$1 # Arbitrary
ExternalUrl=$2 # E.g. https://mylistener.myexternalipaddress.nip.io
SecretName=$3 # Arbitrary, but must match what's used in the Ingress creation task (CertificateSecretName)

if [ "$#" -ne 3 ]; then
  echo "Three parameters are required: the passphrase to use, the external URL and the name of the secret created that will store your certificate"
  echo "For example, ./scripts/generate-certificate.sh mypassphrase mylistener.myexternalipaddress.nip.io mycertificatesecret"
  exit;
fi

mkdir -p cert-files

openssl genrsa -des3 -out cert-files/key.pem -passout pass:${CertificateKeyPassphrase} 2048

openssl req -x509 -new -nodes -key cert-files/key.pem -sha256 -days 1825 -out cert-files/certificate.pem -passin pass:${CertificateKeyPassphrase} -subj /CN=${ExternalUrl}

openssl rsa -in cert-files/key.pem -out cert-files/key.pem -passin pass:${CertificateKeyPassphrase}

# Now create the secret that will be mounted in for use by the Ingress task
kubectl create secret tls ${SecretName} --cert=cert-files/certificate.pem --key=cert-files/key.pem

