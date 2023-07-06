package mocks

//go:generate mockgen -destination gen/mockclock/mocks.go -package mockclock github.com/xiaoxu5271/can-go/internal/clock Clock,Ticker
//go:generate mockgen -destination gen/mocksocketcan/mocks.go -package mocksocketcan -source ../../pkg/socketcan/fileconn.go
//go:generate mockgen -destination gen/mockcanrunner/mocks.go -package mockcanrunner github.com/xiaoxu5271/can-go/pkg/canrunner Node,TransmittedMessage,ReceivedMessage,FrameTransmitter,FrameReceiver
