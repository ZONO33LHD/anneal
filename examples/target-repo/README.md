# 対象リポジトリを Anneal 管理下に置く

Anneal に依存更新を任せたいリポジトリへ、このディレクトリのワークフローをコピーする。
運用の全体像は [docs/how-to/manage-projects.md](../../docs/how-to/manage-projects.md) を参照。

## 手順

1. `.github/workflows/anneal.yml`（このディレクトリのもの）を対象リポジトリの
   同じパスにコピーする。
2. 対象リポジトリに設定を追加する:
   - **Variable** `ANNEAL_URL` = Anneal の Cloud Run URL
   - **Secret** `ANNEAL_TOKEN` = 内部エンドポイントの共有トークン（Anneal 側の
     `ANNEAL_INTERNAL_TOKEN` と同じ値）
3. （任意）除外パッケージや更新パラメータを変えたい場合は、対象リポジトリのルートに
   `.anneal.yml` を置く（`ignore` / `auto_pr_types` / `regression_window_days` /
   `base_branch`）。**無ければ**組み込みの既定ポリシー（patch/minor 自動 PR・除外なし・
   main 向け・7 日監視）が適用される。**あるが壊れている / 取得に失敗した場合は
   エラーで停止**する（意図した除外や設定を黙って既定へ落とさないため）。
4. PR → merge。以降、ワークフローの cron（または手動実行）ごとに Anneal がこの repo を
   スキャンし、更新候補があれば PR を作る。

## 仕組み

ワークフローは Anneal の `POST /internal/scan` に `{"repository": "<owner>/<repo>"}` を
送るだけ。Anneal（Cloud Run）は GitHub Contents API でこの repo のマニフェスト
（package.json / go.mod ＋ lockfile）と `.anneal.yml` を取得してスキャンする
（clone しない）。Anneal 側に監視対象の中央一覧は持たず、このワークフローを置いた
リポジトリだけが対象になる（完全オプトイン）。

## 認証

最小構成は共有トークン（`ANNEAL_TOKEN`）。トークンを配りたくない場合は GitHub OIDC も
使える（`anneal.yml` 末尾の注記、Anneal 側は `ANNEAL_INTERNAL_OIDC_*` で検証）。

> なお、Anneal の **メンテナが自リポジトリから手動で** scan/tick を回す運用ワークフローは
> 別途 `.github/workflows/run-anneal.yml`（env の `ANNEAL_SCAN_TARGETS` 対象・GCP OIDC）に
> ある。本テンプレートは **対象 repo 側に置く push トリガー**で、役割が異なる。
