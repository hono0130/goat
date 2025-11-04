# E2E Testing for Protocol Buffers

Protocol Buffer生成用のgoatの記述を用いて、e2eテストコードを自動生成します。

## 概要

この機能により、以下のことが可能になります：

1. **テスト入力の指定**: 実際に値が入っているイベントをテストケースとして指定
2. **期待値の自動計算**: ハンドラを実行して出力（期待値）を自動的に取得
3. **Goテストコード生成**: 入出力ペアからGoの`_test.go`ファイルを自動生成

## 使用方法

### 基本的な使い方

```go
package main

import (
	"github.com/goatx/goat/protobuf"
)

func main() {
	// E2Eテストを生成
	err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
		OutputDir:   "./tests",
		PackageName: "main",
		Filename:    "user_service_e2e_test.go",
		TestCases: []protobuf.TestCase{
			{
				MethodName: "CreateUser",
				// テスト入力（実際に値が入ったイベント）
				Input: &CreateUserRequest{
					Username: "alice",
					Email:    "alice@example.com",
				},
				// 期待値を取得する関数
				// ここでハンドラを実行して出力を取得
				GetOutput: func() (protobuf.AbstractProtobufMessage, error) {
					// ハンドラを実行
					return &CreateUserResponse{
						UserID:  "user_123",
						Success: true,
					}, nil
				},
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
err := protobuf.GenerateE2ETest(protobuf.E2ETestOptions{
	OutputDir:   "./tests",
	PackageName: "main",
	Filename:    "user_service_e2e_test.go",
	TestCases: []protobuf.TestCase{
		{
			MethodName: "CreateUser",
			Input:      &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
			GetOutput: func() (protobuf.AbstractProtobufMessage, error) {
				return &CreateUserResponse{UserID: "user_123", Success: true}, nil
			},
		},
		{
			MethodName: "GetUser",
			Input:      &GetUserRequest{UserID: "user_123"},
			GetOutput: func() (protobuf.AbstractProtobufMessage, error) {
				return &GetUserResponse{Username: "alice", Email: "alice@example.com", Found: true}, nil
			},
		},
	},
})
```

## API リファレンス

### GenerateE2ETest

```go
func GenerateE2ETest(opts E2ETestOptions) error
```

E2Eテストコードを生成します。

**パラメータ:**
- `opts.OutputDir`: 生成したテストファイルを保存するディレクトリ（デフォルト: "./tests"）
- `opts.PackageName`: 生成するテストのパッケージ名（デフォルト: "main"）
- `opts.Filename`: 生成するファイル名（デフォルト: "generated_e2e_test.go"）
- `opts.TestCases`: テストケースのリスト

### TestCase

```go
type TestCase struct {
	MethodName string
	Input      AbstractProtobufMessage
	GetOutput  func() (AbstractProtobufMessage, error)
}
```

単一のテストケースを表します。

**フィールド:**
- `MethodName`: テストするRPCメソッド名
- `Input`: 実際に値が入った入力イベント
- `GetOutput`: ハンドラを実行して出力を取得する関数

### E2ETestOptions

```go
type E2ETestOptions struct {
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
protobuf.GenerateE2ETest(opts)
```

### 入力と出力の分離

- **入力**: ユーザーが明示的に指定（実際に値が入ったイベント）
- **出力**: `GetOutput()` 関数を実行して自動的に取得

これにより、モデル検査のハンドラを実行すれば期待値がわかるという設計思想を実現しています。

## 今後の拡張

- モデル検査との統合: `spec`からハンドラを自動抽出して実行
- 他言語対応: Python、Rustなどのテストコード生成

## ライセンス

このプロジェクトはgoatプロジェクトの一部です。
