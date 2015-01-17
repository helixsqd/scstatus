package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const httpPort = "8000"

// writes out the entries via json
func jsonData(w http.ResponseWriter, r *http.Request) {
	sortStr := r.FormValue("sort")
	refresh := r.FormValue("refresh")
	if refresh == "true" {
		gatherData()
	}
	if sortStr != "" {
		Sort(sortStr, entries)
	}

	jsonBytes, err := json.Marshal(entries)
	if err != nil {
		fmt.Println(err)
		return
	}

	buf := new(bytes.Buffer)
	json.HTMLEscape(buf, jsonBytes)
	w.Header().Add("Content-Type", "application/json")
	w.Write(buf.Bytes())
}

func startServer() {
	http.HandleFunc("/data", jsonData)
	http.HandleFunc("/", serveHtml)
	err := http.ListenAndServe(":"+httpPort, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func serveHtml(w http.ResponseWriter, r *http.Request) {
	htmlBuf := bytes.NewBufferString(`
	<html>
<head>
    <link href="//maxcdn.bootstrapcdn.com/bootstrap/3.3.1/css/bootstrap.min.css" rel="stylesheet">
    <script src="//cdnjs.cloudflare.com/ajax/libs/zepto/1.1.4/zepto.min.js"></script>
</head>
<body>
<div class="container">
    <div class="row">
        <div class="col-md-12">
            <a class="btn btn-default" onclick="refetchData(true); return false;" role="button">Fetch Latest Status</a>
        </div>
    </div>
    <div class="row">
        <div class="col-md-12">
            <table class="table table-condensed table-striped table-bordered">
                <thead>
                <tr>
                    <th><a onclick="sort='host'; refetchData(); return false;">Host</a></th>
                    <th><a onclick="sort='uri'; refetchData(); return false;">URI</a></th>
                    <th><a onclick="sort='time'; refetchData(); return false;">Time (ms)</a></th>
                    <th><a onclick="sort='ip'; refetchData(); return false;">Remote IP</a></th>
                </tr>
                </thead>
                <tbody>
                </tbody>
            </table>
        </div>
    </div>
</div>
<script>
    var sort = "";
    function refetchData(refreshData) {
        $.ajax({
            url: "/data?sort=" + sort + "&refresh=" + refreshData,
            context: document.body,
            cache: false,
            success: function (data) {
                $('tbody').html("");
                $.each(data, function (index, entry) {
                    var newTr = $('<tr></tr>');
                    newTr.addClass(pickClass(entry.attrs.requestProcessingTime))
                    var newTh = $("<td></td>").text(entry.host);
                    newTr.append(newTh);
                    newTh = $("<td></td>").text(entry.attrs.uri);
                    newTr.append(newTh);
                    newTh = $("<td></td>").text(entry.attrs.requestProcessingTime);
                    newTr.append(newTh);
                    newTh = $("<td></td>").text(entry.attrs.remoteAddr);
                    newTr.append(newTh);
                    $('tbody').append(newTr);
                });
            }

        })
    }

    function pickClass(timeStr) {
        time = parseInt(timeStr);
        if(time > 5000) {
            return "danger";
        } else if(time > 2000) {
            return "warning"
        } else {
            return "";
        }
    }

    refetchData();
</script>
</body>
</html>`)

	w.Write(htmlBuf.Bytes())
}
