package model

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/vincentchyu/sonic-lens/common"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// NormalizeGenreQueryToken 统一清洗并解码流派查询 Token，防御 URL 编码遗留与二次编码
func NormalizeGenreQueryToken(genre string) string {
	clean := strings.TrimSpace(genre)
	if clean == "" {
		return ""
	}
	if unescaped, err := url.PathUnescape(clean); err == nil && unescaped != "" {
		clean = unescaped
	}
	if strings.Contains(clean, "%20") || strings.Contains(clean, "%2B") || strings.Contains(clean, "%2F") {
		if unescaped, err := url.QueryUnescape(clean); err == nil && unescaped != "" {
			clean = unescaped
		}
	}
	return strings.TrimSpace(clean)
}

// Genre represents a music genre
type Genre struct {
	ID        int64     `gorm:"column:id;type:int;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(255);not null;unique;index:idx_genre_name" json:"name"`
	NameZh    string    `gorm:"column:name_zh;type:varchar(255)" json:"name_zh"`
	Extra     string    `gorm:"column:extra;type:text" json:"extra"`
	PlayCount int64     `gorm:"column:play_count;type:bigint" json:"play_count"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

// TopGenre represents a top genre with play count
type TopGenre struct {
	TrackGenreName  string `json:"track_genre_name"`  // 流派英文名
	TrackGenreCount int64  `json:"track_genre_count"` // 流派播放次数
	GenreNameZh     string `json:"genre_name_zh"`     // 流派中文名
	GenreCount      int64  `json:"genre_count"`       // 流派总播放次数
	Rank            int    `json:"rank"`              // 排名
}

// TableName sets the table name for the Genre model
func (Genre) TableName() string {
	return "genre"
}

// CreateGenre creates a new genre
func CreateGenre(ctx context.Context, genre *Genre) error {
	log.Debug(ctx, "creating genre", zap.Any("genre", genre))
	return GetDB().WithContext(ctx).Create(genre).Error
}

// GetGenreByName retrieves a genre by name
func GetGenreByName(ctx context.Context, name string) (*Genre, error) {
	cleanName := NormalizeGenreQueryToken(name)
	if cleanName == "" {
		return nil, nil
	}
	var genre Genre
	err := GetDB().WithContext(ctx).Where("name = ?", cleanName).First(&genre).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &genre, nil
}

// GetGenreByID retrieves a genre by ID
func GetGenreByID(ctx context.Context, id int64) (*Genre, error) {
	var genre Genre
	err := GetDB().WithContext(ctx).Where("id = ?", id).First(&genre).Error
	if err != nil {
		return nil, err
	}
	return &genre, nil
}

// GetAllGenres retrieves all genres with pagination
func GetAllGenres(ctx context.Context, limit, offset int) ([]*Genre, error) {
	var genres []*Genre
	err := GetDB().WithContext(ctx).Order("play_count DESC").Limit(limit).Offset(offset).Find(&genres).Error
	if err != nil {
		return nil, err
	}
	return genres, nil
}

// GetAllGenresWithFilter retrieves genres with keyword search, sorting, and pagination, returning list and total count
func GetAllGenresWithFilter(ctx context.Context, keyword, sortBy string, limit, offset int) ([]*Genre, int64, error) {
	query := GetDB().WithContext(ctx).Model(&Genre{})

	cleanKeyword := strings.TrimSpace(keyword)
	if cleanKeyword != "" {
		likePat := "%" + cleanKeyword + "%"
		query = query.Where("name LIKE ? OR name_zh LIKE ?", likePat, likePat)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderClause := "play_count DESC, id ASC"
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "name", "name_asc":
		orderClause = "name ASC"
	case "name_desc":
		orderClause = "name DESC"
	case "play_count_asc":
		orderClause = "play_count ASC, id ASC"
	case "created_at_desc", "created_at":
		orderClause = "created_at DESC"
	case "updated_at_desc", "updated_at":
		orderClause = "updated_at DESC"
	}

	var genres []*Genre
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Order(orderClause).Find(&genres).Error; err != nil {
		return nil, 0, err
	}

	return genres, total, nil
}

// UpdateGenre updates a genre
func UpdateGenre(ctx context.Context, genre *Genre) error {
	return GetDB().WithContext(ctx).Save(genre).Error
}

// DeleteGenre deletes a genre
func DeleteGenre(ctx context.Context, id int64) error {
	return GetDB().WithContext(ctx).Delete(&Genre{}, id).Error
}

// IncrementGenrePlayCount increments the play count for a genre, creating it if it doesn't exist
func IncrementGenrePlayCount(ctx context.Context, name string) error {
	return InTx(
		ctx, func(tx *gorm.DB) error {
			var genre Genre
			// 使用 FirstOrCreate 确保流派记录存在
			if err := tx.Where("name = ?", name).FirstOrCreate(&genre, Genre{Name: name}).Error; err != nil {
				return err
			}
			// 原子增加播放次数
			return tx.Model(&genre).Update("play_count", gorm.Expr("play_count + 1")).Error
		},
	)
}

// GetGenreCount returns the total number of genres
func GetGenreCount(ctx context.Context) (int64, error) {
	var count int64
	err := GetDB().WithContext(ctx).Model(&Genre{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetTopGenresByPlayCount returns the top genres by play count
func GetTopGenresByPlayCount(ctx context.Context, limit int) ([]*Genre, error) {
	var genres []*Genre
	err := GetDB().WithContext(ctx).Order("play_count DESC").Limit(limit).Find(&genres).Error
	if err != nil {
		return nil, err
	}
	return genres, nil
}

// GetTopGenresWithDetails returns the top genres with detailed information including track count
func GetTopGenresWithDetails(ctx context.Context, limit int) ([]*TopGenre, error) {
	if statRows, err := GetTopGenresWithDetailsFromStat(ctx, limit); err == nil && len(statRows) >= limit {
		return statRows, nil
	}

	var result []*TopGenre

	// 根据数据库类型使用不同的SQL语法
	// 现在优先从 genre 表获取播放统计，以包含未归因的数据
	err := GetDB().WithContext(ctx).Raw(
		`SELECT name AS track_genre_name, play_count AS genre_count, name_zh AS genre_name_zh, play_count AS track_genre_count
		 FROM genre 
		 WHERE name != ''
		 ORDER BY play_count DESC 
		 LIMIT ?`, limit,
	).Scan(&result).Error
	if err != nil {
		if statRows, statErr := GetTopGenresWithDetailsFromStat(ctx, limit); statErr == nil && len(statRows) > 0 {
			return statRows, nil
		}
		return nil, err
	}

	for index := range result {
		result[index].Rank = index + 1
	}

	return result, nil
}

// UnmatchedGenreItem 表示在流派物理库/字典中未匹配到已知英文 Name 的未归因流派项
type UnmatchedGenreItem struct {
	RawGenre  string `json:"raw_genre"`  // 原始流派文本
	PlayCount int64  `json:"play_count"` // 影响的听歌流水次数
}

var (
	unmatchedGenresLock   sync.RWMutex
	globalUnmatchedGenres []UnmatchedGenreItem
)

// GetUnmatchedGenres 获取当前未匹配的未归因流派列表
func GetUnmatchedGenres() []UnmatchedGenreItem {
	unmatchedGenresLock.RLock()
	defer unmatchedGenresLock.RUnlock()
	res := make([]UnmatchedGenreItem, len(globalUnmatchedGenres))
	copy(res, globalUnmatchedGenres)
	return res
}

func setUnmatchedGenres(items []UnmatchedGenreItem) {
	unmatchedGenresLock.Lock()
	defer unmatchedGenresLock.Unlock()
	globalUnmatchedGenres = items
}

// ExtractPrimaryGenreTag 提取多段流派字符串的首个 Segment（支持逗号、分号、竖线、顿号及带空格的 ' / ' 分隔；严禁拆分 Cantopop/HK-Pop, Pop/Rock, Singer/Songwriter 等复合流派）
func ExtractPrimaryGenreTag(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}

	// 仅将明确的多流派列表分隔符（逗号、分号、竖线、顿号、带空格的斜杠）规整为逗号
	// 注意：保留紧凑词内单斜杠 '/'，如 "Cantopop/HK-Pop", "Pop/Rock", "Singer/Songwriter"
	replacer := strings.NewReplacer(
		" / ", ",",
		";", ",",
		"；", ",",
		"|", ",",
		"、", ",",
		"，", ",",
	)
	normalized := replacer.Replace(clean)
	parts := strings.Split(normalized, ",")
	for _, p := range parts {
		item := strings.TrimSpace(p)
		if item != "" {
			return item
		}
	}
	return clean
}

func extractPrimaryGenreTag(raw string) string {
	return ExtractPrimaryGenreTag(raw)
}

// NormalizeGenre 对任意传入的原始流派文本进行防脏提取与权威认证规范化：
// 1. 自动提取首个 Segment（剥离多流派拼接）
// 2. 调用 ResolveGenreIdentity（优先匹配权威流派规范名）
// 3. 若未命中标准英文流派，且为中文标签，自动繁转简为规范简体中文（如 "中國流行樂" -> "中国流行乐"）
// 4. 若为纯英文文本，自动进行 common.CapitalizeWords 规范化为 Title Case（如 "folk" -> "Folk"）
func NormalizeGenre(tx *gorm.DB, raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return ""
	}
	primaryTag := ExtractPrimaryGenreTag(clean)
	if primaryTag == "" {
		return ""
	}
	engName, zhName := ResolveGenreIdentity(tx, primaryTag)
	if engName != "" {
		return engName
	}
	if zhName != "" {
		return zhName
	}
	if !common.IsExistsChineseSimplified(primaryTag) {
		return common.CapitalizeWords(primaryTag)
	}
	return common.ConversionSimplifiedFx(primaryTag)
}

var (
	genreCacheResolverLock   sync.RWMutex
	globalGenreCacheResolver func(tag string) (string, string, bool)
)

// SetGenreCacheResolver 注册已认证的动态流派缓存解析器（如 cache.GenreCache）
func SetGenreCacheResolver(fn func(tag string) (string, string, bool)) {
	genreCacheResolverLock.Lock()
	defer genreCacheResolverLock.Unlock()
	globalGenreCacheResolver = fn
}

func getGenreFromCache(tag string) (string, string, bool) {
	genreCacheResolverLock.RLock()
	fn := globalGenreCacheResolver
	genreCacheResolverLock.RUnlock()
	if fn != nil {
		return fn(tag)
	}
	return "", "", false
}

type genreAccumulator struct {
	Name      string
	NameZh    string
	PlayCount int64
}

// ResolveGenreIdentity 智能解析流派身份，直接基于 ResolveStrictGenreIdentity 权威底座：
// - 若命中权威认证流派，返回 (canonicalEng, canonicalZh)；
// - 若为未认证中文，返回 ("", simplified)；
// - 若为未认证英文，返回 (CapitalizeWords(clean), "") 用于曲目/专辑展示，绝不捏造伪流派入库。
func ResolveGenreIdentity(tx *gorm.DB, tag string) (name string, nameZh string) {
	clean := strings.TrimSpace(tag)
	if clean == "" {
		return "", ""
	}

	matched, eng, zh := ResolveStrictGenreIdentity(tx, clean)
	if matched {
		return eng, zh
	}

	// 未匹配时：中文返回规范简体，英文返回 Title Case 形式用于展示
	if common.IsExistsChineseSimplified(clean) {
		if zh != "" {
			return "", zh
		}
		return "", common.ConversionSimplifiedFx(clean)
	}

	return common.CapitalizeWords(clean), ""
}

// ResolveStrictGenreIdentity 严格解析流派身份，判断是否能精准归因到已知标准权威 Name
func ResolveStrictGenreIdentity(tx *gorm.DB, tag string) (matched bool, eng string, zh string) {
	ctx := context.Background()
	if tx != nil && tx.Statement != nil && tx.Statement.Context != nil {
		ctx = tx.Statement.Context
	}

	clean := strings.TrimSpace(tag)
	if clean == "" {
		log.Info(ctx, "流派严格解析输入为空")
		return false, "", ""
	}

	log.Info(ctx, "开始严格解析流派身份", zap.String("tag", tag), zap.String("clean", clean))

	// 0. 优先查权威流派缓存 (getGenreFromCache 内部已全量承载 GenreCustomFit、繁简转换与 NormalizeChineseGenre)
	if canonicalEng, canonicalZh, ok := getGenreFromCache(clean); ok && canonicalEng != "" {
		log.Info(
			ctx, "步骤0: 流派命中动态权威缓存 (GenreCache)",
			zap.String("tag", clean),
			zap.String("canonical_eng", canonicalEng),
			zap.String("canonical_zh", canonicalZh),
		)
		return true, canonicalEng, canonicalZh
	}
	log.Info(ctx, "步骤0: 流派权威缓存未命中，继续排查数据库", zap.String("tag", clean))

	// 1. 仅在未命中缓存且提供了事务/连接 tx 时，查数据库已认证的记录作为兜底
	fit := common.GenreCustomFit(clean)
	if common.IsExistsChineseSimplified(fit) {
		simplified := common.ConversionSimplifiedFx(fit)
		normalizedZh := common.NormalizeChineseGenre(simplified)
		zh = simplified
		log.Info(
			ctx, "步骤1: 检测为中文流派标签，查库兜底",
			zap.String("tag", clean),
			zap.String("simplified", simplified),
			zap.String("normalized_zh", normalizedZh),
		)

		if tx != nil {
			var existing Genre
			if err := tx.Where(
				"TRIM(name_zh) = ? OR TRIM(name_zh) = ?", simplified, normalizedZh,
			).First(&existing).Error; err == nil && existing.Name != "" && !common.IsExistsChineseSimplified(existing.Name) && !strings.HasPrefix(
				existing.Name, "cn-slug-",
			) {
				log.Info(
					ctx, "步骤1: 中文标签命中数据库已认证记录",
					zap.String("simplified", simplified),
					zap.String("name", existing.Name),
					zap.Int64("id", existing.ID),
				)
				return true, existing.Name, simplified
			}
		}
		log.Info(ctx, "步骤1: 中文标签在数据库中未找到认证英文名，判定为未归因", zap.String("simplified", simplified))
		return false, "", zh
	}

	// 2. 本身是英文标志：查数据库已有合法英文记录
	log.Info(ctx, "步骤2: 检测为纯英文流派标签，查询数据库记录", zap.String("tag", clean))
	if tx != nil {
		var existing Genre
		if err := tx.Where(
			"TRIM(name) = ? OR LOWER(name) = LOWER(?)", fit, fit,
		).First(&existing).Error; err == nil && existing.Name != "" && !common.IsExistsChineseSimplified(existing.Name) && !strings.HasPrefix(
			existing.Name, "cn-slug-",
		) {
			log.Info(
				ctx, "步骤2: 英文标签命中数据库已认证记录",
				zap.String("tag", clean),
				zap.String("name", existing.Name),
				zap.String("name_zh", existing.NameZh),
				zap.Int64("id", existing.ID),
			)
			return true, existing.Name, existing.NameZh
		}
	}

	log.Info(ctx, "步骤2: 英文标签在数据库中未匹配到认证记录，判定为未归因", zap.String("tag", clean))
	return false, clean, ""
}

// ReconcileGenrePlayCountsTx 在事务中对流派播放量进行全量流水对账（仅对已知匹配流派计分，全量反向清洗流水表，暴露未匹配项供人工干预，绝对禁止自动插入未认证流派）
func ReconcileGenrePlayCountsTx(tx *gorm.DB) error {
	if tx == nil {
		return errors.New("tx is nil")
	}

	// 0. 自动自我修正物理表 genre 中被历史错误写入的小写脏数据，并将 cn-slug- 或中文 name 等历史脏流派 play_count 置 0
	var allGenres []Genre
	if err := tx.Find(&allGenres).Error; err == nil {
		for _, g := range allGenres {
			// 若物理表中存在多段组合、中文 name 或 cn-slug- 伪流派，重置其 play_count 为 0
			if strings.Contains(g.Name, ",") || strings.Contains(
				g.Name, "/",
			) || common.IsExistsChineseSimplified(g.Name) || strings.HasPrefix(g.Name, "cn-slug-") {
				tx.Model(&g).Update("play_count", 0)
				continue
			}
		}
	}

	type genreStat struct {
		Genre string `gorm:"column:genre"`
		Cnt   int64  `gorm:"column:cnt"`
	}

	var stats []genreStat
	if err := tx.Model(&TrackPlayRecord{}).
		Select("genre, COUNT(*) as cnt").
		Where("TRIM(COALESCE(genre, '')) <> ''").
		Group("genre").
		Scan(&stats).Error; err != nil {
		return err
	}

	tagCounts := make(map[string]*genreAccumulator)
	unmatchedMap := make(map[string]int64)

	for _, s := range stats {
		// 取首个 Segment
		primaryTag := extractPrimaryGenreTag(s.Genre)
		if primaryTag == "" {
			continue
		}

		matched, engName, zhName := ResolveStrictGenreIdentity(tx, primaryTag)
		if !matched {
			unmatchedMap[primaryTag] += s.Cnt
			continue // 未匹配流派仅计入未归因列表，绝不参与累加和自动建表！
		}
		if engName == "" {
			continue
		}

		// 核心关键：反向更正清洗 track_play_records 流水表中的原始脏流派字符串 s.Genre！
		// 只要原始流水 s.Genre 与权威规范 engName 不一致（包含多段拼接、中文或大小写差异），全量反向回写为 engName
		if s.Genre != engName {
			_ = tx.Model(&TrackPlayRecord{}).Where("genre = ?", s.Genre).Update("genre", engName).Error
		}

		lowerKey := strings.ToLower(engName)
		if acc, ok := tagCounts[lowerKey]; ok {
			acc.PlayCount += s.Cnt
			if acc.NameZh == "" && zhName != "" {
				acc.NameZh = zhName
			}
		} else {
			tagCounts[lowerKey] = &genreAccumulator{
				Name:      engName,
				NameZh:    zhName,
				PlayCount: s.Cnt,
			}
		}
	}

	// 保存未匹配的流派供前端人工干预展示
	var unmatchedItems []UnmatchedGenreItem
	for raw, cnt := range unmatchedMap {
		unmatchedItems = append(
			unmatchedItems, UnmatchedGenreItem{
				RawGenre:  raw,
				PlayCount: cnt,
			},
		)
	}
	setUnmatchedGenres(unmatchedItems)

	processedIDs := make(map[int64]bool)

	for _, acc := range tagCounts {
		var g Genre
		err := tx.Where("TRIM(name) = ? OR LOWER(name) = LOWER(?)", acc.Name, acc.Name).First(&g).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 仅当经认证的标准英文名确实不在物理表中时才创建，严禁插入中文或 cn-slug-
				if !common.IsExistsChineseSimplified(acc.Name) && !strings.HasPrefix(acc.Name, "cn-slug-") {
					g = Genre{
						Name:      acc.Name,
						NameZh:    acc.NameZh,
						PlayCount: acc.PlayCount,
					}
					if err := tx.Create(&g).Error; err != nil {
						return err
					}
					processedIDs[g.ID] = true
				}
				continue
			}
			return err
		}

		// 若现存记录与规范写仅大小写不同，且 acc.Name 是更标准写法，修正 name
		if g.Name != acc.Name && strings.EqualFold(g.Name, acc.Name) && acc.Name != "" {
			tx.Model(&g).Update("name", acc.Name)
		}

		// 仅仅更新 play_count 与 name_zh，绝不盲目覆写 name 字段！
		updates := map[string]interface{}{
			"play_count": acc.PlayCount,
		}
		if acc.NameZh != "" && g.NameZh == "" {
			updates["name_zh"] = acc.NameZh
		}
		if err := tx.Model(&g).Updates(updates).Error; err != nil {
			return err
		}

		processedIDs[g.ID] = true
	}

	// 3. 对于数据库中所有本次对账未出现的流派，将 play_count 安全置 0
	if len(processedIDs) > 0 {
		var ids []int64
		for id := range processedIDs {
			ids = append(ids, id)
		}
		if err := tx.Where("id NOT IN (?) AND play_count != 0", ids).Model(&Genre{}).Update(
			"play_count", 0,
		).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Where("play_count != 0").Model(&Genre{}).Update("play_count", 0).Error; err != nil {
			return err
		}
	}
	return nil
}

// ReconcileGenrePlayCounts 对流派播放量进行全量对账
func ReconcileGenrePlayCounts(ctx context.Context) error {
	return InTx(
		ctx, func(tx *gorm.DB) error {
			return ReconcileGenrePlayCountsTx(tx)
		},
	)
}

/*// GetGenreCache returns the global genre cache instance
func GetGenreCache() *cache.GenreCache {
	return cache.GetGenreCache()
}
*/
