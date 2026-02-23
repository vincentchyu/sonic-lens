package scrobbler

import (
	"github.com/vincentchyu/sonic-lens/common"
)

type BaseWrapper struct {
}

func (m BaseWrapper) ConversionSimplified(target string) string {
	return common.ConversionSimplifiedFx(common.CustomReplaceStringFunction(target))
}
