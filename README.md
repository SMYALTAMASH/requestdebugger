# Request Debugger:-
## Helps to debug the HTTP Request Headers, Query Params, Method, URL and Request Body.

## What does the tool do?
* A tool to log HTTP request Method, URL, HEADERS, Query Params, Body and to geenrate the curl command for easy debugging.

* It writes the request inside /tmp/requestHeadersQueryParamsAndBody.log file.

* The container Runs internally on port 5464, expose it to your favourite port.

## Try the request debugger using the below docker container:-
```
# To Run a docker container
docker run -d  -v /tmp:/tmp  -p 1111:5464 --name requestdebugger masteralt/requestdebugger:version1

# Check the logs after one of the request is fired in our host system using,
tail -f /tmp/requestHeadersQueryParamsAndBody.log

# To delete the requestdebugger container
docker rm -f requestdebugger
```

## Try with a prebuilt and ready to use binary.
#### SUPPORTED OS WITH PREBUILT BINARIES ARE: windows, MAC, Linux.
```
# Download the binary from latest release from the folder "RequestDebuggerBinariesForAllOS" from root directory

# unzip the latest release folder

# Choose the binary which matches your OS

# make the binary executable [In my case the OS is ubuntu which is linux amd64]
chmod +x requestDebugger-linux-amd64

# Run the binary [In my case the OS is ubuntu which is linux amd64]
./requestDebugger-linux-amd64
```

## Steps to build an executable go binary are:-
```
# clone the repo

# build the binary using the below command
go build -o main requestHeadersQueryParamsAndBody.go

# Run the server
./main # Runs the server
```

## Test the request debugger by firing some curl commands or the one below.
```
curl "http://127.0.0.1:5464/?size=8192&firstkey=firstvalue%40123" -H 'Header1: value1' -d '{"dataKey":"dataValue"}'

# we will receive the value of the HTTP request body in the output, OUTPUT WILL BE AS FOLLOWS:
{"dataKey":"dataValue"}
```

## Check the Logs
```
tail -f /tmp/requestHeadersQueryParamsAndBody.log

# OUTPUT:
###################################################################
HTTP Method:  POST
REQUEST URL:  /
BODY:         {"dataKey":"dataValue"}
Query Param:  size = 8192
Query Param:  firstkey = firstvalue@123
HEADER:       User-Agent = curl/7.58.0
HEADER:       Accept = */*
HEADER:       Header1 = value1
HEADER:       Content-Length = 23
HEADER:       Content-Type = application/x-www-form-urlencoded

CURL COMMAND:
curl -XPOST '{{host}}/?size=8192&firstkey=firstvalue@123' \
-H 'User-Agent: curl/7.58.0' \
-H 'Accept: */*' \
-H 'Header1: value1' \
-H 'Content-Length: 23' \
-H 'Content-Type: application/x-www-form-urlencoded' \--data-urlencode: '{"dataKey":"dataValue"}'

###################################################################
```
* The curl command really helps in using it directly without the hassle of going through all the query params, headers and different body params.
* change the value of "{{host}}" from the curl command output as your environment host name, else create an env variable in postman as host and directly import the curl command and fire it.
