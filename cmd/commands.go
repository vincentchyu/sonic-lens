package cmd

import (
	"github.com/spf13/cobra"
)

// RegisterCommands 是 SonicLens 命令行子工具的统一注册总入口。
//
// 📌 【AI Agent 与开发者子命令接入规约 (CLI Extension Protocol)】：
//
// 1. 架构定位：
//    - cmd 包仅作为轻量 CLI 交互外壳，负责 flags 参数解析、输入校验与标准输出格式化；
//    - 严禁在 cmd 层直接书写复杂领域逻辑或裸 SQL，核心业务与数据操作必须下沉至 internal/logic 或 internal/model。
//
// 2. 新增子命令标准步骤：
//    a. 在 cmd/ 目录下新建对应功能的 go 文件（例如 cmd/my_tool.go）；
//    b. 导出标准的构造函数：func NewMyToolCommand() *cobra.Command；
//    c. 在当前文件的 RegisterCommands 函数中通过 rootCmd.AddCommand(NewMyToolCommand()) 完成挂载。
//
// 3. 资源初始化标准：
//    - 配置文件由 rootCmd.PersistentPreRun 统一自动加载（config.InitConfig）；
//    - 若子命令需要访问数据库，统一在 RunE 内部调用 model.InitDB(nil)；
//    - 若需要结构化日志，使用 core/log 模块记录。
//
// 4. 错误处理规范：
//    - RunE 函数统一返回 error（通过 fmt.Errorf("%w") 包装），由 Cobra 处理异常退出。
func RegisterCommands(rootCmd *cobra.Command) {
	// 在此处按需注册新的子命令，示例：
	// rootCmd.AddCommand(NewMyToolCommand())
}
