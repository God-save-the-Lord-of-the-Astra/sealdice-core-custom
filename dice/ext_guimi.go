package dice

import (
	"strconv"
	"strings"
)

func RegisterGuiMiCommands(d *Dice) {
	helpForGuimiCharacterGenerate := ".诡秘 [<数量>] // 制卡指令，返回<数量>组人物属性"
	cmdGuimiCharacterGenerate := CmdItemInfo{
		Name:      "诡秘人物作成",
		ShortHelp: helpForGuimiCharacterGenerate,
		Help:      "诡秘3制卡指令:\n" + helpForGuimiCharacterGenerate,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			n := cmdArgs.GetArgN(1)
			val, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				if n == "" {
					val = 1 // 数量不存在时，视为1次
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
			}
			if val > ctx.Dice.MaxCocCardGen {
				val = ctx.Dice.MaxCocCardGen
			}
			var i int64
			var ss []string
			for i = 0; i < val; i++ {
				result := ctx.EvalFString(`力量:{力量=2d3} 敏捷:{敏捷=2d3} 意志:{意志=2d3}\n体质:{体质=2d3} 魅力:{魅力=2d3} 教育:{教育=2d3}\n灵感:{灵感=2d3} 幸运:{幸运=2d3}\nHP:{体质+10} <DB:{力量 <= 0 ? -2, 力量 <= 1 ? -1, 力量 <= 2 ? 0, 力量 <= 3 ? 1, 力量 <= 4 ? 1d2, 力量 <= 5 ? 1d4, 力量 <= 6 ? 1d6, 力量 <= 7 ? 1d8, 力量 <= 8 ? 1d10, 力量 <= 9 ? 1d12, 力量 <= 10 ? 2d6, 力量 <= 11 ? 2d8, 力量 <= 12 ? 2d10, 力量 <= 13 ? 2d12, 力量 <= 14 ? 3d8, 力量 <= 15 ? 3d10}> [{力量+敏捷+意志+体质+魅力+教育+灵感}/{力量+敏捷+意志+体质+魅力+教育+灵感+幸运}]`, nil)
				if result.vm.Error != nil {
					break
				}
				resultText := result.ToString()
				resultText = strings.ReplaceAll(resultText, `\n`, "\n")
				ss = append(ss, resultText)
			}
			sep := DiceFormatTmpl(ctx, "诡秘:制卡_分隔符")
			info := strings.Join(ss, sep)
			VarSetValueStr(ctx, "$t制卡结果文本", info)
			text := DiceFormatTmpl(ctx, "诡秘:制卡")
			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.RegisterExtension(&ExtInfo{
		Name:            "诡秘", // 扩展的名称，需要用于指令中，写简短点      2024.05.10: 目前被看成是 function 的缩写了（
		Version:         "3.0",
		Brief:           "诡秘3规则包",
		AutoActive:      true, // 是否自动开启
		ActiveOnPrivate: true,
		Author:          "海棠，星界之主",
		Official:        true,
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {

		},
		OnLoad: func() {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"诡秘": &cmdGuimiCharacterGenerate,
		}})
}
