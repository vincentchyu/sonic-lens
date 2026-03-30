package cache

import (
	"context"

	"github.com/vincenty1ung/yeung-go-study/lru"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/exec"
	alog "github.com/vincentchyu/sonic-lens/core/log"
)

var lruCache = lru.Constructor[string](200)

func FindMataDataHandle(ctx context.Context, key string) exec.MataDataHandle {
	var (
		mataDataHandle exec.MataDataHandle
		err            error
	)

	if exiftoolInfo := lruCache.Get(key); exiftoolInfo != nil {
		mataDataHandle = exiftoolInfo.(exec.MataDataHandle)
	} else {
		if ok, path, _ := exec.IsValidPath(ctx, key); ok {
			if exec.GetFilePathExt(path) == common.FileExtWav1 || exec.GetFilePathExt(path) == common.FileExtWav2 {
				// WAV 内嵌封面依赖 exiftool 元数据；失败时再回退旧解析，避免丢失基础字段。
				mataDataHandle, err = exec.BuildExiftoolHandle(ctx, path)
				if err != nil {
					alog.Warn(ctx, "exec BuildExiftoolHandle", zap.Error(err))
					mataDataHandle, err = exec.BuildWavInfoHandle(path)
					if err != nil {
						alog.Warn(ctx, "exec BuildWavInfoHandle", zap.Error(err))
						return mataDataHandle
					}
				}
				if mataDataHandle != nil {
					lruCache.Put(key, mataDataHandle)
				}
			} else {
				mataDataHandle, err = exec.BuildExiftoolHandle(ctx, path)
				if err != nil {
					alog.Warn(ctx, "exec BuildExiftoolHandle", zap.Error(err))
					return mataDataHandle
				}
				if mataDataHandle != nil {
					lruCache.Put(key, mataDataHandle)
				}
			}
		}
	}
	return mataDataHandle
}
