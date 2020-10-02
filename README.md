# Request Debugger:-
## Helps to debug the HTTP Request Headers, Query Params, Method, URL and Request Body.

## Try he request debugger using the below docker container:-
```
docker run -d  -v /tmp:/tmp  -p 1111:5464 --name requestdebugger kingalt/requestdebugger:1.1
```

## Try with a linux binary.
```
# Download the binary
wget https://github.com/SMYALTAMASH/requestdebugger/releases/download/V-1.1/main

# Run the binary
./main
```

## What does the too do?
* A tool to log HTTP request Method, URL, HEADERS, Query Params, Body.

* It writes the request inside /tmp/requestHeadersQueryParamsAndBody.log file.

* The container Runs internally on port 5464, expose it to your favourite port.

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
```

## Add it to nginx with mirror module to track the requests to whatever the path you want in real time.
```
        location / {
                mirror /mirror;
                # your other routing configuration
        }

        location /mirror {
            internal;
            proxy_pass http://127.0.0.1:5464$request_uri;
        }
```
