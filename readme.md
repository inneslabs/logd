# Logd
![A circular buffer](.doc/circular_buffer.svg)

## At first
Read the **onboarding** readme on GITHUB

## Logs for your apps in constant time and constant space.
Logd (pronounced "logged") is a circular buffer for writing & reading millions of logs per minute.

Logd will never run out of memory if the buffer size is ok for the given machine spec. Reads & writes are constant-time.

As the buffer becomes full, each write overwrites the oldest element.

# Auth
Logd authenticates clients for either reading or writing using 2 shared secrets.
These are stored encrypted in our secrets SOPS file.

## Why no SSO?
This is an important design choice for separation of concerns.
This will probably not change.

If you want to put it behind SSO, just write a proxy. ;)

This is a simple, performant & usable service that
you can, and should, build on.

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
Logd is built on Protobuf.

## Logger
The simplest way to write logs is using the `log` package.
```go
l, _ := log.NewLogger(&log.LoggerConfig{
  Host:        "logd.fly.dev",
  WriteSecret: "the-secret",
  Env:         "prod",
  Svc:         "example-service",
  Fn:          "Readme",
})
l.Log(log.Info, "this is an example %s", "log message")
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
Logd is built on protobuf.
```bash
protoc --go_out=. cmd.proto # generate source files
```
