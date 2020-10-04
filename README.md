# Request Debugger:-
## Helps to debug the HTTP Request Headers, Query Params, Method, URL and Request Body.

## What does the too do?
* A tool to log HTTP request Method, URL, HEADERS, Query Params, Body and to geenrate the curl command for easy debugging.

* It writes the request inside /tmp/requestHeadersQueryParamsAndBody.log file.

* The container Runs internally on port 5464, expose it to your favourite port.

## Try he request debugger using the below docker container:-
```
# To Run a docker container
docker run -d  -v /tmp:/tmp  -p 1111:5464 --name requestdebugger kingalt/requestdebugger:1.2

# Check the logs after one of the request is fired in our host system using,
tail -f /tmp/requestHeadersQueryParamsAndBody.log

# To delete the requestdebugger container
docker rm -f requestdebugger
```

## Try with a linux binary.
```
# Download the binary
wget https://github.com/SMYALTAMASH/requestdebugger/releases/download/V-1.2/main

# Make the binary executabe
chmod +x ./main

# Run the binary
./main
```

## Steps to build an executable go binary are:-
```
go build -o main requestHeadersQueryParamsAndBody.go
```

## Run the server
```
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
