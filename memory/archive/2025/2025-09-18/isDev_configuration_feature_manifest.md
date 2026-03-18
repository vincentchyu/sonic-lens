# isDev 配置项特性清单

## 特性概述

isDev 配置项用于控制数据库表结构的初始化行为，根据环境不同提供不同的数据库初始化策略。

## 功能详情

### 1. 配置项说明

- **配置位置**: `config/config.yaml` 中的 `isDev` 字段
- **数据类型**: boolean
- **默认值**: true

### 2. 不同数据库类型的行为

#### SQLite 数据库

- 无论 `isDev` 设置为何值，都会自动创建本地表
- 表结构迁移通过 GORM 的 AutoMigrate 功能实现

#### MySQL 数据库

- **开发环境** (`isDev: true`):
    - 程序启动时自动进行表结构迁移
    - 通过 GORM 的 AutoMigrate 功能创建或更新表结构
    - 适用于开发和测试环境，简化部署流程

- **生产环境** (`isDev: false`):
    - 不会自动进行表结构迁移
    - 需要手动执行 SQL 语句来初始化或更新数据库表结构
    - 适用于生产环境，避免意外的表结构变更

### 3. 使用场景

#### 开发环境推荐配置

```yaml
database:
  type: "mysql"
  mysql:
    host: "localhost"
    port: 3306
    user: "dev_user"
    password: "dev_password"
    database: "lastfm_scrobbler_dev"
isDev: true
```

#### 生产环境推荐配置

```yaml
database:
  type: "mysql"
  mysql:
    host: "prod-host"
    port: 3306
    user: "prod_user"
    password: "prod_password"
    database: "lastfm_scrobbler_prod"
isDev: false
```

## 实现细节

### 核心代码位置

- 配置定义: `config/config.go`
- 数据库初始化逻辑: `internal/model/init.go`

### 关键实现逻辑

在 `internal/model/init.go` 的 `InitDB` 函数中，根据 `config.ConfigObj.IsDev` 的值决定是否执行 MySQL 数据库的
AutoMigrate：

```go
case string(common.DatabaseTypeMySQL):
    // Open MySQL database with custom logger
    GlobalDBForMysql, err = gorm.Open(
        mysql.Open(db.MysqlDSN(config.ConfigObj.Database.Mysql.GetMysqlDSN())), &gorm.Config{
            Logger: customLogger,
        },
    )
    if err != nil {
        return err
    }
    if config.ConfigObj.IsDev {
        // Auto migrate the schema for TrackPlayRecord
        err = GlobalDBForMysql.AutoMigrate(&TrackPlayRecord{})
        if err != nil {
            return err
        }

        // Auto migrate the schema for Track
        err = GlobalDBForMysql.AutoMigrate(&Track{})
        if err != nil {
            return err
        }

        // Auto migrate the schema for Genre
        err = GlobalDBForMysql.AutoMigrate(&Genre{})
        if err != nil {
            return err
        }
    }
```

## 注意事项

1. 在生产环境中，将 `isDev` 设置为 `false` 后，需要手动维护数据库表结构
2. 表结构变更时，需要准备相应的 SQL 脚本进行升级
3. 建议在生产环境中使用专门的数据库迁移工具来管理表结构变更