# v0.16.0:
v0.15 through to v0.16 has been about solving replays. Since v0, the message consists of the following:
[_TIME_BYTES]
```go
// pkg/pkg.go
type Pkg struct {
	Sum, TimeBytes, Payload []byte
}

func Unpack(data []byte, pkg *Pkg) error {
	if len(data) < 32+15 {
		return errors.New("data too short")
	}
	pkg.Sum = data[:32]              /*32B sha256*/
	pkg.TimeBytes = data[32 : 32+15] /*15B time*/
	pkg.Payload = data[32+15:]       /* payload */
	return nil
}

func Sign(secret, payload []byte) []byte {
	timeBytes, _ := time.Now().MarshalBinary() // 15B
	data := make([]byte, 0, 32 /* sha256 */ +15 /* time */ +len(payload))
	data = append(data, secret...)
	data = append(data, timeBytes...)
	data = append(data, payload...)
	h := sha256.Sum256(data)
	data = append(data[:0], h[:32]...)
	data = append(data, timeBytes...)
	return append(data, payload...)
}
```
## First iteration was a ring buffer of the sha256 sums, named `history`. This did not scale well, because the more packets received, the larger the history must be, but then the more expensive it is to verify each packet is not in the history.


## Cuckoo filter
- Simpler
- Faster (>200K/s on my machine)


# Tail & query real-time logs of many apps.
A simple program for streaming log data, built on Protobuf, SHA256, and UDP.

This program does not yet write log data to a file, although this is clearly an important feature to come.
```bash
go run .
```

# Config
The program will search up to the root for a file named `logdrc.yml`.
```yaml
# logdrc.yml
udp:
  laddr_port: ":6102"
  guard:
    history_size: 10000
    sum_ttl: 100ms
app:
  laddr_port: ":6101"
store:
  ring_sizes:
    /prod/my/app/http: 1000000
    /prod/my/app/udp: 1000000
    /debug: 10000
  fallback_size: 1000000
```
You may set your secrets in here, or as env vars.
```bash
export LOGD_READ_SECRET = "123456"
export LOGD_WRITE_SECRET = "123456"
```

# Auth
Logd authenticates clients for either reading or writing using SHA256 hash-based message authentication.

I chose to use hash-based ephemeral message authentication with a very short TTL (100ms)
because it's computationally cheap, and simple, and it's cheap to guard against replays over a short timespan.

Writing is over UDP only. This will *probably* not change for sake of simplicity, although sometimes I do wish for it.

# Logger
The simplest way to write logs is using the `logger` package.
```go
log, err := logger.NewLogger(context.TODO(), &logger.LoggerCfg{
		Host:   "some.host",
		Port:   6102,
		Secret: "your-writer-secret",
		MsgKey: "/ops/joey/my-app", // allows us to filter log data
		Stdout: true,               // also write to stdout
	})
log.Info("🌱 this is how we write logs, baby: %s", err)
```

## Custom integration
Logs are written by connecting to a UDP socket.
See the following example. Error checks skipped for brevity.
```go
// dial udp
addr, _ := conn.GetAddr("your.host")
socket, _ := conn.Dial(addr)

// serialise message using protobuf
payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:          timestamppb.Now(),
			Key:        "/your/app",
			Lvl:        cmd.Lvl_INFO,
			Txt:        "some log message",
		},
	})

// sign packet
signedMsg, _ := auth.Sign([]byte("your-secret"), payload, time.Now())

// write to socket
socket.Write(signedMsg)
```

# Protobuf
If you modify the protobuf spec in `cmd.proto`, you must re-generate the code.
```bash
# install protobuf & gen-go
brew install protobuf
brew install protoc-gen-go

# generate protobuf source files
protoc --go_out=. cmd.proto
```
