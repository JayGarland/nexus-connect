module github.com/JayGarland/nexus-connect

go 1.25.0

require github.com/chenhg5/cc-connect v1.3.3

replace github.com/chenhg5/cc-connect => ../cc-connect-nexus

require (
	github.com/go-telegram/bot v1.23.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
)
