# go-north

[north](https://north.rip) APIの非公式Goクライアントです。

> [!WARNING]
>
> - north公式のSDKではありません。
> - `nth_live_`で始まる旧APIキーでは、新しく追加されたAPIを利用できません。

API v0.56.1の全エンドポイントに対応しています。

## インストール

```bash
go get github.com/Hayao0819/go-north
```

Go 1.23以上が必要です。

## 使い方

northの「設定 › 開発者向け」で個人トークンを発行します。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Hayao0819/go-north"
)

func main() {
	client, err := north.NewClient(os.Getenv("NORTH_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	page, _, err := client.HomeTimeline(context.Background(), north.TimelineOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, post := range page.Items {
		fmt.Printf("@%s: %s\n", post.Author.Handle, post.Text)
	}
}
```

## OAuth

配布アプリではDevice FlowまたはPKCEを使います。クライアントシークレットは使用しません。

```go
func connect(ctx context.Context, save north.SaveTokenFunc) (*north.Client, error) {
	config := north.NewOAuthConfig(
		os.Getenv("NORTH_CLIENT_ID"),
		"",
		north.ScopePostsRead,
		north.ScopeUsersRead,
	)
	authorization, err := config.DeviceAuth(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s で %s を入力してください\n", authorization.VerificationURI, authorization.UserCode)

	token, err := config.DeviceAccessToken(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if save != nil {
		if err := save(token); err != nil {
			return nil, err
		}
	}
	return north.NewClientWithOAuth(ctx, config, token, save)
}
```

`save`には取得時と更新時のトークンをキーリング等へ原子的に保存する関数を渡します。不要なら`nil`も指定できます。
PKCEでは同じ設定を`golang.org/x/oauth2`の`GenerateVerifier`、`AuthCodeURL`、`Exchange`と共に使います。
OAuthアクセストークンと個人トークンはどちらも`nth_oat_`で始まります。

仕様は[north APIドキュメント](https://api.north.rip/docs)を参照してください。

## 開発

```bash
direnv allow
nix fmt
nix flake check
```

本番APIへの読み取りテストは、`NORTH_API_KEY`を設定したうえで明示的に実行します。

```bash
go test -tags=integration ./...
```

## License

MIT
