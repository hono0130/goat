# E2E Testing for Protocol Buffers

Protocol Buffer生成用のgoatの記述を用いて、e2eテストコードを自動生成します。

## 概要

この機能により、以下のことが可能になります：

1. **テスト入力の指定**: 実際に値が入っているイベントをテストケースとして指定
2. **期待値の自動計算**: 登録されたハンドラを自動実行して出力（期待値）を取得
3. **Goテストコード生成**: 入出力ペアからGoの`_test.go`ファイルを自動生成

## 使用方法

### 基本的な使い方

```go
package main

import (
	"context"
	"github.com/goatx/goat"
	"github.com/goatx/goat/protobuf"
)

func main() {
	// サービス仕様を作成
	spec := protobuf.NewProtobufServiceSpec(&UserService{})
	idleState := &IdleState{}

	// 状態を定義
	spec.DefineStates(idleState).SetInitialState(idleState)

	// ハンドラを登録
	protobuf.OnProtobufMessage(spec, idleState, "CreateUser",
		&CreateUserRequest{}, &CreateUserResponse{},
		func(ctx context.Context, req *CreateUserRequest, svc *UserService) protobuf.ProtobufResponse[*CreateUserResponse] {
			return protobuf.ProtobufSendTo(ctx, svc, &CreateUserResponse{
				UserID:  "user_123",
				Success: true,
			})
		})

	// E2Eテストを生成 - 入力のみ指定、出力は自動計算
	err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
		Spec:        spec,  // ハンドラが登録されたspec
		OutputDir:   "./tests",
		PackageName: "main",
		Filename:    "user_service_e2e_test.go",
		TestCases: []protobuf.TestCase{
			{
				MethodName: "CreateUser",
				// テスト入力のみ指定（実際に値が入ったイベント）
				Input: &CreateUserRequest{
					Username: "alice",
					Email:    "alice@example.com",
				},
				// GetOutput は不要！ハンドラが自動実行される
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
```

### 生成されるコード

上記のコードを実行すると、以下のようなGoテストコードが生成されます：

```go
package main

import (
	"reflect"
	"testing"
)

// TestCreateUser_0 tests the CreateUser RPC call.
// This test was automatically generated from model checking execution.
func TestCreateUser_0(t *testing.T) {
	// Input: CreateUserRequest
	input := &CreateUserRequest{
		Username: "alice",
		Email:    "alice@example.com",
	}

	// Expected output: CreateUserResponse
	expected := &CreateUserResponse{
		UserID:  "user_123",
		Success: true,
	}

	// TODO: Execute the RPC call to get actual output
	// Example:
	//   service := &UserService{}
	//   ctx := context.Background()
	//   output := service.CreateUser(ctx, input)
	//
	// For now, this is a placeholder that will fail until you implement the execution.
	var output interface{}
	_ = input // Use input when implementing

	// Verify the output matches expected
	if !compareE2EOutput(expected, output) {
		t.Errorf("CreateUser output mismatch:\nexpected: %+v\ngot:      %+v", expected, output)
	}
}

// compareE2EOutput compares two values for equality in E2E tests.
// This is a helper function automatically generated for E2E testing.
func compareE2EOutput(expected, actual interface{}) bool {
	return reflect.DeepEqual(expected, actual)
}
```

### 複数のテストケース

```go
// 複数のハンドラを登録
protobuf.OnProtobufMessage(spec, idleState, "CreateUser",
	&CreateUserRequest{}, &CreateUserResponse{}, createUserHandler)

protobuf.OnProtobufMessage(spec, idleState, "GetUser",
	&GetUserRequest{}, &GetUserResponse{}, getUserHandler)

// 複数のテストケースを生成
err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
	Spec:        spec,
	OutputDir:   "./tests",
	PackageName: "main",
	Filename:    "user_service_e2e_test.go",
	TestCases: []protobuf.TestCase{
		{
			MethodName: "CreateUser",
			Input:      &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
		},
		{
			MethodName: "GetUser",
			Input:      &GetUserRequest{UserID: "user_123"},
		},
	},
})
```

## API リファレンス

### GenerateE2ETest

```go
func GenerateE2ETest(opts E2ETestOptions) error
```

E2Eテストコードを生成します。登録されたハンドラを自動実行して期待値を計算します。

**パラメータ:**
- `opts.Spec`: ハンドラが登録されたProtobufServiceSpec
- `opts.OutputDir`: 生成したテストファイルを保存するディレクトリ（デフォルト: "./tests"）
- `opts.PackageName`: 生成するテストのパッケージ名（デフォルト: "main"）
- `opts.Filename`: 生成するファイル名（デフォルト: "generated_e2e_test.go"）
- `opts.TestCases`: テストケースのリスト

### TestCase

```go
type TestCase struct {
	MethodName string
	Input      AbstractProtobufMessage
}
```

単一のテストケースを表します。

**フィールド:**
- `MethodName`: テストするRPCメソッド名
- `Input`: 実際に値が入った入力イベント

期待される出力は、specに登録されたハンドラを自動実行して計算されます。

### E2ETestOptions

```go
type E2ETestOptions struct {
	Spec        AbstractProtobufServiceSpec
	OutputDir   string
	PackageName string
	Filename    string
	TestCases   []TestCase
}
```

テスト生成のオプションを設定します。

## 設計思想

### GenerateProtobuf と同じスタイル

この機能は `GenerateProtobuf()` と同じスタイルで設計されています：

```go
// Protocol Buffer生成
protobuf.GenerateProtobuf(opts, spec1, spec2)

// E2Eテスト生成
protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
	Spec: spec,
	// ...
})
```

### 入力と出力の分離

- **入力**: ユーザーが明示的に指定（実際に値が入ったイベント）
- **出力**: specに登録されたハンドラを自動実行して取得

ユーザーは**入力のみを指定**します。期待される出力は、`OnProtobufMessage`で登録されたハンドラを内部で実行して自動的に計算されます。

### シンプルな実装

- ハンドラの実行は`spec.NewStateMachineInstance()`でインスタンスを作成
- `goat.NewHandlerContext()`で実行環境を準備
- リフレクションでハンドラを呼び出し
- `ProtobufResponse.GetEvent()`で結果を取得

わずか30行程度のシンプルな実装で実現しています。

## 今後の拡張

- 他言語対応: Python、Rustなどのテストコード生成
- カスタムテンプレート: 生成されるテストコードのカスタマイズ

## ライセンス

このプロジェクトはgoatプロジェクトの一部です。
