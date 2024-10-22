**Security_2**

To run this program you must have openSSL installed on your computer, and use the following commands to set up a certificate:

**Create RSA key**

openssl genrsa -out server.key 2048

**Create Certificate from RSA key**

openssl req -new -x509 -sha256 -key server.key -out server.crt -days 365 -addext "subjectAltName = DNS:localhost"

after the server.crt and server.key has been created, open a bash terminal

**run the program in bash from the root directory**

bash main.sh
