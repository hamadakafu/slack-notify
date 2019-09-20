# Memo
## kubebuilder
[kubebuilder](https://book.kubebuilder.io/introduction.html)

## 使い方
### CRDの定義
```bash
kubectl apply -f example/0crd.yaml
vim example/sample-slack-notify.yaml # incoming webhookのurlを設定
kubectl apply -f example/sample-slack-notify.yaml
```

## 開発
### コントローラのビルド
```bash
make docker-build docker-push IMG=<some-registry>/controller
```

### コントローラのデプロイ
```bash
make deploy IMG=<some-registry>/controller
```

### SlackNotifyのデプロイ
```bash
kubectl apply -f example/sample-slack-notify.yaml
```

# 備考
## テスト実行すると失敗するのでスキップしている
## JSONPATHではなくJSONPath kubebuilder のアノテーション
