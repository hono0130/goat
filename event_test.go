package goat

import (
	"reflect"
	"testing"
)

type TestEvent struct {
	Event
	value int
}

type TestEventWithPointer struct {
	Event
	ptr *testStruct
}

type testStruct struct {
	value int
}

func TestCloneEvent(t *testing.T) {
	original := &TestEvent{}
	cloned := cloneEvent(original)

	// ポインタの比較（元とクローンが異なるアドレスを持つことを確認）
	if reflect.ValueOf(original).Pointer() == reflect.ValueOf(cloned).Pointer() {
		t.Errorf("Expected different pointer addresses, but got the same: %p", original)
	}

	// 型の比較（元とクローンが同じ型であることを確認）
	if reflect.TypeOf(original) != reflect.TypeOf(cloned) {
		t.Errorf("Expected same type, but got different: %T vs %T", original, cloned)
	}
}
