package store

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestDeviceStore_AddToken(t *testing.T) {
	store := NewDeviceStore()

	token1 := "test_token_1"
	token2 := "test_token_2"

	// 1. 新しいトークンを追加
	added := store.AddToken(token1)
	if !added {
		t.Errorf("AddToken(%q) = %v, want %v (should be newly added)", token1, added, true)
	}
	if len(store.tokens) != 1 {
		t.Errorf("len(store.tokens) after adding %q = %d, want %d", token1, len(store.tokens), 1)
	}

	// 2. 既に存在するトークンを再度追加 (重複)
	added = store.AddToken(token1)
	if added {
		t.Errorf("AddToken(%q) = %v, want %v (should indicate already exists)", token1, added, false)
	}
	if len(store.tokens) != 1 {
		t.Errorf("len(store.tokens) after attempting to add duplicate %q = %d, want %d", token1, len(store.tokens), 1)
	}

	// 3. 別の新しいトークンを追加
	added = store.AddToken(token2)
	if !added {
		t.Errorf("AddToken(%q) = %v, want %v (should be newly added)", token2, added, true)
	}
	if len(store.tokens) != 2 {
		t.Errorf("len(store.tokens) after adding %q = %d, want %d", token2, len(store.tokens), 2)
	}
}

func TestDeviceStore_GetTokens(t *testing.T) {
	store := NewDeviceStore()
	tokensToAdd := []string{"token1", "token2", "token3"}

	// 0トークンの場合
	if got := store.GetTokens(); len(got) != 0 {
		t.Errorf("GetTokens() on empty store = %v, want empty slice", got)
	}

	for _, token := range tokensToAdd {
		store.AddToken(token)
	}

	retrievedTokens := store.GetTokens()
	if len(retrievedTokens) != len(tokensToAdd) {
		t.Errorf("len(GetTokens()) = %d, want %d", len(retrievedTokens), len(tokensToAdd))
	}

	// スライスの内容を比較するためにはソートする
	sort.Strings(retrievedTokens)
	sort.Strings(tokensToAdd)

	if !reflect.DeepEqual(retrievedTokens, tokensToAdd) {
		t.Errorf("GetTokens() = %v, want %v", retrievedTokens, tokensToAdd)
	}
}

func TestDeviceStore_RemoveToken(t *testing.T) {
	store := NewDeviceStore()
	token1 := "test_token_to_remove_1"
	token2 := "test_token_to_keep_1"

	store.AddToken(token1)
	store.AddToken(token2)

	if len(store.tokens) != 2 {
		t.Fatalf("Initial store size incorrect before removal test.")
	}

	// 存在するトークンを削除
	store.RemoveToken(token1)
	if len(store.tokens) != 1 {
		t.Errorf("len(store.tokens) after removing %q = %d, want %d", token1, len(store.tokens), 1)
	}
	if _, exists := store.tokens[token1]; exists {
		t.Errorf("token %q should have been removed, but still exists", token1)
	}
	if _, exists := store.tokens[token2]; !exists {
		t.Errorf("token %q should not have been removed, but was", token2)
	}

	// 存在しないトークンを削除 (何もしないはず)
	store.RemoveToken("non_existent_token")
	if len(store.tokens) != 1 {
		t.Errorf("len(store.tokens) after attempting to remove non_existent_token = %d, want %d", len(store.tokens), 1)
	}
}

func TestDeviceStore_Concurrency(t *testing.T) {
	store := NewDeviceStore()
	numGoroutines := 100
	tokensPerGoroutine := 10
	expectedTotalTokens := numGoroutines * tokensPerGoroutine

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// 各ゴルーチンでAddTokenを呼び出す
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < tokensPerGoroutine; j++ {
				token := fmt.Sprintf("token_g%d_t%d", goroutineID, j)
				store.AddToken(token)
			}
		}(i)
	}

	// 全てのゴルーチンが終了するのを待つ
	wg.Wait()

	// GetTokensを呼び出してクラッシュしないか、期待される数のトークンがあるか確認
	allTokens := store.GetTokens()
	if len(allTokens) != expectedTotalTokens {
		t.Errorf("Total tokens after concurrent adds = %d, want %d", len(allTokens), expectedTotalTokens)
	}

	// RemoveTokenの並行処理テスト（一部のトークンを削除）
	tokensToRemovePerGoroutine := 5
	expectedTokensAfterRemove := expectedTotalTokens - (numGoroutines * tokensToRemovePerGoroutine)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < tokensToRemovePerGoroutine; j++ {
				// 削除するトークンは、追加したトークンと同じ命名規則で、
				// 各ゴルーチンが担当したトークンの一部を削除するようにする
				token := fmt.Sprintf("token_g%d_t%d", goroutineID, j)
				store.RemoveToken(token)
			}
		}(i)
	}
	wg.Wait()

	remainingTokens := store.GetTokens()
	if len(remainingTokens) != expectedTokensAfterRemove {
		t.Errorf("Total tokens after concurrent removes = %d, want %d", len(remainingTokens), expectedTokensAfterRemove)
	}
}

// time パッケージは Concurrency テストでは直接使わなくなったが、
// 他のテストで必要になる可能性もあるので残しても良い。
// 今回は不要なので削除も検討できる。
// import "time" // ← Concurrencyテストで直接は不要になったが、他で使う可能性を考慮して残すか判断
// 上記の通り、timeパッケージは現在のテストコードでは不要なので削除しました。
