# アプリの状態（dependency update レコード・評価・改善履歴）を永続化する Firestore。
# T3 の Firestore backend（ANNEAL_STORE_BACKEND=firestore）が参照する。
# 状態を失うと PR の重複作成や進行中の更新の取りこぼしにつながるため、削除保護を有効にする。
resource "google_firestore_database" "default" {
  name        = "(default)"
  location_id = "asia-northeast1"
  type        = "FIRESTORE_NATIVE"

  # 誤操作による DB 削除を防ぐ。destroy も拒否する。
  delete_protection_state = "DELETE_PROTECTION_ENABLED"
  deletion_policy         = "DELETE_PROTECTION"
}
