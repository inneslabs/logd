# Logd


## Tail & query logs for unlimited apps.
Logd is an application that stores an in-memory map of ring buffers. The service listens for ephemerally-signed UDP packets. Each packet is a log event.

The protocol is very simple, defined in Protobuf.

Logd will not run out of memory if the buffer sizes are ok for the provisioned memory.
I sometimes bet that most logs don't fill the packet buffer, so I under-provision memory.

The ring buffer uses a single atomic pointer, `head`.
Each write advances the pointer forward.
Reading is normally back from `head`.

# To Do
## Fix replay-attack vulnerability
Note: If we run logd in the private network, this is absolutely no issue, but would be nice to implement for sake of correctness.

There is currently no cache of UDP packet hashes, so we can't yet detect & drop a replay. A small ring buffer would probably be ideal for this.
`Estimated time: 2 hours`

# Auth
Logd authenticates clients for either reading or writing using shared that could be named `LOGD_READ_SECRET` and `LOGD_WRITE_SECRET`. These are stored encrypted in our secrets SOPS file, and set in AWS Secrets Manager.

## Why shared secrets?
Writing is over UDP only. This will not change because cheap real-time logging is the core offering.

I chose to use hash-based ephemeral message authentication with a very short signature ttl (100ms) to limit the potential for replays. Preventing replays futher is then much easier & less computationally expensive.

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

// get ephemeral signature using current time
signedMsg, _ := auth.Sign("some-secret-value", payload, time.Now())

// write to socket
socket.Write(signedMsg)
```

# Protobuf
Generate protobuf code
```bash
protoc --go_out=. cmd.proto # generate source files
```

