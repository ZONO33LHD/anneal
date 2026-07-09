package gateway

import "github.com/ZONO33LHD/anneal/domain/model"

// RepoConfigLoader はリポジトリごとの .anneal.yml を読み取る。
//   - ファイルが存在しない場合は組み込み既定（DefaultRepoConfig）を error なしで返す。
//   - ファイルはあるが壊れている / 取得に失敗した場合は error を返す（意図した
//     ポリシーを黙って既定へ落とし、除外や設定を握りつぶさないため）。
type RepoConfigLoader interface {
	Load(repoRef string) (model.RepoConfig, error)
}
