# scstatus: Servlet Container Aggregator

scstatus is a command line application for aggregating the container status from multiple servers. Currently only Tomcat is supported.


Feature list:

 * Grab the list of requests being processed by multiple tomcat servers
 * Requests status in parallel for fast performance
 * Shows the Remote IP, URI, Host, and Request Processing time
 * Builtin HTTP server for viewing results in browser

Usage:
```
# scstatus -h
Usage of ./scstatus:
  -p="": Password
  -s="host": Field to sort on: [host, uri, time, ip]
  -t=3000: Request timeout
  -u="": Username
  -w=false: Start HTTP server on port 8000
```

Simple example:
````
# scstatus -u manageruser -p myvoiceismypassport http://127.0.0.1:8080/manager/status

Host            URI              Time(ms) Remote Addr       
127.0.0.1:8080  /manager/status  4        127.0.0.1
```
If the manager is using the default context this can be shortened to:
```
# scstatus -u manageruser -p myvoiceismypassport 127.0.0.1:8080

Host            URI              Time(ms) Remote Addr       
127.0.0.1:8080  /manager/status  4        127.0.0.1
```

A list of servers works as well:
```
# scstatus -u manageruser -p myvoiceismypassport server1:8080 server2:8080 ... serverN:8080
```

Or you can pass the list via stdin:
```
# cat server-list | scstatus -u manageruser -p myvoiceismypassport
# scstatus -u manageruser -p myvoiceismypassport < server-list
```

You can "discover" servers using DNS as well:
```
# scstatus -u manageruser -p myvoiceismypassport server%d:8080
```
The %d will be replaced with incrementing integers starting at 1 until DNS resolution fails.

The results can be sorted by any field:
```
# scstatus -s host -u manageruser -p myvoiceismypassport server%d:8080
# scstatus -s time -u manageruser -p myvoiceismypassport server%d:8080
# scstatus -s uri -u manageruser -p myvoiceismypassport server%d:8080
# scstatus -s addr -u manageruser -p myvoiceismypassport server%d:8080
```

These results can be used to gather data such as number of connections by a single IP (or URI, host, etc):
```
# scstatus -u manageruser -p myvoiceismypassport server%d:8080 | awk '{print $4}'| sort | uniq -c | sort -n
```

If you prefer a web interface (for sharing the URL, sorting in browser, etc) you can start one with:
```
# scstatus -u manageruser -p myvoiceismypassport http://127.0.0.1:8080/manager/status

Host            URI              Time(ms) Remote Addr       
127.0.0.1:8080  /manager/status  4        127.0.0.1
Starting server on port 8000
```
Building
-------
```
# git clone https://github.com/RyanAD/scstatus.git
# cd scstatus
# go get
# go build
```
TODO
-------
* Code cleanup
* Create interfaces to abstract fetching and parsing the status from various containers
* Add support for more containers

License
-------

   Copyright 2014 Ryan Dearing

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
