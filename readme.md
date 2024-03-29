# Tail & query real-time logs of many apps.
A simple protocol for log data, built on Protobuf, SHA256, and UDP.
Currently, there is only a map of ring buffers in memory.
This program does not yet write log data to a file, although this is clearly an important feature to come.
```bash
go run .
```

# To Do
## Fix replay-attack vulnerability
There is currently no cache of UDP packet hashes, so we can't yet detect & drop a replay. A small ring buffer would be ideal for this.
`Estimated time: 2 hours`

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
Generate protobuf code
```bash
protoc --go_out=. cmd.proto # generate source files
```

