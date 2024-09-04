set -e
rm -rf bin
mkdir bin
cd cli
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../bin/easytc
cd ../bin && tar -zcvf easytc.amd64.tgz easytc
cd ../cli
env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../bin/easytc
cd ../bin && tar -zcvf easytc.arm64.tgz easytc
rm -f easytc
cd ..

