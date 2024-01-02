The benchmarks will only work for amd64 builds.
To run the benchmarks, you have to amend the path in the `replace` line in `go.mod`.

To generate a diagram use e.g.
  GOMAXPROCS=100 go test -bench=Get -benchmem | grep -E ': |Parallel' >bench.out
  ./plot.py
(a diagram opens, it is also stored in bench.png)
