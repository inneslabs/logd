# Logd
![A circular buffer](.doc/circular_buffer.svg)

## Tail & query logs for unlimited apps.
Logd is an application that stores an in-memory map of ring buffers. The service listens for ephemerally-signed UDP packets. Each packet is a log event.

The protocol is very simple, defined in Protobuf. 

Logd will never run out of memory if the buffer sizes are ok for the provisioned memory.

The buffer uses a single moving pointer, `head`. Each write advances the pointer forward. Reading is normally back from `head`. **No mutex**, `head` is an `atomic.Uint32`. This is a key reason for the performance of the ring buffer.

# To Do
## Mmove to AWS, or bridge WireGuard PN with VPC.
I want to applications to write directly to Logd. This will give us **REALTIME** log data. Some apps run in the VPC. I also want to put logd in the PN, and expose only the HTTP endpoint externally (only status). Authenticated RPCs are already **ONLY OVER UDP**.

### 2024-03-07 Decision to move to AWS.
See (./doc/bridge-fly-aws.md)[bridging fly with aws].
This would involve either configuring WireGuard on an EC2 instance, or configuring a VPN Gateway service from AWS.

The added complexity & therefore caution required currently seems to outweight the advantage of running apps on fly.

Therefore, I will migrate logd & zoe to EC2 instances. We can then benefit from the simplicity of running logged in the private network, and exposing only the http port for the status json externally.

## Fix replay-attack vulnerability
Note: If we run logd in the private network, this is absolutely no issue, but would be nice to implement for sake of correctness.

There is currently no cache of UDP packet hashes, so we can't yet detect & drop a replay. A small ring buffer would probably be ideal for this.
`Estimated time: 2 hours`

## Automate secret rotation
Note: Easier on EC2. See Move to AWS.
Once the Secrets Manager Rotation topic is in production, we can integrate this.
There is one consideration. We will need to periodically read the env var so that we can update this during runtime, without need to restart the application.
**Maybe it is no-longer necessary to store this secret in the SOPS file.**
`Estimated time: 4 hours`

# Auth
Logd authenticates clients for either reading or writing using shared that could be named `LOGD_READ_SECRET` and `LOGD_WRITE_SECRET`. These are stored encrypted in our secrets SOPS file, and set in AWS Secrets Manager.

## Why shared secrets?
Writing is over UDP only. This will not change because cheap real-time logging is the core offering.

I chose to use hash-based ephemeral message authentication with a very short signature ttl (100ms) to limit the potential for replays. Preventing replays futher is then much easier & less computationally expensive.

# HTTP API
Logd starts a http server.
## GET /
```bash
curl --location "$LOGD_HOST/?limit=10" \
--header "Authorization: $LOGD_READ_SECRET"
```
## GET /info

# UDP
## I'd tell you a joke about UDP, but you might not get it...
Logd is built on Protobuf & UDP.

## Logger
The simplest way to write logs is using the `logger` package.
```go
log, _ := logger.NewLogger(&logger.LoggerConfig{
  // Host:     optional
  // Port:     optional
  WriteSecret: "the-very-secret-secret",
  Env:         "very-productive",
  Svc:         "readme-service",
  Fn:          "ReadmeApp",
  Stdout:      true // also write to stdout
})
log.Log(logger.Info, "this is an example %s", "log message")
```

## Custom integration
Logs are written by connecting to a UDP socket on port `:6102`.
See the following example. Error checks skipped for brevity.
```go
// dial udp
addr, _ := conn.GetAddr("logd.fly.dev")
socket, _ := conn.Dial(addr)

// serialise message using protobuf
payload, err := proto.Marshal(&cmd.Cmd{
		Name: cmd.Name_WRITE,
		Msg: &cmd.Msg{
			T:          timestamppb.Now(),
			Env:        env,
			Svc:        cwmsg.Svc,
			Fn:         cwmsg.Fn,
			Lvl:        &lvl,
			Txt:        &cwmsg.Msg,
			StackTrace: st,
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

