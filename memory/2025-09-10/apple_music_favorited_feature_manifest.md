# Apple Music 点赞功能特性清单

## 概述

此特性为 Apple Music 支持添加了检查和设置歌曲点赞状态的功能，使用户能够通过程序化方式管理 Apple Music 中歌曲的点赞状态。

## 功能范围

1. 实现检查当前播放歌曲是否被点赞的功能
2. 实现设置当前播放歌曲点赞状态的功能
3. 添加相应的测试用例确保功能正常工作

## 实现要点

1. 在 `core/applemusic/sciprt.go` 中添加了 `IsFavorited` 函数用于检查当前歌曲是否被点赞
2. 在 `core/applemusic/sciprt.go` 中添加了 `SetFavorited` 函数用于设置当前歌曲的点赞状态
3. 在 `core/applemusic/sciprt_test.go` 中添加了相应的测试函数

## 技术细节

- 使用 AppleScript 命令 `if exists current track then return favorited of current track` 检查歌曲是否被点赞
- 使用 AppleScript 命令 `set favorited of current track to true/false` 设置歌曲的点赞状态
- 通过 `applesciprt.Tell` 函数执行 AppleScript 命令
- 正确处理了可能的错误情况，包括应用未运行或无当前播放歌曲的情况

## 并发安全

- 函数设计为无状态的，不涉及共享资源的访问
- 错误处理机制确保在异常情况下不会导致程序崩溃

## 测试

- 添加了 `TestIsFavorited` 测试函数验证检查点赞状态功能
- 添加了 `TestSetFavorited` 测试函数验证设置点赞状态功能
- 测试涵盖了应用未运行时的错误处理情况