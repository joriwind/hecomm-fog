# HeComm fog implementation

# TLS server
## Generating password and certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365
!!Remark: "-nodes" for unprotected password

###Generating unencrypted password from protected password:
$ openssl rsa -in key.pem -out key.unencrypted.pem -passin pass:TYPE_YOUR_PASS

insert node {"devid":"11111111","platformid":1,"isprovider":false,"inftype":2}

#Certs for platforms:
Generate certificate and key:
    Option 1: Self-signed
    The simplest option. Just run this command on your server and you have a valid all-purpose certificate that is valid for the next ten years:

    openssl req -newkey rsa:4096 -nodes -sha512 -x509 -days 3650 -nodes -out /etc/ssl/certs/mailserver.pem -keyout /etc/ssl/private/mailserver.pem
    You will be asked for several pieces of information. Enter whatever you like. The only important field is the “Common Name” that must contain the fully-qualified host name that you want your server to be known on the internet. Fully-qualified means host + domain.

    Make sure that the secret key is only accessible by the ‘root’ user:

    chmod go= /etc/ssl/private/mailserver.pem
    Source: https://workaround.org/ispmail/jessie/create-certificate
##Set IP validation
So what you need to is:

- Edit your /etc/ssl/openssl.cnf on the logstash host - add subjectAltName = IP:192.168.2.107 in [v3_ca] section.
- Recreate the certificate
- Copy the cert and key to both hosts