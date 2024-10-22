# Commands for testing

For the go.sum files
```
go get github.com/JonasSkjodt/security_2@latest
```

Create RSA key
```
openssl genrsa -out server.key 2048
```

Create Certificate from RSA key
```
openssl req -new -x509 -sha256 -key server.key -out server.crt -days 365 -addext "subjectAltName = DNS:localhost"
```

Run program with bash
```
bash main.sh
```
