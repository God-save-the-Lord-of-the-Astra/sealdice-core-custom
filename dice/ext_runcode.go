package dice

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	ds "github.com/sealdice/dicescript"
	lua "github.com/yuin/gopher-lua"
)

func ExecCodeJsInit(d *Dice, vm *goja.Runtime, ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	//设置全局变量和方法
	vm.Set("msg", msg)
	vm.Set("ctx", ctx)
	vm.Set("cmdArgs", cmdArgs)
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))

	_ = vm.Set("intGet", VarGetValueInt64)
	_ = vm.Set("intSet", VarSetValueInt64)
	_ = vm.Set("strGet", VarGetValueStr)
	_ = vm.Set("strSet", VarSetValueStr)

	_ = vm.Set("addBan", func(ctx *MsgContext, id string, place string, reason string) {
		d.BanList.AddScoreBase(id, d.BanList.ThresholdBan, place, reason, ctx)
		d.BanList.SaveChanged(d)
	})
	_ = vm.Set("addTrust", func(ctx *MsgContext, id string, place string, reason string) {
		d.BanList.SetTrustByID(id, place, reason)
		d.BanList.SaveChanged(d)
	})
	_ = vm.Set("remove", func(ctx *MsgContext, id string) {
		_, ok := d.BanList.GetByID(id)
		if !ok {
			return
		}
		d.BanList.DeleteByID(d, id)
	})
	_ = vm.Set("getList", func() []BanListInfoItem {
		var list []BanListInfoItem
		d.BanList.Map.Range(func(key string, value *BanListInfoItem) bool {
			list = append(list, *value)
			return true
		})
		return list
	})
	_ = vm.Set("getUser", func(id string) *BanListInfoItem {
		i, ok := d.BanList.GetByID(id)
		if !ok {
			return nil
		}
		cp := *i
		return &cp
	})

	ext := vm.NewObject()
	_ = vm.Set("ext", ext)
	_ = vm.Set("newCmdItemInfo", func() *CmdItemInfo {
		return &CmdItemInfo{IsJsSolveFunc: true}
	})
	_ = vm.Set("newCmdExecuteResult", func(solved bool) CmdExecuteResult {
		return CmdExecuteResult{
			Matched: true,
			Solved:  solved,
		}
	})
	_ = vm.Set("new", func(name, author, version string) *ExtInfo {
		var official bool
		if d.JsLoadingScript != nil {
			official = d.JsLoadingScript.Official
		}
		return &ExtInfo{
			Name: name, Author: author, Version: version,
			GetDescText: GetExtensionDesc,
			AutoActive:  true,
			IsJsExt:     true,
			Brief:       "一个JS自定义扩展",
			Official:    official,
			CmdMap:      CmdMapCls{},
			Source:      d.JsLoadingScript,
		}
	})
	_ = vm.Set("find", func(name string) *ExtInfo {
		return d.ExtFind(name)
	})
	_ = vm.Set("register", func(ei *ExtInfo) {
		defer func() {
			// 增加recover, 以免在scripts目录中存在名字冲突扩展时导致启动崩溃
			if e := recover(); e != nil {
				d.Logger.Error(e)
			}
		}()

		d.RegisterExtension(ei)
		if ei.OnLoad != nil {
			ei.OnLoad()
		}
		d.ApplyExtDefaultSettings()
		d.ImSession.ServiceAtNew.Range(func(key string, groupInfo *GroupInfo) bool {
			groupInfo.ExtActive(ei)
			return true
		})
	})
	_ = vm.Set("registerStringConfig", func(ei *ExtInfo, key string, defaultValue string, description string) error {
		if ei.dice == nil {
			return errors.New("请先完成此扩展的注册")
		}
		config := &ConfigItem{
			Key:          key,
			Type:         "string",
			Value:        defaultValue,
			DefaultValue: defaultValue,
			Description:  description,
		}
		d.ConfigManager.RegisterPluginConfig(ei.Name, config)
		return nil
	})
	_ = vm.Set("registerIntConfig", func(ei *ExtInfo, key string, defaultValue int64, description string) error {
		if ei.dice == nil {
			return errors.New("请先完成此扩展的注册")
		}
		config := &ConfigItem{
			Key:          key,
			Type:         "int",
			Value:        defaultValue,
			DefaultValue: defaultValue,
			Description:  description,
		}
		d.ConfigManager.RegisterPluginConfig(ei.Name, config)
		return nil
	})
	_ = vm.Set("registerBoolConfig", func(ei *ExtInfo, key string, defaultValue bool, description string) error {
		if ei.dice == nil {
			return errors.New("请先完成此扩展的注册")
		}
		config := &ConfigItem{
			Key:          key,
			Type:         "bool",
			Value:        defaultValue,
			DefaultValue: defaultValue,
			Description:  description,
		}
		d.ConfigManager.RegisterPluginConfig(ei.Name, config)
		return nil
	})
	_ = vm.Set("registerFloatConfig", func(ei *ExtInfo, key string, defaultValue float64, description string) error {
		if ei.dice == nil {
			return errors.New("请先完成此扩展的注册")
		}
		config := &ConfigItem{
			Key:          key,
			Type:         "float",
			Value:        defaultValue,
			DefaultValue: defaultValue,
			Description:  description,
		}
		d.ConfigManager.RegisterPluginConfig(ei.Name, config)
		return nil
	})
	_ = vm.Set("registerTemplateConfig", func(ei *ExtInfo, key string, defaultValue []string, description string) error {
		if ei.dice == nil {
			return errors.New("请先完成此扩展的注册")
		}
		config := &ConfigItem{
			Key:          key,
			Type:         "template",
			Value:        defaultValue,
			DefaultValue: defaultValue,
			Description:  description,
		}
		d.ConfigManager.RegisterPluginConfig(ei.Name, config)
		return nil
	})
	_ = vm.Set("registerOptionConfig", func(ei *ExtInfo, key string, defaultValue string, option []string, description string) error {
		if ei.dice == nil {
			return errors.New("请先完成此扩展的注册")
		}
		config := &ConfigItem{
			Key:          key,
			Type:         "option",
			Value:        defaultValue,
			DefaultValue: defaultValue,
			Option:       option,
			Description:  description,
		}
		d.ConfigManager.RegisterPluginConfig(ei.Name, config)
		return nil
	})
	_ = vm.Set("newConfigItem", func(ei *ExtInfo, key string, defaultValue interface{}, description string) *ConfigItem {
		if ei.dice == nil {
			panic(errors.New("请先完成此扩展的注册"))
		}
		return d.ConfigManager.NewConfigItem(key, defaultValue, description)
	})
	_ = vm.Set("registerConfig", func(ei *ExtInfo, config ...*ConfigItem) error {
		if ei.dice == nil {
			return errors.New("请先完成此扩展的注册")
		}
		d.ConfigManager.RegisterPluginConfig(ei.Name, config...)
		return nil
	})
	_ = vm.Set("getConfig", func(ei *ExtInfo, key string) *ConfigItem {
		if ei.dice == nil {
			return nil
		}
		return d.ConfigManager.getConfig(ei.Name, key)
	})
	_ = vm.Set("getStringConfig", func(ei *ExtInfo, key string) string {
		if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "string" {
			panic("配置不存在或类型不匹配")
		}
		return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
	})
	_ = vm.Set("getIntConfig", func(ei *ExtInfo, key string) int64 {
		if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "int" {
			panic("配置不存在或类型不匹配")
		}
		return d.ConfigManager.getConfig(ei.Name, key).Value.(int64)
	})
	_ = vm.Set("getBoolConfig", func(ei *ExtInfo, key string) bool {
		if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "bool" {
			panic("配置不存在或类型不匹配")
		}
		return d.ConfigManager.getConfig(ei.Name, key).Value.(bool)
	})
	_ = vm.Set("getFloatConfig", func(ei *ExtInfo, key string) float64 {
		if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "float" {
			panic("配置不存在或类型不匹配")
		}
		return d.ConfigManager.getConfig(ei.Name, key).Value.(float64)
	})
	_ = vm.Set("getTemplateConfig", func(ei *ExtInfo, key string) []string {
		if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "template" {
			panic("配置不存在或类型不匹配")
		}
		return d.ConfigManager.getConfig(ei.Name, key).Value.([]string)
	})
	_ = vm.Set("getOptionConfig", func(ei *ExtInfo, key string) string {
		if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "option" {
			panic("配置不存在或类型不匹配")
		}
		return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
	})
	_ = vm.Set("unregisterConfig", func(ei *ExtInfo, key ...string) {
		if ei.dice == nil {
			return
		}
		d.ConfigManager.UnregisterConfig(ei.Name, key...)
	})

	_ = vm.Set("registerTask", func(ei *ExtInfo, taskType string, value string, fn func(taskCtx JsScriptTaskCtx), key string, desc string) *JsScriptTask {
		if ei.dice == nil {
			panic(errors.New("请先完成此扩展的注册"))
		}
		scriptCron := ei.dice.JsScriptCron
		if scriptCron == nil {
			panic(errors.New("插件cron未成功初始化")) // 按理是不会发生的
		}

		task := JsScriptTask{cron: scriptCron, key: key, task: fn, lock: ei.dice.JsScriptCronLock, logger: ei.dice.Logger}
		expr := value
		if key != "" {
			if config := d.ConfigManager.getConfig(ei.Name, key); config != nil {
				expr = config.Value.(string)
				// Stop old task
				if config.task != nil {
					config.task.Off()
				}
			}
		}

		switch taskType {
		case "cron":
			entryID, err := scriptCron.AddFunc(expr, func() {
				task.run()
			})
			if err != nil {
				panic("插件注册定时任务失败：" + err.Error())
			}
			task.taskType = taskType
			task.rawValue = expr
			task.cronExpr = expr
			task.entryID = &entryID
			ei.dice.Logger.Infof("插件注册定时任务：cron=%s", expr)
		case "daily":
			// 支持每天定时触发，24 小时表示
			cronExpr, err := parseTaskTime(expr)
			if err != nil {
				panic("插件注册定时任务失败：" + err.Error())
			}

			entryID, err := scriptCron.AddFunc(cronExpr, func() {
				task.run()
			})
			if err != nil {
				panic("插件注册定时任务失败：" + err.Error())
			}
			task.taskType = taskType
			task.rawValue = expr
			task.cronExpr = cronExpr
			task.entryID = &entryID
			ei.dice.Logger.Infof("插件注册定时任务：daily=%s", expr)
		default:
			panic(fmt.Sprintf("错误的任务类型：%s，当前仅支持 cron|daily", taskType))
		}

		if key != "" {
			config := d.ConfigManager.getConfig(ei.Name, key)

			switch taskType {
			case "cron":
				config = &ConfigItem{
					Key:          key,
					Type:         "task:cron",
					Value:        expr,
					DefaultValue: value,
					Description:  desc,
					task:         &task,
				}
			case "daily":
				config = &ConfigItem{
					Key:          key,
					Type:         "task:daily",
					Value:        expr,
					DefaultValue: value,
					Description:  desc,
					task:         &task,
				}
			}
			d.ConfigManager.RegisterPluginConfig(ei.Name, config)
		}

		if key == "" {
			// 如果不提供 key，手动避免 task 失去引用
			if ei.taskList == nil {
				ei.taskList = make([]*JsScriptTask, 0)
				ei.taskList = append(ei.taskList, &task)
			} else {
				ei.taskList = append(ei.taskList, &task)
			}
		}

		return &task
	})

	_ = vm.Set("newRule", func() *CocRuleInfo {
		return &CocRuleInfo{}
	})
	_ = vm.Set("newRuleCheckResult", func() *CocRuleCheckRet {
		return &CocRuleCheckRet{}
	})
	_ = vm.Set("registerRule", func(rule *CocRuleInfo) bool {
		return d.CocExtraRulesAdd(rule)
	})

	_ = vm.Set("draw", func(ctx *MsgContext, deckName string, isShuffle bool) map[string]interface{} {
		exists, result, err := deckDraw(ctx, deckName, isShuffle)
		var errText string
		if err != nil {
			errText = err.Error()
		}
		return map[string]interface{}{
			"exists": exists,
			"err":    errText,
			"result": result,
		}
	})
	_ = vm.Set("reload", func() {
		DeckReload(d)
	})

	// 设置其他 seal 方法
	_ = vm.Set("replyGroup", ReplyGroup)
	_ = vm.Set("replyPerson", ReplyPerson)
	_ = vm.Set("replyToSender", ReplyToSender)
	_ = vm.Set("memberBan", MemberBan)
	_ = vm.Set("memberKick", MemberKick)
	_ = vm.Set("format", DiceFormat)
	_ = vm.Set("formatTmpl", DiceFormatTmpl)
	_ = vm.Set("getCtxProxyFirst", GetCtxProxyFirst)
	_ = vm.Set("newMessage", func() *Message {
		return &Message{}
	})
	_ = vm.Set("createTempCtx", CreateTempCtx)
	_ = vm.Set("applyPlayerGroupCardByTemplate", func(ctx *MsgContext, tmpl string) string {
		if tmpl != "" {
			ctx.Player.AutoSetNameTemplate = tmpl
		}
		if ctx.Player.AutoSetNameTemplate != "" {
			text, _ := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
			return text
		}
		return ""
	})
	_ = vm.Set("setPlayerGroupCard", SetPlayerGroupCardByTemplate)
	_ = vm.Set("base64ToImage", Base64ToImageFunc(d.Logger))
	_ = vm.Set("getCtxProxyAtPos", GetCtxProxyAtPos)
	_ = vm.Set("getVersion", func() map[string]interface{} {
		return map[string]interface{}{
			"versionCode":   VERSION_CODE,
			"version":       VERSION.String(),
			"versionSimple": VERSION_MAIN + VERSION_PRERELEASE,
			"versionDetail": map[string]interface{}{
				"major":         VERSION.Major(),
				"minor":         VERSION.Minor(),
				"patch":         VERSION.Patch(),
				"prerelease":    VERSION.Prerelease(),
				"buildMetaData": VERSION.Metadata(),
			},
		}
	})
	_ = vm.Set("getEndPoints", func() []*EndPointInfo {
		return d.ImSession.EndPoints
	})

	// 设置 atob 和 btoa 方法
	_ = vm.Set("atob", func(s string) (string, error) {
		s = strings.ReplaceAll(s, "data:text/plain;base64,", "")
		s = strings.ReplaceAll(s, " ", "")
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return "", errors.New("atob: 不合法的base64字串")
		}
		return string(b), nil
	})
	_ = vm.Set("btoa", func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	})

}
func RegisterExecCodeCommands(d *Dice) {
	helpForExecCode := ".execcode <语言> <代码块> //运行指定语言的代码块"
	cmdExecCode := CmdItemInfo{
		Name:      "execcode",
		ShortHelp: helpForExecCode,
		Help:      "运行代码指令:\n" + helpForExecCode,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("lua", "js", "javascript", "ds", "dicescript")
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, "你不具备Master权限")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if len(cmdArgs.Args) < 2 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			var val = cmdArgs.GetArgN(1)
			switch strings.ToLower(val) {
			case "lua":
				//---------------------------替换容错-------------------------------------
				code := strings.Join(cmdArgs.Args[1:], " ")
				code = strings.Replace(code, "。", ".", -1)
				code = strings.Replace(code, "，", ",", -1)
				code = strings.Replace(code, "（", "(", -1)
				code = strings.Replace(code, "）", ")", -1)
				code = strings.Replace(code, "【", "[", -1)
				code = strings.Replace(code, "】", "]", -1)
				code = strings.Replace(code, "；", ";", -1)
				code = strings.Replace(code, "——", "_", -1)
				code = strings.Replace(code, "：", ":", -1)
				code = strings.Replace(code, "！", "!", -1)
				code = strings.Replace(code, "!=", "~=", -1)
				//----------------------------------------------------------------
				L := lua.NewState()
				defer L.Close()

				//初始化lua全局变量
				LuaVarInit(L, d, ctx, msg, cmdArgs)
				//初始化lua全局函数
				LuaFuncInit(L, ctx, msg, cmdArgs)

				// 加载并执行 Lua 脚本
				if err := L.DoString(fmt.Sprintf("%s %s %s", "function main() ", code, " end")); err != nil {
					ReplyToSender(ctx, msg, fmt.Sprintf("Lua 代码执行出错:\n%s", err))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 获取 Lua 中的 `main` 函数
				luaMain := L.GetGlobal("main")

				// 调用 Lua 函数
				L.Push(luaMain) // 将函数压入栈
				L.Call(0, 1)    // 调用函数，0个参数，期望1个返回值

				// 获取并打印返回值
				if L.GetTop() >= 1 {
					ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "Lua 代码执行成功，返回结果:\n", L.ToString(-1))) // Lua栈中的最后一个元素（即返回值）
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "ds":
				code := strings.Join(cmdArgs.Args[1:], " ")
				vm := ds.NewVM()
				if err := vm.Run(code); err != nil {
					ReplyToSender(ctx, msg, fmt.Sprintf("DiceScript 代码执行出错:\n%s", err.Error()))
				} else {
					dsResult := vm.Ret.ToString()
					ReplyToSender(ctx, msg, fmt.Sprintf("DiceScript 代码执行成功，返回结果:\n%s", dsResult))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "dicescript":
				code := strings.Join(cmdArgs.Args[1:], " ")
				vm := ds.NewVM()
				if err := vm.Run(code); err != nil {
					ReplyToSender(ctx, msg, fmt.Sprintf("DiceScript 代码执行出错:\n%s", err.Error()))
				} else {
					dsResult := vm.Ret.ToString()
					ReplyToSender(ctx, msg, fmt.Sprintf("DiceScript 代码执行成功，返回结果:\n%s", dsResult))
				}
				return CmdExecuteResult{Matched: true, Solved: true}

			case "js":
				code := strings.Join(cmdArgs.Args[1:], " ")
				vm := goja.New()
				// 确保在执行代码之前已经设置好 seal 对象和其他全局变量
				ExecCodeJsInit(d, vm, ctx, msg, cmdArgs)
				// 执行传入的 JavaScript 代码
				jsItf, err := vm.RunString(code)
				if err != nil {
					ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行出错:\n%s", err))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行成功，返回结果:\n%s", jsItf.String()))
				return CmdExecuteResult{Matched: true, Solved: true}
			case "javascript":
				code := strings.Join(cmdArgs.Args[1:], " ")
				vm := goja.New()
				// 确保在执行代码之前已经设置好 seal 对象和其他全局变量
				ExecCodeJsInit(d, vm, ctx, msg, cmdArgs)
				// 执行传入的 JavaScript 代码
				jsItf, err := vm.RunString(code)
				if err != nil {
					ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行出错:\n%s", err))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行成功，返回结果:\n%s", jsItf.String()))
				return CmdExecuteResult{Matched: true, Solved: true}

			default:
				ReplyToSender(ctx, msg, "不支持的语言："+val)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
		},
	}
	d.RegisterExtension(&ExtInfo{
		Name:            "execcode", // 扩展的名称，需要用于指令中，写简短点      2024.05.10: 目前被看成是 function 的缩写了（
		Version:         "0.0.1",
		Brief:           "运行代码",
		AutoActive:      true, // 是否自动开启
		ActiveOnPrivate: true,
		Author:          "海棠",
		Official:        false,
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {

		},
		OnLoad: func() {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"execcode": &cmdExecCode,
			"ec":       &cmdExecCode,
		}})
}
