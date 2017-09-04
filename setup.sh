
PASS="winderickx"


build(){
    echo "Building executable..."
    env GOOS=linux GOARCH=arm go build -o hecomm-fog
}

copy(){
    echo "Copying files..."
    sshpass -p "$PASS" scp hecomm-fog root@192.168.2.1:/tmp/
    sshpass -p "$PASS" scp -r certs/ root@192.168.2.1:/tmp/
}

case $1 in
    "build")
        build 
    ;;
    "router")
        copy
    ;;
    "all")
        build
        copy
        ;;

    *)
        echo "Unexpected command: $1; expected: build || router"
    ;;


esac
echo "Done"