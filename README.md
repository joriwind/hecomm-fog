# HeComm fog implementation

# TLS server
## Generating password and certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365
!!Remark: "-nodes" for unprotected password

###Generating unencrypted password from protected password:
$ openssl rsa -in key.pem -out key.unencrypted.pem -passin pass:TYPE_YOUR_PASS

insert node {"devid":"11111111","platformid":1,"isprovider":false,"inftype":2}