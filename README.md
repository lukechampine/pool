mempool
-------

[![GoDoc](https://godoc.org/github.com/lukechampine/mempool?status.svg)](https://godoc.org/github.com/lukechampine/mempool)
[![Go Report Card](http://goreportcard.com/badge/github.com/lukechampine/mempool)](https://goreportcard.com/report/github.com/lukechampine/mempool)

```
go get github.com/lukechampine/mempool
```

`mempool` is a tiny library for pooling fixed-size `[]byte` buffers.


## API ##
```go
// create pool with 10 100-byte buffers
pool := mempool.New(10, 100)

// get a buffer
b := pool.Get()

// return a buffer
pool.Put(b)
```

See the [GoDoc](https://godoc.org/github.com/lukechampine/mempool) for full documentation.


## Benchmarks ##

```
BenchmarkPool-4             	30000000	        39 ns/op	       0 B/op	       0 allocs/op
BenchmarkPoolContention-4   	20000000	       122 ns/op	       0 B/op	       0 allocs/op
```
