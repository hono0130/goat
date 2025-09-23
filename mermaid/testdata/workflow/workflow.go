package workflow

import (
	"context"

	"github.com/goatx/goat"
)

type (
	senderState   struct{ goat.State }
	receiverState struct{ goat.State }
	loggerState   struct{ goat.State }
)

type (
	PingEvent   struct{ goat.Event }
	AckEvent    struct{ goat.Event }
	NotifyEvent struct{ goat.Event }
)

type Sender struct {
	goat.StateMachine
	Receiver *Receiver
}

type Receiver struct {
	goat.StateMachine
	Sender *Sender
	Logger *Logger
}

type Logger struct {
	goat.StateMachine
}

func Configure() {
	senderSpec := goat.NewStateMachineSpec(&Sender{})
	receiverSpec := goat.NewStateMachineSpec(&Receiver{})
	loggerSpec := goat.NewStateMachineSpec(&Logger{})

	senderIdle := &senderState{}
	receiverIdle := &receiverState{}
	loggerIdle := &loggerState{}

	senderSpec.DefineStates(senderIdle).SetInitialState(senderIdle)
	receiverSpec.DefineStates(receiverIdle).SetInitialState(receiverIdle)
	loggerSpec.DefineStates(loggerIdle).SetInitialState(loggerIdle)

	goat.OnEntry(senderSpec, senderIdle, func(ctx context.Context, sm *Sender) {
		goat.SendTo(ctx, sm.Receiver, &PingEvent{})
	})

	goat.OnEvent(receiverSpec, receiverIdle, &PingEvent{}, func(ctx context.Context, event *PingEvent, sm *Receiver) {
		goat.SendTo(ctx, sm.Sender, &AckEvent{})
		goat.SendTo(ctx, sm.Logger, &NotifyEvent{})
	})

	goat.OnEvent(senderSpec, senderIdle, &AckEvent{}, func(ctx context.Context, event *AckEvent, sm *Sender) {
		goat.SendTo(ctx, sm.Receiver, &NotifyEvent{})
	})

	goat.OnEvent(loggerSpec, loggerIdle, &NotifyEvent{}, func(ctx context.Context, event *NotifyEvent, sm *Logger) {})
}
