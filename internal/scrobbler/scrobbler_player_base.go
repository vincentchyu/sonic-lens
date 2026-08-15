package scrobbler

import (
	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/internal/cache"
)

type BaseWrapper struct {
}

func (m BaseWrapper) ConversionSimplified(target string) string {
	return common.ConversionSimplifiedFx(target)
}

func (m BaseWrapper) GetGenre(genre string) string {
	return cache.GetEnglishGenre(genre)
}
