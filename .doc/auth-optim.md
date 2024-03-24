# Optimizing the packet signer
The packet signer hashes the secret, payload & current time.
The returned byte slice contains the 32B checksum, 8B time, and the payload.
```go
func Sign(secret, payload []byte, t time.Time) ([]byte, error) {
	timeBytes, err := convertTimeToBytes(t)
	if err != nil {
		return nil, fmt.Errorf("convert time to bytes err: %w", err)
	}
	// pre-allocate slice
	totalLen := SumLen + len(timeBytes) + len(payload)
	data := make([]byte, 0, totalLen)
	// copy data
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	// compute checksum
	h := sha256.Sum256(data)
	sum := h[:SumLen]
	// return sum + time + payload
	signed := make([]byte, 0, SumLen+TimeLen+len(payload))
	signed = append(signed, sum...)
	signed = append(signed, timeBytes...)
	return append(signed, payload...), nil
}
```

As we see from the benchmark result, we make 5 allocations & need ~190ns/op. Can we do better?
```
Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -bench ^BenchmarkSign$ github.com/inneslabs/logd/auth

goos: darwin
goarch: arm64
pkg: github.com/inneslabs/logd/auth
BenchmarkSign-12    	 5643075	       194.3 ns/op	     280 B/op	       5 allocs/op
PASS
ok  	github.com/inneslabs/logd/auth	1.484s
```

We can save an allocation by reusing the data slice:
```go
func Sign(secret, payload []byte, t time.Time) ([]byte, error) {
	// ...
	sum := h[:SumLen]
	// append sum and timeBytes to emptied data slice
	data = append(data[:0], sum...)
	data = append(data, timeBytes...)
	return append(data, payload...), nil
}
```

Benchmark result:
```
Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -bench ^BenchmarkSign$ github.com/inneslabs/logd/auth

goos: darwin
goarch: arm64
pkg: github.com/inneslabs/logd/auth
BenchmarkSign-12    	 6281907	       178.7 ns/op	     200 B/op	       4 allocs/op
PASS
ok  	github.com/inneslabs/logd/auth	1.424s
```

That's a little better...