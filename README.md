# dal

- Created: 2023/08/24
- Discord API v9,v10

## Description

- AWS Lambda用

## 手順

1. switch optionValueで選択肢ごとの挙動記載
2. `make install`でzip用toolのインストール
3. `make build`で作成したzipをLambdaにアップロード
4. Lambda作成
5. Lambdaのランタイム設定からハンドラをdalに修正
    - ログに`no such file or directory`が出たときは恐らくこの設定漏れ
6. API GatewayでREST API作成
    - メソッドをPOSTで作成
        - Lambda プロキシ統合の使用にチェックを入れる
    - APIのデプロイ
