package dice

import (
	"fmt"
	"strings"

	ds "github.com/sealdice/dicescript"
	lua "github.com/yuin/gopher-lua"
)

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
				LuaVarInit(L, ctx, msg, cmdArgs)
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
			/*
				case "js":
					code := strings.Join(cmdArgs.Args[1:], " ")

					//----------------------------------------------------------------
					// 重建js vm
					reg := new(require.Registry)
					vm := goja.New()
					// console 模块
					console.Enable(vm)
					// require 模块
					d.JsRequire = reg.Enable(vm)
					vm.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))
					//----------------------------------------------------------------
					seal := vm.NewObject()

					vars := vm.NewObject()
					_ = seal.Set("vars", vars)
					_ = vars.Set("intGet", VarGetValueInt64)
					_ = vars.Set("intSet", VarSetValueInt64)
					_ = vars.Set("strGet", VarGetValueStr)
					_ = vars.Set("strSet", VarSetValueStr)

					ban := vm.NewObject()
					_ = seal.Set("ban", ban)
					_ = ban.Set("addBan", func(ctx *MsgContext, id string, place string, reason string) {
						d.BanList.AddScoreBase(id, d.BanList.ThresholdBan, place, reason, ctx)
						d.BanList.SaveChanged(d)
					})
					_ = ban.Set("addTrust", func(ctx *MsgContext, id string, place string, reason string) {
						d.BanList.SetTrustByID(id, place, reason)
						d.BanList.SaveChanged(d)
					})
					_ = ban.Set("remove", func(ctx *MsgContext, id string) {
						_, ok := d.BanList.GetByID(id)
						if !ok {
							return
						}
						d.BanList.DeleteByID(d, id)
					})
					_ = ban.Set("getList", func() []BanListInfoItem {
						var list []BanListInfoItem
						d.BanList.Map.Range(func(key string, value *BanListInfoItem) bool {
							list = append(list, *value)
							return true
						})
						return list
					})
					_ = ban.Set("getUser", func(id string) *BanListInfoItem {
						i, ok := d.BanList.GetByID(id)
						if !ok {
							return nil
						}
						cp := *i
						return &cp
					})
					ext := vm.NewObject()
					_ = seal.Set("ext", ext)
					_ = ext.Set("newCmdItemInfo", func() *CmdItemInfo {
						return &CmdItemInfo{IsJsSolveFunc: true}
					})
					_ = ext.Set("newCmdExecuteResult", func(solved bool) CmdExecuteResult {
						return CmdExecuteResult{
							Matched: true,
							Solved:  solved,
						}
					})
					_ = ext.Set("new", func(name, author, version string) *ExtInfo {
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
					_ = ext.Set("find", func(name string) *ExtInfo {
						return d.ExtFind(name)
					})
					_ = ext.Set("register", func(ei *ExtInfo) {
						// NOTE(Xiangze Li): 移动到dice.RegisterExtension里去检查
						// if d.ExtFind(ei.Name) != nil {
						// 	panic("扩展<" + ei.Name + ">已被注册")
						// }

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
						// Pinenutn: Range模板 ServiceAtNew重构代码
						d.ImSession.ServiceAtNew.Range(func(key string, groupInfo *GroupInfo) bool {
							// Pinenutn: ServiceAtNew重构
							groupInfo.ExtActive(ei)
							return true
						})
					})
					_ = ext.Set("registerStringConfig", func(ei *ExtInfo, key string, defaultValue string, description string) error {
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
					_ = ext.Set("registerIntConfig", func(ei *ExtInfo, key string, defaultValue int64, description string) error {
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
					_ = ext.Set("registerBoolConfig", func(ei *ExtInfo, key string, defaultValue bool, description string) error {
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
					_ = ext.Set("registerFloatConfig", func(ei *ExtInfo, key string, defaultValue float64, description string) error {
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
					_ = ext.Set("registerTemplateConfig", func(ei *ExtInfo, key string, defaultValue []string, description string) error {
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
					_ = ext.Set("registerOptionConfig", func(ei *ExtInfo, key string, defaultValue string, option []string, description string) error {
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
					_ = ext.Set("newConfigItem", func(ei *ExtInfo, key string, defaultValue interface{}, description string) *ConfigItem {
						if ei.dice == nil {
							panic(errors.New("请先完成此扩展的注册"))
						}
						return d.ConfigManager.NewConfigItem(key, defaultValue, description)
					})
					_ = ext.Set("registerConfig", func(ei *ExtInfo, config ...*ConfigItem) error {
						if ei.dice == nil {
							return errors.New("请先完成此扩展的注册")
						}
						d.ConfigManager.RegisterPluginConfig(ei.Name, config...)
						return nil
					})
					_ = ext.Set("getConfig", func(ei *ExtInfo, key string) *ConfigItem {
						if ei.dice == nil {
							return nil
						}
						return d.ConfigManager.getConfig(ei.Name, key)
					})
					_ = ext.Set("getStringConfig", func(ei *ExtInfo, key string) string {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "string" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
					})
					_ = ext.Set("getIntConfig", func(ei *ExtInfo, key string) int64 {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "int" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(int64)
					})
					_ = ext.Set("getBoolConfig", func(ei *ExtInfo, key string) bool {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "bool" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(bool)
					})
					_ = ext.Set("getFloatConfig", func(ei *ExtInfo, key string) float64 {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "float" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(float64)
					})
					_ = ext.Set("getTemplateConfig", func(ei *ExtInfo, key string) []string {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "template" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.([]string)
					})
					_ = ext.Set("getOptionConfig", func(ei *ExtInfo, key string) string {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "option" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
					})
					_ = ext.Set("unregisterConfig", func(ei *ExtInfo, key ...string) {
						if ei.dice == nil {
							return
						}
						d.ConfigManager.UnregisterConfig(ei.Name, key...)
					})
					_ = seal.Set("replyGroup", ReplyGroup)
					_ = seal.Set("replyPerson", ReplyPerson)
					_ = seal.Set("replyToSender", ReplyToSender)
					_ = seal.Set("memberBan", MemberBan)
					_ = seal.Set("memberKick", MemberKick)
					_ = seal.Set("format", DiceFormat)
					_ = seal.Set("formatTmpl", DiceFormatTmpl)
					_ = seal.Set("getCtxProxyFirst", GetCtxProxyFirst)
					_ = seal.Set("newMessage", func() *Message {
						return &Message{}
					})
					_ = seal.Set("createTempCtx", CreateTempCtx)
					_ = seal.Set("applyPlayerGroupCardByTemplate", func(ctx *MsgContext, tmpl string) string {
						if tmpl != "" {
							ctx.Player.AutoSetNameTemplate = tmpl
						}
						if ctx.Player.AutoSetNameTemplate != "" {
							text, _ := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
							return text
						}
						return ""
					})
					_, _ = vm.RunString(`Object.freeze(seal);Object.freeze(seal.ext);Object.freeze(seal.vars);`)

					//----------------------------------------------------------------
					jsItf, err := vm.RunString(code)
					if err != nil {
						ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行出错:\n%s", err))
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行成功，返回结果:\n%s", jsItf.String()))
					return CmdExecuteResult{Matched: true, Solved: true}
				case "javascript":
					code := strings.Join(cmdArgs.Args[1:], " ")

					//----------------------------------------------------------------
					// 重建js vm
					reg := new(require.Registry)
					vm := goja.New()
					// console 模块
					console.Enable(vm)
					// require 模块
					d.JsRequire = reg.Enable(vm)
					vm.SetFieldNameMapper(goja.TagFieldNameMapper("jsbind", true))
					//----------------------------------------------------------------
					seal := vm.NewObject()

					vars := vm.NewObject()
					_ = seal.Set("vars", vars)
					_ = vars.Set("intGet", VarGetValueInt64)
					_ = vars.Set("intSet", VarSetValueInt64)
					_ = vars.Set("strGet", VarGetValueStr)
					_ = vars.Set("strSet", VarSetValueStr)

					ban := vm.NewObject()
					_ = seal.Set("ban", ban)
					_ = ban.Set("addBan", func(ctx *MsgContext, id string, place string, reason string) {
						d.BanList.AddScoreBase(id, d.BanList.ThresholdBan, place, reason, ctx)
						d.BanList.SaveChanged(d)
					})
					_ = ban.Set("addTrust", func(ctx *MsgContext, id string, place string, reason string) {
						d.BanList.SetTrustByID(id, place, reason)
						d.BanList.SaveChanged(d)
					})
					_ = ban.Set("remove", func(ctx *MsgContext, id string) {
						_, ok := d.BanList.GetByID(id)
						if !ok {
							return
						}
						d.BanList.DeleteByID(d, id)
					})
					_ = ban.Set("getList", func() []BanListInfoItem {
						var list []BanListInfoItem
						d.BanList.Map.Range(func(key string, value *BanListInfoItem) bool {
							list = append(list, *value)
							return true
						})
						return list
					})
					_ = ban.Set("getUser", func(id string) *BanListInfoItem {
						i, ok := d.BanList.GetByID(id)
						if !ok {
							return nil
						}
						cp := *i
						return &cp
					})
					ext := vm.NewObject()
					_ = seal.Set("ext", ext)
					_ = ext.Set("newCmdItemInfo", func() *CmdItemInfo {
						return &CmdItemInfo{IsJsSolveFunc: true}
					})
					_ = ext.Set("newCmdExecuteResult", func(solved bool) CmdExecuteResult {
						return CmdExecuteResult{
							Matched: true,
							Solved:  solved,
						}
					})
					_ = ext.Set("new", func(name, author, version string) *ExtInfo {
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
					_ = ext.Set("find", func(name string) *ExtInfo {
						return d.ExtFind(name)
					})
					_ = ext.Set("register", func(ei *ExtInfo) {
						// NOTE(Xiangze Li): 移动到dice.RegisterExtension里去检查
						// if d.ExtFind(ei.Name) != nil {
						// 	panic("扩展<" + ei.Name + ">已被注册")
						// }

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
						// Pinenutn: Range模板 ServiceAtNew重构代码
						d.ImSession.ServiceAtNew.Range(func(key string, groupInfo *GroupInfo) bool {
							// Pinenutn: ServiceAtNew重构
							groupInfo.ExtActive(ei)
							return true
						})
					})
					_ = ext.Set("registerStringConfig", func(ei *ExtInfo, key string, defaultValue string, description string) error {
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
					_ = ext.Set("registerIntConfig", func(ei *ExtInfo, key string, defaultValue int64, description string) error {
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
					_ = ext.Set("registerBoolConfig", func(ei *ExtInfo, key string, defaultValue bool, description string) error {
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
					_ = ext.Set("registerFloatConfig", func(ei *ExtInfo, key string, defaultValue float64, description string) error {
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
					_ = ext.Set("registerTemplateConfig", func(ei *ExtInfo, key string, defaultValue []string, description string) error {
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
					_ = ext.Set("registerOptionConfig", func(ei *ExtInfo, key string, defaultValue string, option []string, description string) error {
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
					_ = ext.Set("newConfigItem", func(ei *ExtInfo, key string, defaultValue interface{}, description string) *ConfigItem {
						if ei.dice == nil {
							panic(errors.New("请先完成此扩展的注册"))
						}
						return d.ConfigManager.NewConfigItem(key, defaultValue, description)
					})
					_ = ext.Set("registerConfig", func(ei *ExtInfo, config ...*ConfigItem) error {
						if ei.dice == nil {
							return errors.New("请先完成此扩展的注册")
						}
						d.ConfigManager.RegisterPluginConfig(ei.Name, config...)
						return nil
					})
					_ = ext.Set("getConfig", func(ei *ExtInfo, key string) *ConfigItem {
						if ei.dice == nil {
							return nil
						}
						return d.ConfigManager.getConfig(ei.Name, key)
					})
					_ = ext.Set("getStringConfig", func(ei *ExtInfo, key string) string {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "string" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
					})
					_ = ext.Set("getIntConfig", func(ei *ExtInfo, key string) int64 {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "int" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(int64)
					})
					_ = ext.Set("getBoolConfig", func(ei *ExtInfo, key string) bool {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "bool" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(bool)
					})
					_ = ext.Set("getFloatConfig", func(ei *ExtInfo, key string) float64 {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "float" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(float64)
					})
					_ = ext.Set("getTemplateConfig", func(ei *ExtInfo, key string) []string {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "template" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.([]string)
					})
					_ = ext.Set("getOptionConfig", func(ei *ExtInfo, key string) string {
						if ei.dice == nil || d.ConfigManager.getConfig(ei.Name, key).Type != "option" {
							panic("配置不存在或类型不匹配")
						}
						return d.ConfigManager.getConfig(ei.Name, key).Value.(string)
					})
					_ = ext.Set("unregisterConfig", func(ei *ExtInfo, key ...string) {
						if ei.dice == nil {
							return
						}
						d.ConfigManager.UnregisterConfig(ei.Name, key...)
					})
					_ = seal.Set("replyGroup", ReplyGroup)
					_ = seal.Set("replyPerson", ReplyPerson)
					_ = seal.Set("replyToSender", ReplyToSender)
					_ = seal.Set("memberBan", MemberBan)
					_ = seal.Set("memberKick", MemberKick)
					_ = seal.Set("format", DiceFormat)
					_ = seal.Set("formatTmpl", DiceFormatTmpl)
					_ = seal.Set("getCtxProxyFirst", GetCtxProxyFirst)
					_ = seal.Set("newMessage", func() *Message {
						return &Message{}
					})
					_ = seal.Set("createTempCtx", CreateTempCtx)
					_ = seal.Set("applyPlayerGroupCardByTemplate", func(ctx *MsgContext, tmpl string) string {
						if tmpl != "" {
							ctx.Player.AutoSetNameTemplate = tmpl
						}
						if ctx.Player.AutoSetNameTemplate != "" {
							text, _ := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
							return text
						}
						return ""
					})
					_, _ = vm.RunString(`Object.freeze(seal);Object.freeze(seal.ext);Object.freeze(seal.vars);`)
					//----------------------------------------------------------------

					jsItf, err := vm.RunString(code)
					if err != nil {
						ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行出错:\n%s", err))
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					ReplyToSender(ctx, msg, fmt.Sprintf("JavaScript 代码执行成功，返回结果:\n%s", jsItf.String()))
					return CmdExecuteResult{Matched: true, Solved: true}*/
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
