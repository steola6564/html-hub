# HTML 書架

`pages/` に置いた HTML を、目録（`/`）から開けます。各ページの URL は `/pages/ファイル名` です。Go が目録を開くたびにファイル一覧を読み、`index.html` に渡します。

## Mac から PVE に転送する

**前の版を設置済みの場合**は、Mac 側で更新する2ファイルを転送します。

```sh
scp main.go README.md steola@100.71.71.36:html-hub/
```

**初めて設置する場合**は、3ファイルと手元の読本を同じフォルダーに置き、Mac 側で次を実行します。

```sh
ssh steola@100.71.71.36 'mkdir -p ~/html-hub/pages'
scp main.go index.html README.md steola@100.71.71.36:html-hub/
scp Red_Dot_Reader.html steola@100.71.71.36:html-hub/pages/
```

## PVE でビルドして起動する

```sh
ssh steola@100.71.71.36
cd ~/html-hub
go build -o html-hub main.go
```

Linux と Go 1.16 以降の標準ライブラリを使います。ビルド時は `main.go` と `index.html` を同じディレクトリに置いてください。この版への更新時はビルドが一度必要です。

```sh
./html-hub
```

初回は作業ディレクトリに `config.conf` を自動作成して、**8090 番**で待ち受けます。すでに設定があればその値を使います。`config.conf` と `pages/` は作業ディレクトリを基準に読むので、毎回 `~/html-hub` で起動してください。手動起動していた旧版は Ctrl+C で止めてから起動します。後述のユーザーサービスで稼働中の場合は、ビルド後に `systemctl --user restart html-hub` を実行します。

| 端末 | 既定の目録 URL |
| --- | --- |
| Tailscale に接続した端末 | `http://100.71.71.36:8090/` |
| 自宅の同一 LAN 上の端末 | `http://192.168.11.26:8090/` |
| PVE 自身 | `http://127.0.0.1:8090/` |

読本へ直接アクセスする場合は、たとえば `http://100.71.71.36:8090/pages/Red_Dot_Reader.html` を開きます。

## ポートを変更する

`~/html-hub/config.conf` の内容は **1行**です。

```ini
port=8090
```

番号を使いたい空きポート（1〜65535 の十進整数）に変更し、サーバーを再起動します。たとえば 9000 番を使う場合、手動起動中なら Ctrl+C で停止して次を実行します。初回起動の前に同じ手順で設定を作ることもできます。

```sh
cd ~/html-hub
printf 'port=9000\n' > config.conf
./html-hub
```

以降も起動コマンドは `./html-hub` です。再ビルドは不要です。上の例なら接続先は `http://100.71.71.36:9000/` または `http://192.168.11.26:9000/` になります。ユーザーサービスで稼働中なら、設定を保存した後、最後の行の代わりに次を実行します。

```sh
systemctl --user restart html-hub
```

通常は全インターフェースで待ち受け、到達できる端末から認証なしで閲覧できます。1回だけ待受先を指定する場合には `./html-hub -addr 100.71.71.36:8090` のように起動します。この起動では `-addr` が優先され、保存済みの設定はそのままです。

## ページを追加する

Mac 側で次を実行して、目録を再読み込みします。

```sh
scp Another_Reader.html steola@100.71.71.36:html-hub/pages/
```

- `pages/` 直下の通常ファイル（`.html`・`.htm`、大文字小文字どちらも可）が対象です。CSS・JavaScript・画像を同梱する場合は HTML 内に埋め込みます。
- ファイル名順に並び、`Red_Dot_Reader.html` の見出しは **Red Dot Reader** になります。隠しファイル、サブディレクトリ、シンボリックリンクは対象外です。
- ページは稼働中に追加・差し替え・削除できます。目録や本文を再読み込みすると反映されます。目録のデザイン自体を変更するときは `index.html` を編集し、再ビルドして起動し直してください。

## 接続を確認する

次の例は既定の 8090 番を使います。変更した場合は2か所とも設定した番号に置き換えてください。PVE の別の SSH セッションで、待受と読本の応答（`200 OK`）を確認できます。

```sh
ss -ltn 'sport = :8090'
curl -I http://127.0.0.1:8090/pages/Red_Dot_Reader.html
```

PVE 内で応答があり、別端末から接続できない場合は、接続元から設定したポートに到達できるか、PVE のファイアウォールや Tailscale のアクセスルールを確認してください。

## SSH を閉じた後も動かす

手動起動中なら Ctrl+C で止めてから、PVE の `steola` ユーザーで実行します。最初のコマンドはログアウト後もユーザーサービスを維持する設定です。権限を求められた場合は管理者が `loginctl enable-linger steola` を一度実行してください。

```sh
loginctl enable-linger steola
cd ~/html-hub
systemd-run --user --unit=html-hub --collect --working-directory="$PWD" "$PWD/html-hub"
```

稼働状況、ログ、停止は次のコマンドで操作します。

```sh
systemctl --user status html-hub
journalctl --user -u html-hub -n 30
systemctl --user stop html-hub
```

この起動方法は一時サービスを使います。PVE の再起動後やサービスを停止した後は、`~/html-hub` で上の `systemd-run` をもう一度実行してください。
