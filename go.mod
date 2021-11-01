module github.com/rtrox/reddit-notifier

go 1.16

require (
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/vartanbeno/go-reddit/v2 v2.0.1
)

replace github.com/vartanbeno/go-reddit/v2 => github.com/rtrox/go-reddit/v2 v2.0.1-0.20211031224903-bfb8c6b683b5
