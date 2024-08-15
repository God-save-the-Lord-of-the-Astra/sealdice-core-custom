package dice

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
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
			cmdArgs.ChopPrefixToArgsWith("lua")
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, "你不具备Master权限")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			var val = cmdArgs.GetArgN(1)
			switch strings.ToLower(val) {
			case "lua":
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
				L := lua.NewState()
				defer L.Close()

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
		}})
}
