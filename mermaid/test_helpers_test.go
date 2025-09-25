package mermaid

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeWorkflowFixture(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	moduleRoot, err := findModuleRoot(filepath.Dir(filename))
	if err != nil {
		t.Fatalf("findModuleRoot returned error: %v", err)
	}

	dir, err := os.MkdirTemp(moduleRoot, "workflow")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	const source = `package workflow

import (
    "context"

    "github.com/goatx/goat"
)

type (
    senderState struct{ goat.State }
    receiverState struct{ goat.State }
    loggerState struct{ goat.State }
)

type (
    PingEvent struct{ goat.Event }
    AckEvent struct{ goat.Event }
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
`

	if err := os.WriteFile(filepath.Join(dir, "workflow.go"), []byte(source), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	return dir
}

func loadWorkflowPackage(t *testing.T) *packageInfo {
	t.Helper()

	dir := writeWorkflowFixture(t)

	pkg, err := loadPackageWithTypes(dir)
	if err != nil {
		t.Fatalf("loadPackageWithTypes returned error: %v", err)
	}
	return pkg
}
