# Goastic

Primitive and simple tool to benchmark Elasticsearch performance.

It's kind of a hack-job so it may not work for newer Elasticsearch versions, nor
for your specific purpose. That's what it says on the tin.

# Installation

Nothing fancy, just the usual:

```
$ go get github.com/lillesvin/goastic
```

# Usage

By default Goastic *will* write to whatever index you point it at, so either
use the `-readonly` switch or point it at a non-existing index (which it will
create, because at least it's got *that* going for it).

Here's the output of `goastic -help`:

```
$ goastic -help
Usage of goastic:
  -baseurl string
    	Base URL of Elasticsearch (default "http://localhost:9200/test")
  -interval int
    	Interval between requests in ms (default 5)
  -readonly
    	Only test reads
  -requests int
    	Number of requests to make (default 10000)
  -workers int
    	Number of parallel workers to run (default 2)
```

After it's done making requests it will output some stats:

```
$ goastic -baseurl http://some.elasticsearch.example.com:9200/test -requests 1000 -workers 8
Elasticsearch: http://some.elasticsearch.example.com:9200/test
Requests:      1000
Interval:      5 ms
Workers:       8
ReadOnly:      false


....................................................................................................

Mode: read
 - Requests (total):  500
 - Requests (failed): 0
 - Request time (total): 22.562 s
 - Request time (max.):  196 ms
 - Request time (min.):  24 ms
 - Request time (avg.):  45 ms
Mode: write
 - Requests (total):  500
 - Requests (failed): 0
 - Request time (total): 34.478 s
 - Request time (max.):  216 ms
 - Request time (min.):  39 ms
 - Request time (avg.):  68 ms
```

