
PASS="winderickx"

if [ "$1" = "router" ]
then

scp hecomm-fog root@192.168.2.1:/tmp/ -w "$PASS"
scp -r certs/ root@192.168.2.1:/tmp/ -w "$PASS"

fi