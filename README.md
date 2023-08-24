# dal

- Created: 2023/08/24
- Discord API v9,v10

## Description

- AWS lambda用

## 手順

- switch optionValueで選択肢ごとの挙動記載
- `make install`でzip用toolのインストール
- `make build`で作成したzipをlambdaにアップロード
- Lambda作成
- Lambdaのランタイム設定からハンドラをdalに修正
    - ログに`no such file or directory`が出たときは恐らくこの設定漏れ
- API GatewayでREST API作成
    - メソッドをPOSTで作成
        - Lambda プロキシ統合の使用にチェックを入れる
    - APIのデプロイ
