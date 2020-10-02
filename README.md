# Request Debugger: Helps to debug the HTTP Request Headers, Query Params, Method, URL and Request Body.

## Try he request debugger using the below docker container:-
```
docker run -d  -v /tmp:/tmp  -p 1111:5464 --name requestdebugger kingalt/requestdebugger:1.1
```

## What does the too do?
* A tool to log HTTP request Method, URL, HEADERS, Query Params, Body.

* It writes the request inside /tmp/requestHeadersQueryParamsAndBody.log file.

* The container Runs internally on port 5464, expose it to your favourite port.

## Steps to build an executable go binary are:-
```
go build -o main requestHeadersQueryParamsAndBody.go
```
