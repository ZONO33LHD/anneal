// Package gateway は外部システムやアダプタへのポートを宣言する。
//
// 外部サービスごとにファイルを分けている:
//   - llm.go        … LLM（テキスト生成）
//   - metadata.go   … パッケージの最新版・脆弱性アドバイザリ取得
//   - notify.go     … 通知（Slack / コンソール）
//   - git.go        … git ホスト（PR 作成・CI 確認）
//   - ecosystem.go  … マニフェスト解析・ソース走査
//   - logger.go     … 横断的ロギング
//   - repoconfig.go … リポジトリ単位の設定読み込み
//
// 実装はすべて infrastructure に存在する。
package gateway
