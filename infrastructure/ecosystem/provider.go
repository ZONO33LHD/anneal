package ecosystem

import (
	"context"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
)

type provider struct {
	all []gateway.Ecosystem
}

// NewProvider は、サポート対象のエコシステムを扱う EcosystemProvider を返す。
func NewProvider() gateway.EcosystemProvider {
	return &provider{all: []gateway.Ecosystem{NPM{}, GoMod{}}}
}

func (p *provider) ForRepo(ctx context.Context, src gateway.ManifestSource) ([]gateway.Ecosystem, error) {
	var found []gateway.Ecosystem
	for _, e := range p.all {
		ok, err := e.Detect(ctx, src)
		if err != nil {
			return nil, err
		}
		if ok {
			found = append(found, e)
		}
	}
	return found, nil
}

func (p *provider) ByID(id model.Ecosystem) gateway.Ecosystem {
	for _, e := range p.all {
		if e.ID() == id {
			return e
		}
	}
	return nil
}
