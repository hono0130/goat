# E2E Testing for Protocol Buffers

この機能は、Protocol Buffer生成用のgoatの記述を用いて、e2eテストを記録し、Goのテストコードを自動生成します。

## 概要

この機能により、以下のことが可能になります：

1. **トレース記録**: RPC呼び出しとその入出力を記録
2. **Goテストコード生成**: 記録したトレースからGoの`_test.go`ファイルを自動生成 **← NEW!**
3. **JSON保存**: 記録したトレースをJSONファイルとして保存（言語非依存の中間フォーマット）
4. **テスト再生**: 保存したテストケースを再実行して、出力が期待通りか検証

## 主な機能

### 1. トレース記録 (Trace Recording)

RPCメッセージとレスポンスのペアを記録します。

```go
// トレースレコーダーを作成
recorder := protobuf.NewTraceRecorder()

// RPC呼び出しを記録
err := recorder.RecordRPC(
    "CreateUser",           // メソッド名
    userService,            // 送信者
    userService,            // 受信者
    &CreateUserRequest{...},// 入力
    &CreateUserResponse{...},// 出力
    worldID,                // ワールドID
)

// テストケースとして変換
testCase := recorder.ToTestCase("test_name", "test description")
```

### 2. E2Eテストレコーダー (E2E Test Recorder)

簡単にテストを記録・保存できるヘルパークラスです。

```go
// E2Eテストレコーダーを作成
recorder := protobuf.NewE2ETestRecorder(
    "user_service_test",
    "Test user creation and retrieval",
)

// イベント型を登録
recorder.RegisterEventType(&CreateUserRequest{})
recorder.RegisterEventType(&CreateUserResponse{})

// RPC呼び出しを記録
recorder.Record("CreateUser", userService, userService,
    &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
    &CreateUserResponse{UserID: "123", Success: true},
    1)

// ファイルに保存
recorder.SaveToFile("testdata/user_service_test.json")
```

### 3. Goテストコード生成 (Go Test Code Generation) **← NEW!**

記録したトレースから、Goのテストコード（`_test.go`ファイル）を自動生成します。

```go
// E2Eテストレコーダーを作成
recorder := protobuf.NewE2ETestRecorder(
    "user_service_e2e",
    "E2E tests for user service",
)

// イベント型を登録
recorder.RegisterEventType(&CreateUserRequest{})
recorder.RegisterEventType(&CreateUserResponse{})

// RPC呼び出しを記録
recorder.Record("CreateUser", userService, userService,
    &CreateUserRequest{Username: "alice", Email: "alice@example.com"},
    &CreateUserResponse{UserID: "123", Success: true},
    1)

// Goのテストコードを生成
code, err := recorder.GenerateGoTest("main")
if err != nil {
    log.Fatal(err)
}

// ファイルに保存
os.WriteFile("user_service_test.go", []byte(code), 0644)

// または、直接ファイルに生成
err = recorder.GenerateGoTestToFile("main", "user_service_test.go")
```

**生成されるコード：**

```go
package main

import (
	"reflect"
	"testing"
)

// TestUser_service_e2e_0_CreateUser tests the CreateUser RPC call.
// This test was automatically generated from recorded trace.
func TestUser_service_e2e_0_CreateUser(t *testing.T) {
	// Input: CreateUserRequest
	input := &CreateUserRequest{
		Email:    "alice@example.com",
		Username: "alice",
	}

	// Expected output: CreateUserResponse
	expected := &CreateUserResponse{
		Success: true,
		UserID:  "123",
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

生成されたテストコードは、実際のRPC実行部分（`// TODO:`の部分）を実装することで、完全に動作するテストになります。

### 4. テスト再生 (Test Replay)

保存したテストケースを読み込んで再生し、出力を検証します。

```go
// テストランナーを作成
runner := protobuf.NewE2ETestRunner()

// イベント型を登録
runner.RegisterEventType(&CreateUserRequest{})
runner.RegisterEventType(&CreateUserResponse{})

// ステートマシンを登録
runner.RegisterStateMachine("UserService", userService)

// ストリクトモードを有効化
runner.SetStrictMode(true)

// テストを実行
result, err := runner.RunFromFile("testdata/user_service_test.json")
if err != nil {
    log.Fatal(err)
}

// 結果を表示
fmt.Println(result.Summary())

if result.FailureCount > 0 {
    fmt.Println("❌ Test FAILED")
} else {
    fmt.Println("✅ Test PASSED")
}
```

### 5. クイック記録 (Quick Record)

単一のRPC呼び出しを簡単に記録する便利関数です。

```go
testCase := protobuf.QuickRecord(
    "quick_test",
    "CreateUser",
    userService,
    &CreateUserRequest{Username: "bob", Email: "bob@example.com"},
    &CreateUserResponse{UserID: "456", Success: true},
)

// JSONに保存
data, _ := protobuf.SaveTestCase(testCase)
os.WriteFile("quick_test.json", data, 0600)

// または、Goテストコードを生成
generator := protobuf.NewGoTestGenerator("main")
code, _ := generator.Generate(testCase)
os.WriteFile("quick_test.go", []byte(code), 0644)
```

## 使用例

### 完全な例

```go
package main

import (
    "fmt"
    "log"

    "github.com/goatx/goat/protobuf"
)

// 1. テストの記録
func recordTest() {
    recorder := protobuf.NewE2ETestRecorder(
        "user_creation_test",
        "Test user creation flow",
    )

    // イベント型を登録
    recorder.RegisterEventType(&CreateUserRequest{})
    recorder.RegisterEventType(&CreateUserResponse{})

    // RPCを記録
    userService := &UserService{}
    recorder.Record("CreateUser", userService, userService,
        &CreateUserRequest{
            Username: "alice",
            Email:    "alice@example.com",
        },
        &CreateUserResponse{
            UserID:  "user_123",
            Success: true,
        },
        1,
    )

    // 保存
    recorder.SaveToFile("testdata/user_creation_test.json")
    fmt.Println("✓ Test recorded and saved")
}

// 2. テストの再生
func replayTest() {
    runner := protobuf.NewE2ETestRunner()

    // イベント型を登録
    runner.RegisterEventType(&CreateUserRequest{})
    runner.RegisterEventType(&CreateUserResponse{})

    // ステートマシンを登録
    runner.RegisterStateMachine("UserService", &UserService{})

    // テストを実行
    result, err := runner.RunFromFile("testdata/user_creation_test.json")
    if err != nil {
        log.Fatal(err)
    }

    // 結果を表示
    fmt.Println(result.Summary())
}
```

### 自動トレース記録

contextを使って自動的にトレースを記録することもできます：

```go
// レコーダーを作成
recorder := protobuf.NewE2ETestRecorder("auto_test", "")

// トレース記録用のcontextを取得
ctx := recorder.GetContext(context.Background(), worldID)

// このcontextでRPCを呼び出すと、自動的に記録されます
// （OnProtobufMessageで登録されたハンドラー内で）
```

## テストケースのフォーマット

テストケースはJSON形式で保存されます：

```json
{
  "name": "user_service_test",
  "description": "Test user creation and retrieval",
  "traces": [
    {
      "method_name": "CreateUser",
      "sender": "UserService@0x...",
      "recipient": "UserService@0x...",
      "input_type": "CreateUserRequest",
      "input": {
        "Username": "alice",
        "Email": "alice@example.com"
      },
      "output_type": "CreateUserResponse",
      "output": {
        "UserID": "user_123",
        "Success": true,
        "ErrorCode": 0
      },
      "world_id": 1
    }
  ]
}
```

## API リファレンス

### TraceRecorder

- `NewTraceRecorder()` - 新しいトレースレコーダーを作成
- `RecordRPC(methodName, sender, recipient, input, output, worldID)` - RPC呼び出しを記録
- `GetTraces()` - 記録されたトレースを取得
- `Clear()` - 記録をクリア
- `ToTestCase(name, description)` - テストケースに変換

### E2ETestRecorder

- `NewE2ETestRecorder(name, description)` - 新しいE2Eテストレコーダーを作成
- `RegisterEventType(event)` - イベント型を登録
- `Record(methodName, sender, recipient, input, output, worldID)` - RPC呼び出しを記録
- `GetContext(ctx, worldID)` - トレース記録用のcontextを取得
- `SaveToFile(filepath)` - テストケースをJSONファイルに保存
- `GetTestCase()` - 現在のテストケースを取得
- **`GenerateGoTest(packageName)`** - Goのテストコードを生成 **← NEW!**
- **`GenerateGoTestToFile(packageName, filepath)`** - Goのテストコードをファイルに生成 **← NEW!**

### GoTestGenerator **← NEW!**

- `NewGoTestGenerator(packageName)` - 新しいGoテストコードジェネレーターを作成
- `AddImport(importPath)` - カスタムimportを追加
- `Generate(testCase)` - テストケースからGoコードを生成
- `GenerateToFile(testCase, filepath)` - Goコードをファイルに生成

### E2ETestRunner

- `NewE2ETestRunner()` - 新しいE2Eテストランナーを作成
- `RegisterEventType(event)` - イベント型を登録
- `RegisterStateMachine(id, sm)` - ステートマシンを登録
- `SetStrictMode(strict)` - ストリクトモードを設定
- `RunFromFile(filepath)` - ファイルからテストを実行
- `Run(testCase)` - テストケースを実行

### ヘルパー関数

- `QuickRecord(testName, methodName, target, input, output)` - 単一のRPC呼び出しを簡単に記録
- `SaveTestCase(testCase)` - テストケースをJSONに変換
- `LoadTestCase(data)` - JSONからテストケースを読み込み

## モデル検査との統合

モデル検査を実行する際に、自動的にトレースを記録することができます。これには、contextにTraceRecorderを埋め込む必要があります。

現在の実装では、`OnProtobufMessage`で登録されたハンドラーは、contextにTraceRecorderが含まれている場合、自動的にトレースを記録します。

```go
// カスタムのモデル検査実装で、contextにレコーダーを追加
ctx = protobuf.WithTraceRecorder(ctx, recorder)
ctx = protobuf.WithWorldID(ctx, worldID)
```

## テスト

すべての機能は包括的なユニットテストでカバーされています：

```bash
go test ./protobuf/... -v
```

## 今後の拡張

- モデル検査との完全な統合
- より高度なアサーション機能
- テストカバレッジレポート
- パフォーマンステスト

## ライセンス

このプロジェクトはgoatプロジェクトの一部です。
