package main

import (
    "fmt"
    "log"
    "net/http"
    "strings"
    "os"
    "io/ioutil"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
  var bodyBytes []byte
  var final string
  var qparams, curlheaders string

  final += "###################################################################\n"

  final += "HTTP Method:  "+r.Method+"\n"
  final += "REQUEST URL:  "+r.URL.Path+"\n"

  if r.Body != nil {
      bodyBytes, _ = ioutil.ReadAll(r.Body)
  }

  final += "BODY:         "+string(bodyBytes)+"\n"
  for key, val := range r.URL.Query() {
    final += "Query Param:  "+key+" = "+val[0]+"\n"
    if qparams != "" {
      qparams += "&"+key+"="+val[0]
    } else {
      qparams = "?"+key+"="+val[0]
    }
  }

  qparams += "' \\\n"
  for name, values := range r.Header {
      for _, value := range values {
        final += "HEADER:       "+name+" = "+value+"\n"
        if curlheaders != "" {
          curlheaders += "\n-H '"+name+": "+value+"' \\"
        } else {
          curlheaders += "-H '"+name+": "+value+"' \\"
        }
      }
  }

  if string(bodyBytes) == "" {
    final += "\nCURL COMMAND: \ncurl -X"+r.Method+" '{{host}}"+r.URL.Path+""+qparams+""+strings.TrimSuffix(curlheaders, "\\")+"\n\n"
  } else {
    final += "\nCURL COMMAND: \ncurl -X"+r.Method+" '{{host}}"+r.URL.Path+""+qparams+""+curlheaders+"--data-urlencode: '"+string(bodyBytes)+"'\n\n"
  }

  final += "###################################################################\n"

  f, err := os.OpenFile("/tmp/requestHeadersQueryParamsAndBody.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
      log.Fatal(err)
  }

  if _, err := f.Write([]byte(final)); err != nil {
      log.Fatal(err)
  }

  if err := f.Close(); err != nil {
      log.Fatal(err)
  }

  fmt.Fprintf(w, string(bodyBytes))
}

func main() {
    http.HandleFunc("/", indexHandler)

    fmt.Printf("Starting server at port 5464\n")
    if err := http.ListenAndServe(":5464", nil); err != nil {
        log.Fatal(err)
    }
}
