package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var log = logrus.New()

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var bodyBytes []byte
	var final string
	var qparams, curlheaders string
	t := time.Now()
	timeStamp := t.Format(time.RFC1123)

	defer r.Body.Close()

	final += "###################################################################\n"
	final += "TIMESTAMP:    " + timeStamp + "\n"
	final += "HTTP Method:  " + r.Method + "\n"
	final += "REQUEST URL:  " + r.URL.Path + "\n"

	if r.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(r.Body)
	}

	final += "BODY:         " + string(bodyBytes) + "\n"
	for key, val := range r.URL.Query() {
		final += "Query Param:  " + key + " = " + val[0] + "\n"
		if qparams != "" {
			qparams += "&" + key + "=" + val[0]
		} else {
			qparams = "?" + key + "=" + val[0]
		}
	}

	qparams += "' \\\n"
	for name, values := range r.Header {
		for _, value := range values {
			final += "HEADER:       " + name + " = " + value + "\n"
			if curlheaders != "" {
				curlheaders += "\n-H '" + name + ": " + value + "' \\"
			} else {
				curlheaders += "-H '" + name + ": " + value + "' \\"
			}
		}
	}

	if string(bodyBytes) == "" {
		if r.Header.Get("Requestdebugger_url") != "" {
			final += "\nCURL COMMAND: \ncurl -X" + r.Method + " '" + r.Header.Get("Requestdebugger_url") + "" + r.URL.Path + "" + qparams + "" + strings.TrimSuffix(curlheaders, "\\") + "\n\n"
		} else {
			final += "\nCURL COMMAND: \ncurl -X" + r.Method + " '{{host}}" + r.URL.Path + "" + qparams + "" + strings.TrimSuffix(curlheaders, "\\") + "\n\n"
		}
	} else {
		if r.Header.Get("Requestdebugger_url") != "" {
			final += "\nCURL COMMAND: \ncurl -X" + r.Method + " '" + r.Header.Get("Requestdebugger_url") + "" + r.URL.Path + "" + qparams + "" + strings.TrimSuffix(curlheaders, "\\") + "-d '" + string(bodyBytes) + "'\n\n"
		} else {
			final += "\nCURL COMMAND: \ncurl -X" + r.Method + " '{{host}}" + r.URL.Path + "" + qparams + "" + strings.TrimSuffix(curlheaders, "\\") + "-d '" + string(bodyBytes) + "'\n\n"
		}
	}

	final += "###################################################################\n"
	log.Info(final)

	//f, err := os.OpenFile("/tmp/requestHeadersQueryParamsAndBody.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//if _, err := f.Write([]byte(final)); err != nil {
	//	log.Fatal(err)
	//}

	//if err := f.Close(); err != nil {
	//	log.Fatal(err)
	//}

	fmt.Fprintf(w, string(bodyBytes)+"\n\n")
}

func main() {
	// LOGGING: https://github.com/sirupsen/logrus
	log.Out = os.Stdout
	//	log.SetLevel(log.InfoLevel)
	http.HandleFunc("/", indexHandler)
	log.Info("Starting server at port 5464\n")
	if err := http.ListenAndServe(":5464", nil); err != nil {
		log.Fatal(err)
	}
}
