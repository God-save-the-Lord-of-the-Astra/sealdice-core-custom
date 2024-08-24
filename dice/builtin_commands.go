package dice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"

	"github.com/golang-module/carbon"
	"github.com/samber/lo"

	"github.com/juliangruber/go-intersect"
	ds "github.com/sealdice/dicescript"
)

func Float64SliceToString(numbers []float64) string {
	// 创建一个空字符串，用于存储结果
	var result strings.Builder

	// 遍历切片中的每个元素
	for i, number := range numbers {
		// 将float64转换为string
		str := strconv.FormatFloat(number, 'f', -1, 64) // 'f' 表示float，-1 表示精度（自动选择）

		// 除非它是第一个元素，否则在元素前添加逗号
		if i > 0 {
			result.WriteString(",")
		}

		// 将转换后的字符串添加到结果中
		result.WriteString(str)
	}

	// 返回结果字符串
	return result.String()
}

type warningMessage struct {
	Wid       int64  `json:"wid"`
	Type      string `json:"type"`
	Danger    int    `json:"danger"`
	FromGroup int64  `json:"fromGroup"`
	FromGID   int64  `json:"fromGID"`
	FromQQ    int64  `json:"fromQQ"`
	FromUID   int64  `json:"fromUID"`
	InviterQQ int64  `json:"inviterQQ"`
	Time      string `json:"time"`
	Note      string `json:"note"`
	DiceMaid  int64  `json:"DiceMaid"`
	MasterQQ  int64  `json:"masterQQ"`
	Comment   string `json:"comment"`
}

func DiceFormatReplyshow(key string, ctx *MsgContext, s string, srcText string) string {
	VarSetValueStr(ctx, "$t原因句子", "{$t原因句子}")
	VarSetValueStr(ctx, "$t结果文本", "{$t结果文本}")
	VarSetValueStr(ctx, "$t旧群内状态", "{$t旧群内状态}")
	VarSetValueStr(ctx, "$t群内工作状态", "{$t群内工作状态}")
	VarSetValueStr(ctx, "$t原因", "{$t原因}")
	VarSetValueStr(ctx, "$t次数", "{$t次数}")
	VarSetValueStr(ctx, "$t表达式文本", "{$t表达式文本}")
	VarSetValueStr(ctx, "$t计算过程", "{$t计算过程}")
	VarSetValueStr(ctx, "$t旧群内状态", "{$t旧群内状态}")
	VarSetValueStr(ctx, "$t计算结果", "{$t计算结果}")
	VarSetValueStr(ctx, "$t事发群名", "{$t事发群名}")
	VarSetValueStr(ctx, "$t事发群号", "{$t事发群号}")
	VarSetValueStr(ctx, "$t黑名单事件", "{$t黑名单事件}")
	VarSetValueStr(ctx, "$t原始列表", "{$t原始列表}")
	VarSetValueStr(ctx, "$t随机名字文本", "{$t随机名字文本}")
	VarSetValueStr(ctx, "$t请求结果", "{$t请求结果}")
	VarSetValueStr(ctx, "$t条数", "{$t条数}")
	VarSetValueStr(ctx, "$t记录名称", "{$t记录名称}")
	VarSetValueStr(ctx, "$t当前记录条数", "{$t当前记录条数}")
	VarSetValueStr(ctx, "$t角色名", "{$t角色名}")
	text := fmt.Sprintf("词条: %s\nwebui: %s\n默认: %s\n预览: %s", key, s, srcText, DiceFormatTmpl(ctx, s))
	return text
}

/** 这几条指令不能移除 */
func (d *Dice) registerCoreCommands() {
	helpForBlack := ".ban add user <帐号> [<原因>] //添加个人\n" +
		".ban add group <群号> [<原因>] //添加群组\n" +
		".ban add <统一ID>\n" +
		".ban rm user <帐号> //解黑/移出信任\n" +
		".ban rm group <群号>\n" +
		".ban rm <统一ID> //同上\n" +
		".ban list // 展示列表\n" +
		".ban list ban/warn/trust //只显示被禁用/被警告/信任用户\n" +
		".ban import <统一ID> <统一ID> ... //批量导入黑名单\n" +
		".ban trust <统一ID> //添加信任\n" +
		".ban query <统一ID> //查看指定用户拉黑情况\n" +
		".ban help //查看帮助\n" +
		"// 统一ID示例: QQ:12345、QQ-Group:12345"
	cmdBlack := &CmdItemInfo{
		Name:      "ban",
		ShortHelp: helpForBlack,
		Help:      "黑名单指令:\n" + helpForBlack,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("add", "rm", "del", "list", "show", "find", "trust", "import")
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			getID := func() string {
				if cmdArgs.IsArgEqual(2, "user") || cmdArgs.IsArgEqual(2, "group") {
					id := cmdArgs.GetArgN(3)
					if id == "" {
						return ""
					}

					isGroup := cmdArgs.IsArgEqual(2, "group")
					return FormatDiceID(ctx, id, isGroup)
				}

				arg := cmdArgs.GetArgN(2)
				if !strings.Contains(arg, ":") {
					return ""
				}
				return arg
			}

			var val = cmdArgs.GetArgN(1)
			var uid string
			switch strings.ToLower(val) {
			case "add":
				uid = getID()
				if uid == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				reason := cmdArgs.GetArgN(4)
				if reason == "" {
					reason = "骰主指令"
				}
				d.BanList.AddScoreBase(uid, d.BanList.ThresholdBan, "骰主指令", reason, ctx)
				ReplyToSender(ctx, msg, fmt.Sprintf("已将用户/群组 %s 加入黑名单，原因: %s", uid, reason))
			case "rm", "del":
				uid = getID()
				if uid == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				item, ok := d.BanList.GetByID(uid)
				if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
					ReplyToSender(ctx, msg, "找不到用户/群组")
					break
				}

				ReplyToSender(ctx, msg, fmt.Sprintf("已将用户/群组 %s 移出%s列表", uid, BanRankText[item.Rank]))
				item.Score = 0
				item.Rank = BanRankNormal
			case "trust":
				uid = cmdArgs.GetArgN(2)
				if !strings.Contains(uid, ":") {
					// 如果不是这种格式，那么放弃
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				d.BanList.SetTrustByID(uid, "骰主指令", "骰主指令")
				ReplyToSender(ctx, msg, fmt.Sprintf("已将用户/群组 %s 加入信任列表", uid))
			case "import":
				BlackUIDCnt := 0
				for _, uid := range cmdArgs.Args[2:] {
					if strings.Contains(uid, "QQ:") {
						BlackUIDCnt++
						item, ok := d.BanList.GetByID(uid)
						if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
							d.BanList.AddScoreBase(uid, d.BanList.ThresholdBan, "骰主指令", "骰主指令，黑名单批量导入", ctx)
						}

					}
					if strings.Contains(uid, "QQ-Group:") {
						BlackUIDCnt++
						item, ok := d.BanList.GetByID(uid)
						if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
							d.BanList.AddScoreBase(uid, d.BanList.ThresholdBan, "骰主指令", "骰主指令，黑名单批量导入", ctx)
						}
					}

				}
				ReplyToSender(ctx, msg, fmt.Sprintf("已导入 %d 个黑名单用户/群组", BlackUIDCnt))

			case "list", "show":
				// ban/warn/trust
				var extra, text string

				extra = cmdArgs.GetArgN(2)
				d.BanList.Map.Range(func(k string, v *BanListInfoItem) bool {
					if v.Rank == BanRankNormal {
						return true
					}

					match := (extra == "trust" && v.Rank == BanRankTrusted) ||
						(extra == "ban" && v.Rank == BanRankBanned) ||
						(extra == "warn" && v.Rank == BanRankWarn)
					if extra == "" || match {
						text += v.toText(d) + "\n"
					}
					return true
				})

				if text == "" {
					text = "当前名单:\n<无内容>"
				} else {
					text = "当前名单:\n" + text
				}
				ReplyToSender(ctx, msg, text)
			case "query":
				var targetID = cmdArgs.GetArgN(2)
				if targetID == "" {
					ReplyToSender(ctx, msg, "未指定要查询的对象！")
					break
				}

				v, exists := d.BanList.Map.Load(targetID)
				if !exists {
					ReplyToSender(ctx, msg, fmt.Sprintf("所查询的<%s>情况：正常(0)", targetID))
					break
				}

				var text = fmt.Sprintf("所查询的<%s>情况：", targetID)
				switch v.Rank {
				case BanRankBanned:
					text += "禁止(-30)"
				case BanRankWarn:
					text += "警告(-10)"
				case BanRankTrusted:
					text += "信任(30)"
				default:
					text += "正常(0)"
				}
				for i, reason := range v.Reasons {
					text += fmt.Sprintf(
						"\n%s在「%s」，原因：%s",
						carbon.CreateFromTimestamp(v.Times[i]).ToDateTimeString(),
						v.Places[i],
						reason,
					)
				}
				ReplyToSender(ctx, msg, text)
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["black"] = cmdBlack
	d.CmdMap["ban"] = cmdBlack

	helpForShikiBlack := ".admin blackqq +/- <帐号> [<原因>]\n" +
		".admin blackgroup +/- <群号> [<原因>]\n" +
		".admin dismiss <群号> [<原因>]\n" +
		".admin notice list //列出消息通知窗口\n" +
		".admin notice +/- <统一ID> //增删消息通知窗口"
	cmdShikiBlack := &CmdItemInfo{
		Name:      "admin",
		ShortHelp: helpForShikiBlack,
		Help:      "管理指令:\n" + helpForShikiBlack,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("blackqq", "blackgroup", "-", "+", "dismiss")
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			getID := func() string {
				if cmdArgs.IsArgEqual(2, "-") || cmdArgs.IsArgEqual(2, "+") {
					id := cmdArgs.GetArgN(3)
					if id == "" {
						return ""
					}

					isGroup := cmdArgs.IsArgEqual(1, "blackgroup")
					return FormatDiceID(ctx, id, isGroup)
				}

				arg := cmdArgs.GetArgN(2)
				if !strings.Contains(arg, ":") {
					return ""
				}
				return arg
			}

			var val = cmdArgs.GetArgN(1)
			var uid string
			switch strings.ToLower(val) {
			case "blackqq":

				var subval = cmdArgs.GetArgN(2)
				if subval == "-" {
					uid = getID()
					if uid == "" {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}

					item, ok := d.BanList.GetByID(uid)
					if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
						ReplyToSender(ctx, msg, "找不到用户")
						break
					}

					ReplyToSender(ctx, msg, fmt.Sprintf("已将用户 %s 移出%s列表", uid, BanRankText[item.Rank]))
					item.Score = 0
					item.Rank = BanRankNormal

				} else if subval == "+" {
					uid = getID()
					if uid == "" {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}
					reason := cmdArgs.GetArgN(4)
					if reason == "" {
						reason = "骰主指令"
					}
					d.BanList.AddScoreBase(uid, d.BanList.ThresholdBan, "骰主指令", reason, ctx)
					ReplyToSender(ctx, msg, fmt.Sprintf("已将用户 %s 加入黑名单，原因: %s", uid, reason))

				} else {
					return CmdExecuteResult{Matched: true, Solved: false, ShowHelp: true}
				}
			case "blackgroup":
				var subval = cmdArgs.GetArgN(2)
				if subval == "-" {
					uid = getID()
					if uid == "" {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}

					item, ok := d.BanList.GetByID(uid)
					if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
						ReplyToSender(ctx, msg, "找不到群组")
						break
					}

					ReplyToSender(ctx, msg, fmt.Sprintf("已将群组 %s 移出%s列表", uid, BanRankText[item.Rank]))
					item.Score = 0
					item.Rank = BanRankNormal

				} else if subval == "+" {
					uid = getID()
					if uid == "" {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}
					reason := cmdArgs.GetArgN(4)
					if reason == "" {
						reason = "骰主指令"
					}
					d.BanList.AddScoreBase(uid, d.BanList.ThresholdBan, "骰主指令", reason, ctx)
					ReplyToSender(ctx, msg, fmt.Sprintf("已将群组 %s 加入黑名单，原因: %s", uid, reason))

				} else {
					return CmdExecuteResult{Matched: true, Solved: false, ShowHelp: true}
				}

			case "dismiss":
				gid := cmdArgs.GetArgN(2)
				if gid == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				n := strings.Split(gid, ":") // 不验证是否合法，反正下面会检查是否在 ServiceAtNew
				gid = "QQ-Group:" + gid      // 强制当作QQ群聊处理
				gp, ok := ctx.Session.ServiceAtNew.Load(gid)
				if !ok || len(n[0]) < 2 {
					ReplyToSender(ctx, msg, fmt.Sprintf("群组列表中没有找到%s", gid))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 既然是骰主自己操作，就不通知了
				// 除非有多骰主……
				ReplyToSender(ctx, msg, fmt.Sprintf("收到指令，将在5秒后退出群组%s", gp.GroupID))

				txt := "注意，收到骰主指令，5秒后将从该群组退出。"
				wherefore := cmdArgs.GetArgN(3)
				if wherefore != "" {
					txt += fmt.Sprintf("原因: %s", wherefore)
				}

				ReplyGroup(ctx, &Message{GroupID: gp.GroupID}, txt)

				mctx := &MsgContext{
					MessageType: "group",
					Group:       gp,
					EndPoint:    ctx.EndPoint,
					Session:     ctx.Session,
					Dice:        ctx.Dice,
					IsPrivate:   false,
				}
				// SetBotOffAtGroup(mctx, gp.GroupID)
				time.Sleep(3 * time.Second)
				gp.DiceIDExistsMap.Delete(mctx.EndPoint.UserID)
				gp.UpdatedAtTime = time.Now().Unix()
				mctx.EndPoint.Adapter.QuitGroup(mctx, gp.GroupID)

				return CmdExecuteResult{Matched: true, Solved: true}

			case "notice":
				var subval = cmdArgs.GetArgN(2)
				NoticeUID := cmdArgs.GetArgN(3)
				if strings.ToLower(subval) == "list" {
					text := ""
					for _, v := range d.NoticeIDs {
						text += "- " + v + "\n"
					}
					if text == "" {
						text = "无"
					}
					reply := fmt.Sprintf("通知窗口列表:\n%s", text)
					ReplyToSender(ctx, msg, reply)
					return CmdExecuteResult{Matched: true, Solved: true}
				} else {
					if strings.HasPrefix(NoticeUID, "g") {
						NoticeUID = strings.ReplaceAll(NoticeUID, "g", "QQ-Group:")
					}
					if strings.HasPrefix(NoticeUID, "QQ-Group:") || strings.HasPrefix(NoticeUID, "QQ:") || strings.HasPrefix(NoticeUID, "Mail:") {
						// 需要以QQ-Group:或者QQ:或者g开头
					} else {
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "不正确的消息通知窗口表达式", NoticeUID))
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					if subval == "" || subval == "help" {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					} else if subval == "+" {
						d.NoticeIDs = append(d.NoticeIDs, NoticeUID)
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s", "已将窗口", NoticeUID, "加入消息通知队列"))
						d.Save(false)
						return CmdExecuteResult{Matched: true, Solved: true}
					} else if subval == "-" {
						for i, v := range d.NoticeIDs {
							if v == NoticeUID {
								d.NoticeIDs = append(d.NoticeIDs[:i], d.NoticeIDs[i+1:]...)
								ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s", "已将窗口", NoticeUID, "移出消息通知队列"))
								d.Save(false)
								return CmdExecuteResult{Matched: true, Solved: true}
							}
						}
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "没有找到消息通知窗口", NoticeUID))
						d.Save(false)
					} else {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}
				}
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["admin"] = cmdShikiBlack

	//-----------------------------云黑对接-----------------------------------

	helpForShikiCloudBlack := ".cloud sync //与溯洄云黑手动同步一次\n" +
		".cloud autosync //与溯洄云黑每天自动同步一次(这是个饼)"

	cmdShikiCloudBlack := &CmdItemInfo{
		Name:      "cloud",
		ShortHelp: helpForShikiCloudBlack,
		Help:      "同步云黑指令:\n" + helpForShikiCloudBlack,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("sync", "autosync")
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			ctx.EndPoint.Platform = "QQ"

			type blackunit struct {
				BlackQQ      string
				BlackGroup   string
				WarningID    string
				BlackComment string
				ErasedStatus bool
			}

			type jsonElement struct {
				Wid       int    `json:"wid"`
				FromGroup int    `json:"fromGroup"`
				FromQQ    int    `json:"fromQQ"`
				Type      string `json:"type"`
				Note      string `json:"note"`
				IsErased  int    `json:"isErased"`
			}

			fetchAndParseJSON_shikiCloudBlack := func(url string) ([]blackunit, error) {
				// 发送 HTTP GET 请求
				resp, err := http.Get(url)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()

				// 读取响应体
				body, _ := io.ReadAll(resp.Body)
				var jsonData []jsonElement
				err = json.Unmarshal(
					body,
					&jsonData,
				)
				if err != nil {
					return nil, err
				}

				// 将 JSON 数据转换为 blackunit 结构体数组
				var blackUnits []blackunit
				for _, item := range jsonData {
					unit := blackunit{
						BlackQQ:      strconv.Itoa(item.FromQQ),
						BlackGroup:   strconv.Itoa(item.FromGroup),
						WarningID:    strconv.Itoa(item.Wid),
						BlackComment: item.Type + " " + item.Note,
						ErasedStatus: item.IsErased != 0,
					}
					blackUnits = append(blackUnits, unit)
				}

				return blackUnits, nil
			}

			blackGroupCnt := 0
			blackGroupNewCnt := 0
			blackQQCnt := 0
			blackQQNewCnt := 0
			erasedCnt := 0
			erasedNewCnt := 0

			var val = cmdArgs.GetArgN(1)
			switch strings.ToLower(val) {
			case "sync":
				ReplyToSender(ctx, msg, "正在同步云黑...")
				time.Sleep(1000 * time.Millisecond)
				url := "https://shiki.stringempty.xyz/blacklist/checked.json?"
				blackUnits, _ := fetchAndParseJSON_shikiCloudBlack(url)
				for _, blackitem := range blackUnits {
					qqTobeBlack := FormatDiceID(ctx, blackitem.BlackQQ, false)
					groupTobeBlack := FormatDiceID(ctx, blackitem.BlackGroup, true)
					if !blackitem.ErasedStatus {
						item, ok := d.BanList.GetByID(qqTobeBlack)
						if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
							d.BanList.AddScoreBase(qqTobeBlack, d.BanList.ThresholdBan, "溯洄云黑", blackitem.BlackComment, ctx)
							blackQQNewCnt++
						}
						blackQQCnt++
						item, ok = d.BanList.GetByID(groupTobeBlack)
						if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
							d.BanList.AddScoreBase(groupTobeBlack, d.BanList.ThresholdBan, "溯洄云黑", blackitem.BlackComment, ctx)
							blackGroupNewCnt++
						}
						blackGroupCnt++

					} else {
						erasedCnt++
						item, ok := d.BanList.GetByID(qqTobeBlack)
						if ok && (item.Rank == BanRankBanned || item.Rank == BanRankTrusted || item.Rank == BanRankWarn) {
							item.Score = 0
							item.Rank = BanRankNormal
							erasedNewCnt++
						}
						item, ok = d.BanList.GetByID(groupTobeBlack)
						if ok && (item.Rank == BanRankBanned || item.Rank == BanRankTrusted || item.Rank == BanRankWarn) {
							item.Score = 0
							item.Rank = BanRankNormal
						}

					}
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("共计从溯洄云黑api获取黑名单群组:%d个，新增:%d个；黑名单用户:%d名，新增:%d名。并有%d组已在云端消除黑名单记录，新增%d组。", blackGroupCnt, blackGroupNewCnt, blackQQCnt, blackQQNewCnt, erasedCnt, erasedNewCnt))
				return CmdExecuteResult{Matched: true, Solved: true}
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		},
	}

	d.CmdMap["cloud"] = cmdShikiCloudBlack

	//-----------------------------云黑对接-----------------------------------

	//--------------------------------接收溯洄系骰子自动生成的warning----------------------------------------
	helpForShikiWarning := "溯洄warning播报处理"
	cmdShikiWarning := &CmdItemInfo{
		Name:      "warning",
		ShortHelp: helpForShikiWarning,
		Help:      "黑名单接收:\n" + helpForShikiWarning,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// 解析警告信息
			warningInformation := cmdArgs.RawArgs
			var warningStruct warningMessage
			err := json.Unmarshal([]byte(warningInformation), &warningStruct)
			if err != nil {
				ReplyToSender(ctx, msg, "警告信息解析失败"+cmdArgs.Args[0])
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			retMes := ""
			if warningStruct.Type != "erase" {
				// 处理fromGroup和fromQQ
				if warningStruct.FromGroup != 0 {
					warningEventGroup := fmt.Sprintf("QQ-Group:%d", warningStruct.FromGroup)
					item, ok := d.BanList.GetByID(warningEventGroup)
					if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
						d.BanList.AddScoreBase(warningEventGroup, d.BanList.ThresholdBan, warningStruct.Comment, "溯洄广播黑名单同步", ctx)
					}
					retMes += fmt.Sprintf("已将%s加入黑名单✓\n", warningEventGroup)
				}
				if warningStruct.FromGID != 0 {
					warningEventGroup := fmt.Sprintf("QQ-Group:%d", warningStruct.FromGID)
					item, ok := d.BanList.GetByID(warningEventGroup)
					if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
						d.BanList.AddScoreBase(warningEventGroup, d.BanList.ThresholdBan, warningStruct.Comment, "溯洄广播黑名单同步", ctx)
					}
					retMes += fmt.Sprintf("已将%s加入黑名单✓\n", warningEventGroup)
				}
				if warningStruct.FromQQ != 0 {
					warningEventQQ := fmt.Sprintf("QQ:%d", warningStruct.FromQQ)
					item, ok := d.BanList.GetByID(warningEventQQ)
					if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
						d.BanList.AddScoreBase(warningEventQQ, d.BanList.ThresholdBan, warningStruct.Comment, "溯洄广播黑名单同步", ctx)
					}
					retMes += fmt.Sprintf("已将%s加入黑名单✓", warningEventQQ)
				}

				if warningStruct.FromUID != 0 {
					warningEventQQ := fmt.Sprintf("QQ:%d", warningStruct.FromUID)
					item, ok := d.BanList.GetByID(warningEventQQ)
					if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
						d.BanList.AddScoreBase(warningEventQQ, d.BanList.ThresholdBan, warningStruct.Comment, "溯洄广播黑名单同步", ctx)
					}
					retMes += fmt.Sprintf("已将%s加入黑名单✓", warningEventQQ)
				}
			} else {
				// 处理fromGroup和fromQQ
				if warningStruct.FromGroup != 0 {
					warningEventGroup := fmt.Sprintf("QQ-Group:%d", warningStruct.FromGroup)
					item, ok := d.BanList.GetByID(warningEventGroup)
					if ok && (item.Rank == BanRankBanned || item.Rank == BanRankTrusted || item.Rank == BanRankWarn) {
						item.Score = 0
						item.Rank = BanRankNormal
					}
					retMes += fmt.Sprintf("已将%s移除黑名单✓\n", warningEventGroup)
				}
				if warningStruct.FromGID != 0 {
					warningEventGroup := fmt.Sprintf("QQ-Group:%d", warningStruct.FromGID)
					item, ok := d.BanList.GetByID(warningEventGroup)
					if ok && (item.Rank == BanRankBanned || item.Rank == BanRankTrusted || item.Rank == BanRankWarn) {
						item.Score = 0
						item.Rank = BanRankNormal
					}
					retMes += fmt.Sprintf("已将%s移除黑名单✓\n", warningEventGroup)
				}
				if warningStruct.FromQQ != 0 {
					warningEventQQ := fmt.Sprintf("QQ:%d", warningStruct.FromQQ)
					item, ok := d.BanList.GetByID(warningEventQQ)
					if ok && (item.Rank == BanRankBanned || item.Rank == BanRankTrusted || item.Rank == BanRankWarn) {
						item.Score = 0
						item.Rank = BanRankNormal
					}
					retMes += fmt.Sprintf("已将%s移除黑名单✓", warningEventQQ)
				}
				if warningStruct.FromUID != 0 {
					warningEventQQ := fmt.Sprintf("QQ:%d", warningStruct.FromUID)
					item, ok := d.BanList.GetByID(warningEventQQ)
					if ok && (item.Rank == BanRankBanned || item.Rank == BanRankTrusted || item.Rank == BanRankWarn) {
						item.Score = 0
						item.Rank = BanRankNormal
					}
					retMes += fmt.Sprintf("已将%s移除黑名单✓", warningEventQQ)
				}

			}
			var warningInformationJson bytes.Buffer
			_ = json.Indent(&warningInformationJson, []byte(warningInformation), "", "    ")

			ReplyToSender(ctx, msg, retMes)
			ReplyToSender(ctx, msg, fmt.Sprintf("%s %s已通知%s不良记录%d:\n!warning%s", time.Now().Format("2006-01-02 15:04:05"), ctx.Player.Name, "骰娘", warningStruct.Wid, warningInformationJson.String()))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	d.CmdMap["warning"] = cmdShikiWarning

	//--------------------------------接收溯洄系骰子自动生成的warning----------------------------------------

	helpForFind := ".find/查询 <关键字> // 查找文档。关键字可以多个，用空格分割\n" +
		".find #<分组> <关键字> // 查找指定分组下的文档。关键字可以多个，用空格分割\n" +
		".find <数字ID> // 显示该ID的词条\n" +
		".find --rand // 显示随机词条\n" +
		".find <关键字> --num=10 // 需要更多结果\n" +
		".find config --group // 查看当前默认搜索分组\n" +
		".find config --group=<分组> // 设置当前默认搜索分组\n" +
		".find config --groupclr // 清空当前默认搜索分组"
	cmdFind := &CmdItemInfo{
		Name:      "find",
		ShortHelp: helpForFind,
		Help:      "查询指令，通常使用全文搜索(x86版)或快速查询(arm, 移动版):\n" + helpForFind,
		// 写不下了
		// + "\n注: 默认搭载的《怪物之锤查询》来自蜜瓜包、October整理\n默认搭载的COC《魔法大典》来自魔骨，NULL，Dr.Amber整理\n默认搭载的DND系列文档来自DicePP项目"
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			if d.Parent.IsHelpReloading {
				ReplyToSender(ctx, msg, "帮助文档正在重新装载，请稍后...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if _config := cmdArgs.GetArgN(1); _config == "config" {
				oldDefault := ctx.Group.DefaultHelpGroup
				if cmdArgs.GetKwarg("groupclr") != nil {
					ctx.Group.SetDefaultHelpGroup("")
					if oldDefault != "" {
						ReplyToSender(ctx, msg, "已清空默认搜索分组，原分组为"+oldDefault)
					} else {
						ReplyToSender(ctx, msg, "未指定默认搜索分组")
					}
				} else if _defaultGroup := cmdArgs.GetKwarg("group"); _defaultGroup != nil {
					defaultGroup := _defaultGroup.Value
					if defaultGroup == "" {
						// 为查看默认分组
						if oldDefault != "" {
							ReplyToSender(ctx, msg, "当前默认搜索分组为"+oldDefault)
						} else {
							ReplyToSender(ctx, msg, "未指定默认搜索分组")
						}
					} else {
						// 为设置默认分组
						ctx.Group.SetDefaultHelpGroup(defaultGroup)
						if oldDefault != "" {
							ReplyToSender(ctx, msg, fmt.Sprintf("默认搜索分组由%s切换到%s", oldDefault, defaultGroup))
						} else {
							ReplyToSender(ctx, msg, "指定默认搜索分组为"+defaultGroup)
						}
					}
				} else {
					ReplyToSender(ctx, msg, "设置选项有误")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			var (
				useGroupSearch bool
				group          string
			)
			if _group := cmdArgs.GetArgN(1); strings.HasPrefix(_group, "#") {
				useGroupSearch = true
				fakeGroup := strings.TrimPrefix(_group, "#")

				// 转换 group 别名
				if _g, ok := d.Parent.Help.GroupAliases[fakeGroup]; ok {
					group = _g
				} else {
					group = fakeGroup
				}
			}
			var groupStr string
			if group != "" {
				groupStr = "[搜索分组" + group + "]"
			}

			var id string
			if cmdArgs.GetKwarg("rand") != nil || cmdArgs.GetKwarg("随机") != nil {
				_id := rand.Uint64()%d.Parent.Help.CurID + 1
				id = strconv.FormatUint(_id, 10)
			}

			if id == "" {
				var _id string
				if useGroupSearch {
					_id = cmdArgs.GetArgN(2)
				} else {
					_id = cmdArgs.GetArgN(1)
				}
				if _id != "" {
					_, err2 := strconv.ParseInt(_id, 10, 64)
					if err2 == nil {
						id = _id
					}
				}
			}

			if id != "" {
				text, exists := d.Parent.Help.TextMap[id]
				if exists {
					content := d.Parent.Help.GetContent(text, 0)
					ReplyToSender(ctx, msg, fmt.Sprintf("词条: %s:%s\n%s", text.PackageName, text.Title, content))
				} else {
					ReplyToSender(ctx, msg, "未发现对应ID的词条")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			var val string
			if useGroupSearch {
				val = cmdArgs.GetArgN(2)
			} else {
				val = cmdArgs.GetArgN(1)
			}
			if val == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			numLimit := 4
			numParam := cmdArgs.GetKwarg("num")
			if numParam != nil {
				_num, err := strconv.ParseInt(numParam.Value, 10, 64)
				if err == nil {
					numLimit = int(_num)
				}
			}

			page := 1
			pageParam := cmdArgs.GetKwarg("page")
			if pageParam != nil {
				if _page, err := strconv.ParseInt(pageParam.Value, 10, 64); err == nil {
					page = int(_page)
				}
			}

			text := strings.TrimPrefix(cmdArgs.CleanArgs, "#"+group+" ")

			if numLimit <= 0 {
				numLimit = 1
			} else if numLimit > 10 {
				numLimit = 10
			}
			if page <= 0 {
				page = 1
			}
			if group == "" {
				// 未指定搜索分组时，取当前群指定的分组
				group = ctx.Group.DefaultHelpGroup
			}
			search, total, pgStart, pgEnd, err := d.Parent.Help.Search(ctx, text, false, numLimit, page, group)
			if err != nil {
				ReplyToSender(ctx, msg, groupStr+"搜索故障: "+err.Error())
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if len(search.Hits) == 0 {
				if total == 0 {
					ReplyToSender(ctx, msg, groupStr+"未找到搜索结果")
				} else {
					ReplyToSender(ctx, msg, fmt.Sprintf("%s找到%d条结果, 但在当前页码并无结果", groupStr, total))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			hasSecond := len(search.Hits) >= 2
			best := d.Parent.Help.TextMap[search.Hits[0].ID]
			others := ""

			for _, i := range search.Hits {
				t := d.Parent.Help.TextMap[i.ID]
				if t.Group != "" && t.Group != HelpBuiltinGroup {
					others += fmt.Sprintf("[%s][%s]【%s:%s】 匹配度%.2f\n", i.ID, t.Group, t.PackageName, t.Title, i.Score)
				} else {
					others += fmt.Sprintf("[%s]【%s:%s】 匹配度%.2f\n", i.ID, t.PackageName, t.Title, i.Score)
				}
			}

			var showBest bool
			if hasSecond {
				offset := d.Parent.Help.GetShowBestOffset()
				val := search.Hits[1].Score - search.Hits[0].Score
				if val < 0 {
					val = -val
				}
				if val > float64(offset) {
					showBest = true
				}
				if best.Title == text {
					showBest = true
				}
			} else {
				showBest = true
			}

			var bestResult string
			if showBest {
				content := d.Parent.Help.GetContent(best, 0)
				bestResult = fmt.Sprintf("最优先结果%s:\n词条: %s:%s\n%s\n\n", groupStr, best.PackageName, best.Title, content)
			}

			prefix := d.Parent.Help.GetPrefixText()
			rplCurPage := fmt.Sprintf("本页结果:\n%s\n", others)
			rplDetailHint := "使用\".find <序号>\"可查看明细，如.find 123\n"
			// pgStart是下标闭左边界, 加1以获得序号; pgEnd是下标开右边界, 无需调整就是最后一条的序号
			rplPageNum := fmt.Sprintf("共%d条结果, 当前显示第%d页(第%d条 到 第%d条)\n", total, page, pgStart+1, pgEnd)
			rplPageHint := "使用\".find <词条> --page=<页码> 查看更多结果\n"
			ReplyToSender(ctx, msg, prefix+groupStr+bestResult+rplCurPage+rplDetailHint+rplPageNum+rplPageHint)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["查询"] = cmdFind
	d.CmdMap["査詢"] = cmdFind
	d.CmdMap["find"] = cmdFind

	helpForHelp := ".help // 查看本帮助\n" +
		".help 指令 // 查看某指令信息\n" +
		".help 扩展模块 // 查看扩展信息，如.help coc7\n" +
		".help 关键字 // 查看任意帮助，同.find\n" +
		".help reload // 重新加载帮助文档，需要Master权限"
	cmdHelp := &CmdItemInfo{
		Name:      "help",
		ShortHelp: helpForHelp,
		Help:      "帮助指令，用于查看指令帮助和helpdoc中录入的信息:\n" + helpForHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			arg := cmdArgs.GetArgN(1)
			if arg == "" {
				text := "海豹核心 " + VERSION.String() + "\n"
				text += "官网: sealdice.com" + "\n"
				text += "海豹群: 524364253" + "\n"
				text += DiceFormatTmpl(ctx, "核心:骰子帮助文本_附加说明")
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if strings.EqualFold(arg, "reload") {
				if ctx.PrivilegeLevel < 100 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master"))
				} else {
					dm := d.Parent
					if dm.JustForTest {
						ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if !dm.IsHelpReloading {
						dm.IsHelpReloading = true
						dm.Help.Close()

						dm.InitHelp()
						dm.AddHelpWithDice(dm.Dice[0])
						ReplyToSender(ctx, msg, "帮助文档已经重新装载")
					} else {
						ReplyToSender(ctx, msg, "帮助文档正在重新装载，请稍后...")
					}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			if cmdArgs.IsArgEqual(1, "骰主", "骰主信息") {
				masterMsg := DiceFormatTmpl(ctx, "核心:骰子帮助文本_骰主")
				ReplyToSender(ctx, msg, masterMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "协议", "使用协议") {
				masterMsg := DiceFormatTmpl(ctx, "核心:骰子帮助文本_协议")
				ReplyToSender(ctx, msg, masterMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "娱乐") {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子帮助文本_娱乐"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "其他", "其它") {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子帮助文本_其他"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if d.Parent.IsHelpReloading {
				ReplyToSender(ctx, msg, "帮助文档正在重新装载，请稍后...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			search, _, _, _, err := d.Parent.Help.Search(ctx, cmdArgs.CleanArgs, true, 1, 1, "")
			if err == nil {
				if len(search.Hits) > 0 {
					// 居然会出现 hits[0] 为nil的情况？？
					// a := d.Parent.ShortHelp.GetContent(search.Hits[0].ID)
					a := d.Parent.Help.TextMap[search.Hits[0].ID]
					content := d.Parent.Help.GetContent(a, 0)
					ReplyToSender(ctx, msg, fmt.Sprintf("%s:%s\n%s", a.PackageName, a.Title, content))
				} else {
					ReplyToSender(ctx, msg, "未找到搜索结果")
				}
			} else {
				ReplyToSender(ctx, msg, "搜索故障: "+err.Error())
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["help"] = cmdHelp

	cmdBot := &CmdItemInfo{
		Name:      "bot",
		ShortHelp: ".bot on/off/bye/quit // 开启、关闭、退群",
		Help:      "骰子管理:\n.bot on/off/bye[exit,quit] // 开启、关闭、退群",
		Raw:       true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			inGroup := msg.MessageType == "group"

			if inGroup {
				// 不响应裸指令选项
				if len(cmdArgs.At) < 1 && ctx.Dice.IgnoreUnaddressedBotCmd {
					return CmdExecuteResult{Matched: true, Solved: false}
				}
				// 不响应at其他人
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: false}
				}
			}

			if len(cmdArgs.Args) > 0 && !cmdArgs.IsArgEqual(1, "about") { //nolint:nestif
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: false}
				}

				cmdArgs.ChopPrefixToArgsWith("on", "off")

				matchNumber := func() (bool, bool) {
					txt := cmdArgs.GetArgN(2)
					if len(txt) >= 4 {
						if strings.HasSuffix(ctx.EndPoint.UserID, txt) {
							return true, txt != ""
						}
					}
					return false, txt != ""
				}

				isMe, exists := matchNumber()
				if exists && !isMe {
					return CmdExecuteResult{Matched: true, Solved: false}
				}

				if cmdArgs.IsArgEqual(1, "on") {
					if !(msg.Platform == "QQ-CH" || ctx.Dice.BotExtFreeSwitch || ctx.PrivilegeLevel >= 40) {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					SetBotOnAtGroup(ctx, msg.GroupID)
					// TODO：ServiceAtNew此处忽略是否合理？
					ctx.Group, _ = ctx.Session.ServiceAtNew.Load(msg.GroupID)
					ctx.IsCurGroupBotOn = true

					text := DiceFormatTmpl(ctx, "核心:骰子开启")
					if ctx.Group.LogOn {
						text += "\n请特别注意: 日志记录处于开启状态"
					}
					ReplyToSender(ctx, msg, text)

					return CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "off") {
					if !(msg.Platform == "QQ-CH" || ctx.Dice.BotExtFreeSwitch || ctx.PrivilegeLevel >= 40) {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					SetBotOffAtGroup(ctx, ctx.Group.GroupID)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子关闭"))
					return CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "bye", "exit", "quit") {
					if cmdArgs.GetArgN(2) != "" {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if ctx.PrivilegeLevel < 40 {
						if !cmdArgs.AmIBeMentioned {
							// 裸指令，如果当前群内开启，予以提示
							if ctx.IsCurGroupBotOn {
								ReplyToSender(ctx, msg, "[退群指令] 请@我使用这个命令，以进行确认")
							}
							return CmdExecuteResult{Matched: true, Solved: true}
						}
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子退群预告"))

					userName := ctx.Dice.Parent.TryGetUserName(msg.Sender.UserID)
					txt := fmt.Sprintf("指令退群: 于群组<%s>(%s)中告别，操作者:<%s>(%s)",
						ctx.Group.GroupName, msg.GroupID, userName, msg.Sender.UserID)
					d.Logger.Info(txt)
					ctx.Notice(txt)

					// SetBotOffAtGroup(ctx, ctx.Group.GroupID)
					time.Sleep(3 * time.Second)
					ctx.Group.DiceIDExistsMap.Delete(ctx.EndPoint.UserID)
					ctx.Group.UpdatedAtTime = time.Now().Unix()
					ctx.EndPoint.Adapter.QuitGroup(ctx, msg.GroupID)

					return CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "save") {
					d.Save(false)

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子保存设置"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			} else {
				inGroup := msg.MessageType == "group"

				if inGroup {
					// 不响应裸指令选项
					if len(cmdArgs.At) < 1 && ctx.Dice.IgnoreUnaddressedBotCmd {
						return CmdExecuteResult{Matched: true, Solved: false}
					}
					// 不响应at其他人
					if cmdArgs.SomeoneBeMentionedButNotMe {
						return CmdExecuteResult{Matched: true, Solved: false}
					}
				}

				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				activeCount := 0
				serveCount := 0
				// Pinenutn: Range模板 ServiceAtNew重构代码
				d.ImSession.ServiceAtNew.Range(func(_ string, gp *GroupInfo) bool {
					// Pinenutn: ServiceAtNew重构
					if gp.GroupID != "" &&
						!strings.HasPrefix(gp.GroupID, "PG-") &&
						gp.DiceIDExistsMap.Exists(ctx.EndPoint.UserID) {
						serveCount++
						if gp.DiceIDActiveMap.Exists(ctx.EndPoint.UserID) {
							activeCount++
						}
					}
					return true
				})

				onlineVer := ""
				/*if d.Parent.AppVersionOnline != nil {
					ver := d.Parent.AppVersionOnline
					// 如果当前不是最新版，那么提示
					if ver.VersionLatestCode != VERSION_CODE {
						onlineVer = "\n最新版本: " + ver.VersionLatestDetail + "\n"
					}
				}*/
				addonText := "God save the Lord of Astra"
				var groupWorkInfo, activeText string
				if inGroup {
					activeText = "关闭"
					if ctx.Group.IsActive(ctx) {
						activeText = "开启"
					}
					groupWorkInfo = "\n群内工作状态: " + activeText
				}

				VarSetValueInt64(ctx, "$t供职群数", int64(serveCount))
				VarSetValueInt64(ctx, "$t启用群数", int64(activeCount))
				VarSetValueStr(ctx, "$t群内工作状态", groupWorkInfo)
				VarSetValueStr(ctx, "$t群内工作状态_仅状态", activeText)
				ver := VERSION.String()
				arch := runtime.GOARCH
				if arch != "386" && arch != "amd64" {
					ver = fmt.Sprintf("%s %s", ver, arch)
				}
				baseText := fmt.Sprintf("SealDice %s%s\n%s", ver, onlineVer, addonText)
				extText := DiceFormatTmpl(ctx, "核心:骰子状态附加文本")
				if extText != "" {
					extText = "\n" + extText
				}
				text := baseText + extText

				ReplyToSender(ctx, msg, text)

			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	d.CmdMap["bot"] = cmdBot

	cmdSealBot := &CmdItemInfo{
		Name:      "sealbot",
		ShortHelp: ".sealbot 查看信息",
		Help:      "骰子管理:\n.sealbot 查看信息",
		Raw:       true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			inGroup := msg.MessageType == "group"

			if inGroup {
				// 不响应裸指令选项
				if len(cmdArgs.At) < 1 && ctx.Dice.IgnoreUnaddressedBotCmd {
					return CmdExecuteResult{Matched: true, Solved: false}
				}
				// 不响应at其他人
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: false}
				}
			}

			if cmdArgs.SomeoneBeMentionedButNotMe {
				return CmdExecuteResult{Matched: false, Solved: false}
			}

			activeCount := 0
			serveCount := 0
			// Pinenutn: Range模板 ServiceAtNew重构代码
			d.ImSession.ServiceAtNew.Range(func(_ string, gp *GroupInfo) bool {
				// Pinenutn: ServiceAtNew重构
				if gp.GroupID != "" &&
					!strings.HasPrefix(gp.GroupID, "PG-") &&
					gp.DiceIDExistsMap.Exists(ctx.EndPoint.UserID) {
					serveCount++
					if gp.DiceIDActiveMap.Exists(ctx.EndPoint.UserID) {
						activeCount++
					}
				}
				return true
			})

			onlineVer := ""
			if d.Parent.AppVersionOnline != nil {
				ver := d.Parent.AppVersionOnline
				// 如果当前不是最新版，那么提示
				if ver.VersionLatestCode != VERSION_CODE {
					onlineVer = "\n最新版本: " + ver.VersionLatestDetail + "\n"
				}
			}
			var groupWorkInfo, activeText string
			if inGroup {
				activeText = "关闭"
				if ctx.Group.IsActive(ctx) {
					activeText = "开启"
				}
				groupWorkInfo = "\n群内工作状态: " + activeText
			}

			VarSetValueInt64(ctx, "$t供职群数", int64(serveCount))
			VarSetValueInt64(ctx, "$t启用群数", int64(activeCount))
			VarSetValueStr(ctx, "$t群内工作状态", groupWorkInfo)
			VarSetValueStr(ctx, "$t群内工作状态_仅状态", activeText)
			ver := VERSION.String()
			arch := runtime.GOARCH
			if arch != "386" && arch != "amd64" {
				ver = fmt.Sprintf("%s %s", ver, arch)
			}
			baseText := fmt.Sprintf("SealDice %s%s", ver, onlineVer)
			extText := DiceFormatTmpl(ctx, "核心:骰子状态附加文本")
			if extText != "" {
				extText = "\n" + extText
			}
			text := baseText + extText

			ReplyToSender(ctx, msg, text)

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["sealbot"] = cmdSealBot

	helpForDismiss := ".dismiss // 退出当前群，主用于QQ，支持机器人的平台可以直接移出成员"
	cmdDismiss := &CmdItemInfo{
		Name:              "dismiss",
		ShortHelp:         helpForDismiss,
		Help:              "退群(映射到bot bye):\n" + helpForDismiss,
		Raw:               true,
		DisabledInPrivate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.SomeoneBeMentionedButNotMe {
				// 如果是别人被at，置之不理
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if !cmdArgs.AmIBeMentioned {
				// 裸指令，如果当前群内开启，予以提示
				if ctx.IsCurGroupBotOn {
					ReplyToSender(ctx, msg, "[退群指令] 请@我使用这个命令，以进行确认")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			rest := cmdArgs.GetArgN(1)
			if rest != "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			cmdArgs.Args = []string{"bye"}
			cmdArgs.RawArgs = "bye " + cmdArgs.RawArgs
			if rest != "" {
				cmdArgs.Args = append(cmdArgs.Args, rest)
			}
			return cmdBot.Solve(ctx, msg, cmdArgs)
		},
	}
	d.CmdMap["dismiss"] = cmdDismiss

	helpForSystem := ".system state/status //查看系统资源占用\n" +
		".system reload/reboot //重启骰子核心\n" +
		".system save //保存核心数据"
	cmdSystem := &CmdItemInfo{
		Name:      "system",
		ShortHelp: helpForSystem,
		Help:      "骰子管理：\n" + helpForSystem,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "state") || cmdArgs.IsArgEqual(1, "status") {
				activeCount := 0
				serveCount := 0
				// Pinenutn: Range模板 ServiceAtNew重构代码
				d.ImSession.ServiceAtNew.Range(func(_ string, gp *GroupInfo) bool {
					// Pinenutn: ServiceAtNew重构
					if gp.GroupID != "" &&
						!strings.HasPrefix(gp.GroupID, "PG-") &&
						gp.DiceIDExistsMap.Exists(ctx.EndPoint.UserID) {
						serveCount++
						if gp.DiceIDActiveMap.Exists(ctx.EndPoint.UserID) {
							activeCount++
						}
					}
					return true
				})
				cpuPercent, _ := cpu.Percent(time.Second, false)
				cpuInformation, _ := cpu.Info()
				diskInformation, _ := disk.Usage("C:")
				currentTime := time.Now().Format("2006-01-02 15:04:05")
				memInfo, _ := mem.VirtualMemory()
				ReplyToSender(ctx, msg, fmt.Sprintf("本地时间:%s\n所在群聊数:%d\n开启群聊数:%d\n内存占用:%s%%\nCPU型号:%s\nCPU占用:%s%%\nC盘剩余空间:%sGB\n", currentTime, serveCount, activeCount, fmt.Sprintf("%f", memInfo.UsedPercent), cpuInformation[0].ModelName, Float64SliceToString(cpuPercent), fmt.Sprintf("%f", float64(diskInformation.Free)/1024/1024/1024)))
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "reload") || cmdArgs.IsArgEqual(1, "reboot") {
				var dm = ctx.Dice.Parent
				if dm.JustForTest {
					ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				ReplyToSender(ctx, msg, "3秒后开始重启")
				time.Sleep(3 * time.Second)
				dm.RebootRequestChan <- 1
			} else if cmdArgs.IsArgEqual(1, "save") {
				d.Save(false)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子保存设置"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		},
	}
	d.CmdMap["system"] = cmdSystem

	readIDList := func(ctx *MsgContext, _ *Message, cmdArgs *CmdArgs) []string {
		var uidLst []string
		for _, i := range cmdArgs.At {
			if i.UserID == ctx.EndPoint.UserID {
				// 不许添加自己
				continue
			}
			uidLst = append(uidLst, i.UserID)
		}

		if len(cmdArgs.Args) > 1 {
			for _, i := range cmdArgs.Args[1:] {
				if i == "me" {
					uidLst = append(uidLst, ctx.Player.UserID)
					continue
				}
				uid := FormatDiceIDQQ(i)
				uidLst = append(uidLst, uid)
			}
		}
		return uidLst
	}

	botListHelp := ".botlist add @A @B @C // 标记群内其他机器人，以免发生误触和无限对话\n" +
		".botlist add @A @B --s  // 同上，不过骰子不会做出回复\n" +
		".botlist del @A @B @C // 去除机器人标记\n" +
		".botlist list/show // 查看当前列表"
	cmdBotList := &CmdItemInfo{
		Name:      "botlist",
		ShortHelp: botListHelp,
		Help:      "机器人列表:\n" + botListHelp,
		Raw:       true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			cmdArgs.ChopPrefixToArgsWith("add", "rm", "del", "show", "list")

			checkSlience := func() bool {
				return (!cmdArgs.AmIBeMentionedFirst) || cmdArgs.GetKwarg("s") != nil ||
					cmdArgs.GetKwarg("slience") != nil
			}

			reply := ""
			showHelp := false

			subCmd := cmdArgs.GetArgN(1)
			switch subCmd {
			case "add":
				allCount := 0
				existsCount := 0
				newCount := 0
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					allCount++
					if ctx.Group.BotList.Exists(uid) {
						existsCount++
					} else {
						ctx.Group.BotList.Store(uid, true)
						newCount++
					}
				}

				reply = fmt.Sprintf(
					"新增标记了%d/%d个帐号，这些账号将被视为机器人。\n因此他们被人@，或主动发出指令时，海豹将不会回复。\n另外对于botlist add/rm，如果群里有多个海豹，只有第一个被@的会回复，其余的执行指令但不回应",
					newCount, allCount,
				)
			case "del", "rm":
				allCount := 0
				existsCount := 0
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					allCount++
					if ctx.Group.BotList.Exists(uid) {
						existsCount++
						ctx.Group.BotList.Delete(uid)
					}
				}

				reply = fmt.Sprintf(
					"删除标记了%d/%d个帐号，这些账号将不再被视为机器人。\n海豹将继续回应他们的命令",
					existsCount, allCount,
				)
			case "list", "show":
				if cmdArgs.SomeoneBeMentionedButNotMe {
					break
				}

				text := ""
				ctx.Group.BotList.Range(func(k string, _ bool) bool {
					text += "- " + k + "\n"
					return true
				})
				if text == "" {
					text = "无"
				}
				reply = fmt.Sprintf("群内其他机器人列表:\n%s", text)
			default:
				showHelp = !cmdArgs.SomeoneBeMentionedButNotMe
			}

			// NOTE(Xiangze-Li): 不可使用 ctx.IsCurGroupBotOn, 因其将被 at 也视为开启
			if ctx.Group.IsActive(ctx) {
				if len(reply) > 0 {
					if !checkSlience() {
						ReplyToSender(ctx, msg, reply)
					} else {
						d.Logger.Infof("botlist 静默执行: " + reply)
					}
				}
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: showHelp}
			}
			if len(reply) > 0 {
				d.Logger.Infof("botlist 静默执行: " + reply)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["botlist"] = cmdBotList

	var (
		reloginFlag     bool
		reloginLastTime int64
		updateCode      = "0000"
	)

	var masterListHelp = `.master add me // 将自己标记为骰主
.master add @A @B // 将别人标记为骰主
.master del @A @B @C // 去除骰主标记
.master unlock <密码(在UI中查看)> // (当Master被人抢占时)清空骰主列表，并使自己成为骰主
.master list // 查看当前骰主列表
.master reboot // 重新启动(需要二次确认)
.master relogin // 30s后重新登录，有机会清掉风控(仅master可用)
.master backup // 做一次备份
.master reload deck/js/helpdoc // 重新加载牌堆/js/帮助文档
.master quitgroup <群组ID> [<理由>] // 从指定群组中退出，必须在同一平台使用
.master jsclear <插件ID> // 清除指定插件的存储，随后重载JS环境`

	cmdMaster := &CmdItemInfo{
		Name:          "master",
		ShortHelp:     masterListHelp,
		Help:          "骰主指令:\n" + masterListHelp,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			var subCmd string

			cmdArgs.ChopPrefixToArgsWith(
				"unlock", "rm", "del", "add", "checkupdate", "reboot", "backup", "reload",
			)
			ctx.DelegateText = ""
			subCmd = cmdArgs.GetArgN(1)

			if subCmd != "add" && subCmd != "del" && subCmd != "rm" {
				// 如果不是add/del/rm，那么就不需要代骰
				// 补充，在组内才这样，私聊不需要at
				if ctx.MessageType == "group" && (!cmdArgs.AmIBeMentionedFirst && len(cmdArgs.At) > 0) {
					return CmdExecuteResult{Matched: false, Solved: false}
				}
			}

			var pRequired int
			if len(ctx.Dice.DiceMasters) >= 1 {
				// 如果帐号没有UI:1001以外的master，所有人都是master
				count := 0
				for _, uid := range ctx.Dice.DiceMasters {
					if uid != "UI:1001" {
						count += 1
					}
				}

				if count >= 1 {
					pRequired = 100
				}
			}

			// 单独处理解锁指令
			if subCmd == "unlock" {
				// 特殊解锁指令
				code := cmdArgs.GetArgN(2)
				if ctx.Dice.UnlockCodeVerify(code) {
					ctx.Dice.MasterRefresh()
					ctx.Dice.MasterAdd(ctx.Player.UserID)

					ctx.Dice.UnlockCodeUpdate(true) // 强制刷新解锁码
					ReplyToSender(ctx, msg, "你已成为Master")
				} else {
					ReplyToSender(ctx, msg, "错误的解锁码")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if ctx.PrivilegeLevel < pRequired {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			switch subCmd {
			case "add":
				var count int
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					if uid != ctx.EndPoint.UserID {
						ctx.Dice.MasterAdd(uid)
						count++
					}
				}
				ctx.Dice.Save(false)
				ReplyToSender(ctx, msg, fmt.Sprintf("海豹将新增%d位master", count))
			case "del", "rm":
				var count int
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					if ctx.Dice.MasterRemove(uid) {
						count++
					}
				}
				ctx.Dice.Save(false)
				ReplyToSender(ctx, msg, fmt.Sprintf("海豹移除了%d名master", count))
			case "relogin":
				var kw *Kwarg

				if kw = cmdArgs.GetKwarg("cancel"); kw != nil {
					if reloginFlag {
						reloginFlag = false
						ReplyToSender(ctx, msg, "已取消重登录")
					} else {
						ReplyToSender(ctx, msg, "错误: 不存在能够取消的重新登录行为")
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				doRelogin := func() {
					reloginLastTime = time.Now().Unix()
					ReplyToSender(ctx, msg, "开始执行重新登录")
					reloginFlag = false
					time.Sleep(1 * time.Second)
					ctx.EndPoint.Adapter.DoRelogin()
				}

				if time.Now().Unix()-reloginLastTime < 5*60 {
					ReplyToSender(
						ctx,
						msg,
						fmt.Sprintf(
							"执行过不久，指令将在%d秒后可以使用",
							5*60-(time.Now().Unix()-reloginLastTime),
						),
					)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if kw = cmdArgs.GetKwarg("now"); kw != nil {
					doRelogin()
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				reloginFlag = true
				ReplyToSender(ctx, msg, "将在30s后重新登录，期间可以输入.master relogin --cancel解除\n若遭遇风控，可能会没有任何输出。静等或输入.master relogin --now立即执行\n此指令每5分钟只能执行一次，可能解除风控，也可能使骰子失联。后果自负")

				go func() {
					time.Sleep(30 * time.Second)
					if reloginFlag {
						doRelogin()
					}
				}()
			case "backup":
				ReplyToSender(ctx, msg, "开始备份数据")

				_, err := ctx.Dice.Parent.Backup(ctx.Dice.Parent.AutoBackupSelection, false)
				if err == nil {
					ReplyToSender(ctx, msg, "备份成功！请到UI界面(综合设置-备份)处下载备份，或在骰子backup目录下读取")
				} else {
					d.Logger.Error("骰子备份:", err)
					ReplyToSender(ctx, msg, "备份失败！错误已写入日志。可能是磁盘已满所致，建议立即进行处理！")
				}
			/*case "checkupdate":
			var dm = ctx.Dice.Parent
			if dm.JustForTest {
				ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if runtime.GOOS == "android" {
				ReplyToSender(ctx, msg, "检测到手机版，手机版海豹不支持指令更新，请手动下载新版本安装包")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if dm.ContainerMode {
				ReplyToSender(ctx, msg, "容器模式下禁止指令更新，请手动拉取最新镜像")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			code := cmdArgs.GetArgN(2)
			if code == "" {
				var text string
				dm.AppVersionOnline = nil
				dm.UpdateCheckRequestChan <- 1

				// 等待获取新版本，最多10s
				for i := 0; i < 5; i++ {
					time.Sleep(2 * time.Second)
					if dm.AppVersionOnline != nil {
						break
					}
				}

				if dm.AppVersionOnline != nil {
					text = fmt.Sprintf("当前本地版本为: %s\n当前线上版本为: %s", VERSION.String(), dm.AppVersionOnline.VersionLatestDetail)
					if dm.AppVersionCode != dm.AppVersionOnline.VersionLatestCode {
						updateCode = strconv.FormatInt(rand.Int63()%8999+1000, 10)
						text += fmt.Sprintf("\n如需升级，请输入.master checkupdate %s 确认进行升级\n升级将花费约2分钟，升级失败可能导致进程关闭，建议在接触服务器情况下操作。\n当前进程启动时间: %s", updateCode, time.Unix(dm.AppBootTime, 0).Format("2006-01-02 15:04:05"))
					}
				} else {
					text = fmt.Sprintf("当前本地版本为: %s\n当前线上版本为: %s", VERSION.String(), "未知")
				}
				ReplyToSender(ctx, msg, text)
				break
			}

			if code != updateCode || updateCode == "0000" {
				ReplyToSender(ctx, msg, "无效的升级指令码")
				break
			}

			ReplyToSender(ctx, msg, "开始下载新版本，完成后将自动进行一次备份")
			go func() {
				ret := <-dm.UpdateDownloadedChan

				if ctx.IsPrivate {
					ctx.Dice.UpgradeWindowID = msg.Sender.UserID
				} else {
					ctx.Dice.UpgradeWindowID = ctx.Group.GroupID
				}
				ctx.Dice.UpgradeEndpointID = ctx.EndPoint.ID
				ctx.Dice.Save(true)

				bakFn, _ := ctx.Dice.Parent.Backup(BackupSelectionAll, false)
				tmpPath := path.Join(os.TempDir(), bakFn)
				_ = os.MkdirAll(tmpPath, 0755)
				ctx.Dice.Logger.Infof("将备份文件复制到此路径: %s", tmpPath)
				_ = cp.Copy(path.Join(BackupDir, bakFn), tmpPath)

				if ret == "" {
					ReplyToSender(ctx, msg, "准备开始升级，服务即将离线")
				} else {
					ReplyToSender(ctx, msg, "升级失败，原因: "+ret)
				}
			}()
			dm.UpdateRequestChan <- d*/
			case "reboot":
				var dm = ctx.Dice.Parent
				if dm.JustForTest {
					ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				code := cmdArgs.GetArgN(2)
				if code == "" {
					updateCode = strconv.FormatInt(rand.Int63()%8999+1000, 10)
					text := fmt.Sprintf("进程重启:\n如需重启，请输入.master reboot %s 确认进行重启\n重启将花费约2分钟，失败可能导致进程关闭，建议在接触服务器情况下操作。\n当前进程启动时间: %s", updateCode, time.Unix(dm.AppBootTime, 0).Format("2006-01-02 15:04:05"))
					ReplyToSender(ctx, msg, text)
					break
				}

				if code == updateCode && updateCode != "0000" {
					ReplyToSender(ctx, msg, "3秒后开始重启")
					time.Sleep(3 * time.Second)
					dm.RebootRequestChan <- 1
				} else {
					ReplyToSender(ctx, msg, "无效的重启指令码")
				}
			case "list":
				text := ""
				for _, i := range ctx.Dice.DiceMasters {
					// uid := FormatDiceIdQQ(i)
					text += "- " + i + "\n"
				}
				if text == "" {
					text = "无"
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("Master列表:\n%s", text))
			case "reload":
				dice := ctx.Dice
				switch cmdArgs.GetArgN(2) {
				case "deck":
					DeckReload(dice)
					ReplyToSender(ctx, msg, "牌堆已重载")
				case "js":
					dice.JsReload()
					ReplyToSender(ctx, msg, "js已重载")
				case "help", "helpdoc":
					dm := dice.Parent
					if !dm.IsHelpReloading {
						dm.IsHelpReloading = true
						dm.Help.Close()
						dm.InitHelp()
						dm.AddHelpWithDice(dice)
						ReplyToSender(ctx, msg, "帮助文档已重载")
					} else {
						ReplyToSender(ctx, msg, "帮助文档正在重新装载")
					}
				}
			case "quitgroup":
				gid := cmdArgs.GetArgN(2)
				if gid == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				n := strings.Split(gid, ":") // 不验证是否合法，反正下面会检查是否在 ServiceAtNew
				platform := strings.Split(n[0], "-")[0]

				gp, ok := ctx.Session.ServiceAtNew.Load(gid)
				if !ok || len(n[0]) < 2 {
					ReplyToSender(ctx, msg, fmt.Sprintf("群组列表中没有找到%s", gid))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if msg.Platform != platform {
					ReplyToSender(ctx, msg, fmt.Sprintf("目标群组不在当前平台，请前往%s完成操作", platform))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 既然是骰主自己操作，就不通知了
				// 除非有多骰主……
				ReplyToSender(ctx, msg, fmt.Sprintf("收到指令，将在5秒后退出群组%s", gp.GroupID))

				txt := "注意，收到骰主指令，5秒后将从该群组退出。"
				wherefore := cmdArgs.GetArgN(3)
				if wherefore != "" {
					txt += fmt.Sprintf("原因: %s", wherefore)
				}

				ReplyGroup(ctx, &Message{GroupID: gp.GroupID}, txt)

				mctx := &MsgContext{
					MessageType: "group",
					Group:       gp,
					EndPoint:    ctx.EndPoint,
					Session:     ctx.Session,
					Dice:        ctx.Dice,
					IsPrivate:   false,
				}
				// SetBotOffAtGroup(mctx, gp.GroupID)
				time.Sleep(3 * time.Second)
				gp.DiceIDExistsMap.Delete(mctx.EndPoint.UserID)
				gp.UpdatedAtTime = time.Now().Unix()
				mctx.EndPoint.Adapter.QuitGroup(mctx, gp.GroupID)

				return CmdExecuteResult{Matched: true, Solved: true}
			case "jsclear":
				extName := cmdArgs.GetArgN(2)
				if extName == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				ext := ctx.Dice.ExtFind(extName)
				if ext == nil {
					ReplyToSender(ctx, msg, "没有找到插件"+extName)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if !ext.IsJsExt {
					ReplyToSender(ctx, msg, fmt.Sprintf("%s是内置模块，为了骰子的正常运行，暂不支持清除", extName))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				err := ClearExtStorage(ctx.Dice, ext, extName)
				if err != nil {
					ctx.Dice.Logger.Errorf("jsclear: %v", err)
					ReplyToSender(ctx, msg, "清除数据失败，请查看日志")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				d.JsReload()
				ReplyToSender(ctx, msg, fmt.Sprintf("已经清除%s数据，重新加载JS插件", extName))
				return CmdExecuteResult{Matched: true, Solved: true}
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["master"] = cmdMaster

	helpRoll := ".r <表达式> [<原因>] // 骰点指令\n.rh <表达式> <原因> // 暗骰"
	cmdRoll := &CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "roll",
		ShortHelp:               helpRoll,
		Help:                    "骰点:\n" + helpRoll,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			var text string
			var diceResult int64
			var diceResultExists bool
			var detail string

			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			ctx.SystemTemplate = ctx.Group.GetCharTemplate(ctx.Dice)
			if ctx.Dice.CommandCompatibleMode {
				if (cmdArgs.Command == "rd" || cmdArgs.Command == "rhd" || cmdArgs.Command == "rdh") && len(cmdArgs.Args) >= 1 {
					if m, _ := regexp.MatchString(`^\d|优势|劣势|\+|-`, cmdArgs.CleanArgs); m {
						if cmdArgs.IsSpaceBeforeArgs {
							cmdArgs.CleanArgs = "d " + cmdArgs.CleanArgs
						} else {
							cmdArgs.CleanArgs = "d" + cmdArgs.CleanArgs
						}
					}
				}
			}

			var r *VMResultV2m
			var commandInfoItems []any

			rollOne := func() *CmdExecuteResult {
				forWhat := ""
				var matched string

				if len(cmdArgs.Args) >= 1 { //nolint:nestif
					var err error
					r, detail, err = DiceExprEvalBase(ctx, cmdArgs.CleanArgs, RollExtraFlags{
						DefaultDiceSideNum: getDefaultDicePoints(ctx),
						DisableBlock:       true,
						V2Only:             true,
					})

					if r != nil && !r.IsCalculated() {
						forWhat = cmdArgs.CleanArgs

						defExpr := "d"
						if ctx.diceExprOverwrite != "" {
							defExpr = ctx.diceExprOverwrite
						}
						r, detail, err = DiceExprEvalBase(ctx, defExpr, RollExtraFlags{
							DefaultDiceSideNum: getDefaultDicePoints(ctx),
							DisableBlock:       true,
						})
					}

					if r != nil && r.TypeId == ds.VMTypeInt {
						diceResult = int64(r.MustReadInt())
						diceResultExists = true
					}

					if err == nil {
						matched = r.GetMatched()
						if forWhat == "" {
							forWhat = r.GetRestInput()
						}
					} else {
						errs := err.Error()
						if strings.HasPrefix(errs, "E1:") || strings.HasPrefix(errs, "E5:") || strings.HasPrefix(errs, "E6:") || strings.HasPrefix(errs, "E7:") || strings.HasPrefix(errs, "E8:") {
							ReplyToSender(ctx, msg, errs)
							return &CmdExecuteResult{Matched: true, Solved: true}
						}
						forWhat = cmdArgs.CleanArgs
					}
				}

				VarSetValueStr(ctx, "$t原因", forWhat)
				if forWhat != "" {
					forWhatText := DiceFormatTmpl(ctx, "核心:骰点_原因")
					VarSetValueStr(ctx, "$t原因句子", forWhatText)
				} else {
					VarSetValueStr(ctx, "$t原因句子", "")
				}

				if diceResultExists { //nolint:nestif
					detailWrap := ""
					if detail != "" {
						detailWrap = "=" + detail
						re := regexp.MustCompile(`\[((\d+)d\d+)\=(\d+)\]`)
						match := re.FindStringSubmatch(detail)
						if len(match) > 0 {
							num := match[2]
							if num == "1" && (match[1] == matched || match[1] == "1"+matched) {
								detailWrap = ""
							}
						}
					}

					// 指令信息标记
					item := map[string]interface{}{
						"expr":   matched,
						"result": diceResult,
						"reason": forWhat,
					}
					if forWhat == "" {
						delete(item, "reason")
					}
					commandInfoItems = append(commandInfoItems, item)

					VarSetValueStr(ctx, "$t表达式文本", matched)
					VarSetValueStr(ctx, "$t计算过程", detailWrap)
					VarSetValueInt64(ctx, "$t计算结果", diceResult)
				} else {
					var val int64
					var detail string
					dicePoints := getDefaultDicePoints(ctx)
					if ctx.diceExprOverwrite != "" {
						r, detail, _ = DiceExprEvalBase(ctx, cmdArgs.CleanArgs, RollExtraFlags{
							DefaultDiceSideNum: dicePoints,
							DisableBlock:       true,
						})
						if r != nil && r.TypeId == ds.VMTypeInt {
							valX, _ := r.ReadInt()
							val = int64(valX)
						}
					} else {
						r, _, _ = DiceExprEvalBase(ctx, "d", RollExtraFlags{
							DefaultDiceSideNum: dicePoints,
							DisableBlock:       true,
						})
						if r != nil && r.TypeId == ds.VMTypeInt {
							valX, _ := r.ReadInt()
							val = int64(valX)
						}
					}

					// 指令信息标记
					item := map[string]any{
						"expr":       fmt.Sprintf("D%d", dicePoints),
						"reason":     forWhat,
						"dicePoints": dicePoints,
						"result":     val,
					}
					if forWhat == "" {
						delete(item, "reason")
					}
					commandInfoItems = append(commandInfoItems, item)

					VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("D%d", dicePoints))
					VarSetValueStr(ctx, "$t计算过程", detail)
					VarSetValueInt64(ctx, "$t计算结果", val)
				}
				return nil
			}

			if cmdArgs.SpecialExecuteTimes > 1 {
				VarSetValueInt64(ctx, "$t次数", int64(cmdArgs.SpecialExecuteTimes))
				if cmdArgs.SpecialExecuteTimes > int(ctx.Dice.MaxExecuteTime) {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰点_轮数过多警告"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				var texts []string
				for i := 0; i < cmdArgs.SpecialExecuteTimes; i++ {
					ret := rollOne()
					if ret != nil {
						return *ret
					}
					texts = append(texts, DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
				}
				VarSetValueStr(ctx, "$t结果文本", strings.Join(texts, "\n"))
				text = DiceFormatTmpl(ctx, "核心:骰点_多轮")
			} else {
				ret := rollOne()
				if ret != nil {
					return *ret
				}
				VarSetValueStr(ctx, "$t结果文本", DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
				text = DiceFormatTmpl(ctx, "核心:骰点")
			}

			isHide := strings.Contains(cmdArgs.Command, "h")

			// 指令信息
			commandInfo := map[string]any{
				"cmd":    "roll",
				"pcName": ctx.Player.Name,
				"items":  commandInfoItems,
			}
			if isHide {
				commandInfo["hide"] = isHide
			}
			ctx.CommandInfo = commandInfo

			if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
				if ctx.PrivilegeLevel >= 40 {
					asm := r.GetAsmText()
					text += "\n" + asm
				}
			}

			if kw := cmdArgs.GetKwarg("ci"); kw != nil {
				info, err := json.Marshal(ctx.CommandInfo)
				if err == nil {
					text += "\n" + string(info)
				} else {
					text += "\n" + "指令信息无法序列化"
				}
			}

			if isHide {
				if msg.Platform == "QQ-CH" {
					ReplyToSender(ctx, msg, "QQ频道内尚不支持暗骰")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if ctx.Group != nil {
					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					} else {
						ctx.CommandHideFlag = ctx.Group.GroupID
						prefix := DiceFormatTmpl(ctx, "核心:暗骰_私聊_前缀")
						ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:暗骰_群内"))
						ReplyPerson(ctx, msg, prefix+text)
					}
				} else {
					ReplyToSender(ctx, msg, text)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpRollX := ".rx <表达式> <原因> // 骰点指令\n.rxh <表达式> <原因> // 暗骰"
	cmdRollX := &CmdItemInfo{
		Name:          "roll",
		ShortHelp:     helpRoll,
		Help:          "骰点(和r相同，但支持代骰):\n" + helpRollX,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			return cmdRoll.Solve(mctx, msg, cmdArgs)
		},
	}

	d.CmdMap["r"] = cmdRoll
	d.CmdMap["rd"] = cmdRoll
	d.CmdMap["roll"] = cmdRoll
	d.CmdMap["rh"] = cmdRoll
	d.CmdMap["rhd"] = cmdRoll
	d.CmdMap["rdh"] = cmdRoll
	d.CmdMap["rx"] = cmdRollX
	d.CmdMap["rxh"] = cmdRollX
	d.CmdMap["rhx"] = cmdRollX

	helpExt := ".ext // 查看扩展列表"
	cmdExt := &CmdItemInfo{
		Name:      "ext",
		ShortHelp: helpExt,
		Help:      "群扩展模块管理:\n" + helpExt,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			showList := func() {
				text := "检测到以下扩展(名称-版本-作者)：\n"
				for index, i := range ctx.Dice.ExtList {
					state := "关"
					for _, j := range ctx.Group.ActivatedExtList {
						if i.Name == j.Name {
							state = "开"
							break
						}
					}
					var officialMark string
					if i.Official {
						officialMark = "[官方]"
					}
					author := i.Author
					if author == "" {
						author = "<未注明>"
					}
					aliases := ""
					if len(i.Aliases) > 0 {
						aliases = "(" + strings.Join(i.Aliases, ",") + ")"
					}
					text += fmt.Sprintf("%d. [%s]%s%s %s - %s - %s\n", index+1, state, officialMark, i.Name, aliases, i.Version, author)
				}
				text += "使用命令: .ext <扩展名> on/off 可以在当前群开启或关闭某扩展。\n"
				text += "命令: .ext <扩展名> 可以查看扩展介绍及帮助"
				ReplyToSender(ctx, msg, text)
			}

			if len(cmdArgs.Args) == 0 {
				showList()
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			var last int
			if len(cmdArgs.Args) >= 2 {
				last = len(cmdArgs.Args)
			}

			//nolint:nestif
			if cmdArgs.IsArgEqual(1, "list") {
				showList()
			} else if cmdArgs.IsArgEqual(last, "on") {
				if !ctx.Dice.BotExtFreeSwitch && ctx.PrivilegeLevel < 40 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				checkConflict := func(ext *ExtInfo) []string {
					var actived []string
					for _, i := range ctx.Group.ActivatedExtList {
						actived = append(actived, i.Name)
					}

					if ext.ConflictWith != nil {
						var ret []string
						for _, i := range intersect.Simple(actived, ext.ConflictWith) {
							ret = append(ret, i.(string))
						}
						return ret
					}
					return []string{}
				}

				var extNames []string
				var conflictsAll []string
				for index := 0; index < len(cmdArgs.Args); index++ {
					extName := strings.ToLower(cmdArgs.Args[index])
					if i := d.ExtFind(extName); i != nil {
						extNames = append(extNames, extName)
						conflictsAll = append(conflictsAll, checkConflict(i)...)
						ctx.Group.ExtActive(i)
					}
				}

				if len(extNames) == 0 {
					ReplyToSender(ctx, msg, "输入的扩展类别名无效")
				} else {
					text := fmt.Sprintf("打开扩展 %s", strings.Join(extNames, ","))
					if len(conflictsAll) > 0 {
						text += "\n检测到可能冲突的扩展，建议关闭: " + strings.Join(conflictsAll, ",")
						text += "\n对于扩展中存在的同名指令，则越晚开启的扩展，优先级越高。"
					}
					ReplyToSender(ctx, msg, text)
				}
			} else if cmdArgs.IsArgEqual(last, "off") {
				if !ctx.Dice.BotExtFreeSwitch && ctx.PrivilegeLevel < 40 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				var closed []string
				var notfound []string
				for index := 0; index < len(cmdArgs.Args); index++ {
					extName := cmdArgs.Args[index]
					extName = d.ExtAliasToName(extName)
					ei := ctx.Group.ExtInactiveByName(extName)
					if ei != nil {
						closed = append(closed, ei.Name)
					} else {
						notfound = append(notfound, extName)
					}
				}

				var text string

				if len(closed) > 0 {
					text += fmt.Sprintf("关闭扩展: %s", strings.Join(closed, ","))
				} else {
					text += fmt.Sprintf(" 已关闭或未找到: %s", strings.Join(notfound, ","))
				}
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			} else {
				extName := cmdArgs.Args[0]
				if i := d.ExtFind(extName); i != nil {
					text := fmt.Sprintf("> [%s] 版本%s 作者%s\n", i.Name, i.Version, i.Author)
					i.callWithJsCheck(d, func() {
						ReplyToSender(ctx, msg, text+i.GetDescText(i))
					})
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["ext"] = cmdExt

	helpNN := ".nn // 查看当前角色名\n" +
		".nn <角色名> // 改为指定角色名，若有卡片不会连带修改\n" +
		".nn clr // 重置回群名片"
	cmdNN := &CmdItemInfo{
		Name:      "nn",
		ShortHelp: helpNN,
		Help:      "角色名设置:\n" + helpNN,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := strings.ToLower(cmdArgs.GetArgN(1))
			switch val {
			case "":
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_当前"))
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "clr", "reset":
				p := ctx.Player
				VarSetValueStr(ctx, "$t旧昵称", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t旧昵称_RAW", ctx.Player.Name)
				p.Name = msg.Sender.Nickname
				p.UpdatedAtTime = time.Now().Unix()
				VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t玩家_RAW", ctx.Player.Name)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_重置"))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
			default:
				p := ctx.Player
				VarSetValueStr(ctx, "$t旧昵称", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t旧昵称_RAW", ctx.Player.Name)

				p.Name = cmdArgs.Args[0]
				p.UpdatedAtTime = time.Now().Unix()
				VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t玩家_RAW", ctx.Player.Name)

				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_改名"))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
			}

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["nn"] = cmdNN

	d.CmdMap["userid"] = &CmdItemInfo{
		Name:      "userid",
		ShortHelp: ".userid // 查看当前帐号和群组ID",
		Help:      "查看ID:\n.userid // 查看当前帐号和群组ID",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			text := fmt.Sprintf("个人账号ID为 %s", ctx.Player.UserID)
			if !ctx.IsPrivate {
				text += fmt.Sprintf("\n当前群组ID为 %s", ctx.Group.GroupID)
			}

			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSet := ".set info// 查看当前面数设置\n" +
		".set dnd/coc // 设置群内骰子面数为20/100，并自动开启对应扩展 \n" +
		".set <面数> // 设置群内骰子面数\n" +
		".set clr // 清除群内骰子面数设置"
	cmdSet := &CmdItemInfo{
		Name:      "set",
		ShortHelp: helpSet,
		Help:      "设定骰子面数:\n" + helpSet,
		HelpFunc: func(isShort bool) string {
			text := ".set info // 查看当前面数设置\n"
			text += ".set <面数> // 设置群内骰子面数\n"
			text += ".set dnd // 设置群内骰子面数为20，并自动开启对应扩展\n"
			d.GameSystemMap.Range(func(key string, tmpl *GameSystemTemplate) bool {
				textHelp := fmt.Sprintf("设置群内骰子面数为%d，并自动开启对应扩展", tmpl.SetConfig.DiceSides)
				text += fmt.Sprintf(".set %s // %s\n", strings.Join(tmpl.SetConfig.Keys, "/"), textHelp)
				return true
			})
			text += `.set clr // 清除群内骰子面数设置`
			if isShort {
				return text
			}
			return "设定骰子面数:\n" + text
		},
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			p := ctx.Player
			isSetGroup := true
			my := cmdArgs.GetKwarg("my")
			if my != nil {
				isSetGroup = false
			}

			arg1 := cmdArgs.GetArgN(1)
			modSwitch := false
			if arg1 == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			tipText := "\n提示:"
			ctx.Dice.GameSystemMap.Range(func(key string, tmpl *GameSystemTemplate) bool {
				isMatch := false
				for _, k := range tmpl.SetConfig.Keys {
					if strings.EqualFold(arg1, k) {
						isMatch = true
						break
					}
				}

				if isMatch {
					modSwitch = true
					ctx.Group.System = key
					ctx.Group.DiceSideNum = tmpl.SetConfig.DiceSides
					ctx.Group.UpdatedAtTime = time.Now().Unix()
					tipText += tmpl.SetConfig.EnableTip

					// TODO: 命令该要进步啦
					cmdArgs.Args[0] = strconv.FormatInt(tmpl.SetConfig.DiceSides, 10)

					for _, name := range tmpl.SetConfig.RelatedExt {
						// 开启相关扩展
						ei := ctx.Dice.ExtFind(name)
						if ei != nil {
							ctx.Group.ExtActive(ei)
						}
					}
					return false
				}
				return true
			})

			num, err := strconv.ParseInt(cmdArgs.Args[0], 10, 64)
			if num < 0 {
				num = 0
			}
			//nolint:nestif
			if err == nil {
				if isSetGroup {
					ctx.Group.DiceSideNum = num
					if !modSwitch {
						if num == 20 {
							tipText += "20面骰。如果要进行DND游戏，建议执行.set dnd以确保开启dnd5e指令"
						} else if num == 100 {
							tipText += "100面骰。如果要进行COC游戏，建议执行.set coc以确保开启coc7指令"
						}
					}
					if tipText == "\n提示:" {
						tipText = ""
					}

					VarSetValueInt64(ctx, "$t群组骰子面数", ctx.Group.DiceSideNum)
					VarSetValueInt64(ctx, "$t当前骰子面数", getDefaultDicePoints(ctx))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认群组骰子面数")+tipText)
				} else {
					p.DiceSideNum = int(num)
					p.UpdatedAtTime = time.Now().Unix()
					VarSetValueInt64(ctx, "$t个人骰子面数", int64(ctx.Player.DiceSideNum))
					VarSetValueInt64(ctx, "$t当前骰子面数", getDefaultDicePoints(ctx))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数"))
				}
			} else {
				switch arg1 {
				case "clr":
					if isSetGroup {
						ctx.Group.DiceSideNum = 0
						ctx.Group.UpdatedAtTime = time.Now().Unix()
					} else {
						p.DiceSideNum = 0
						p.UpdatedAtTime = time.Now().Unix()
					}
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数_重置"))
				case "help":
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				case "info":
					ReplyToSender(ctx, msg, DiceFormat(ctx, `个人骰子面数: {$t个人骰子面数}\n`+
						`群组骰子面数: {$t群组骰子面数}\n当前骰子面数: {$t当前骰子面数}`))
				default:
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数_错误"))
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["set"] = cmdSet

	helpCh := ".pc new <角色名> // 新建角色并绑卡\n" +
		".pc tag [<角色名> | <角色序号>] // 当前群绑卡/解除绑卡(不填角色名)\n" +
		".pc untagAll [<角色名> | <角色序号>] // 全部群解绑(不填即当前卡)\n" +
		".pc list // 列出当前角色和序号\n" +
		".pc rename <新角色名> // 将当前绑定角色改名\n" +
		".pc rename <角色名|序号> <新角色名> // 将指定角色改名 \n" +
		// ".ch group // 列出各群当前绑卡\n" +
		".pc save [<角色名>] // [不绑卡]保存角色，角色名可省略\n" +
		".pc load (<角色名> | <角色序号>) // [不绑卡]加载角色\n" +
		".pc del/rm (<角色名> | <角色序号>) // 删除角色 角色序号可用pc list查询\n" +
		"> 注: 海豹各群数据独立(多张空白卡)，单群游戏不需要存角色。"

	cmdChar := &CmdItemInfo{
		Name:      "pc",
		ShortHelp: helpCh,
		Help:      "角色管理:\n" + helpCh,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) (result CmdExecuteResult) {
			cmdArgs.ChopPrefixToArgsWith("list", "lst", "load", "save", "del", "rm", "new", "tag", "untagAll", "rename")
			val1 := cmdArgs.GetArgN(1)
			am := d.AttrsManager

			defer func() {
				if err, ok := recover().(error); ok {
					ReplyToSender(ctx, msg, fmt.Sprintf("错误: %s\n", err.Error()))
					result = CmdExecuteResult{Matched: true, Solved: true}
				}
			}()

			getNicknameRaw := func(usePlayerName bool, tryIndex bool) string {
				// name := cmdArgs.GetArgN(2)
				name := cmdArgs.CleanArgsChopRest

				if tryIndex {
					index, err := strconv.ParseInt(name, 10, 64)
					if err == nil && index > 0 {
						items, _ := am.GetCharacterList(ctx.Player.UserID)
						if index <= int64(len(items)) {
							item := items[index-1]
							return item.Name
						}
					}
				}

				if usePlayerName && name == "" {
					name = ctx.Player.Name
				}
				name = strings.ReplaceAll(name, "\n", "")
				name = strings.ReplaceAll(name, "\r", "")

				if len(name) > 90 {
					name = name[:90]
				}
				return name
			}

			getNickname := func() string {
				return getNicknameRaw(true, true)
			}

			getBindingId := func() string {
				id, _ := am.CharGetBindingId(ctx.Group.GroupID, ctx.Player.UserID)
				return id
			}

			setCurPlayerName := func(name string) {
				ctx.Player.Name = name
				ctx.Player.UpdatedAtTime = time.Now().Unix()
			}

			switch val1 {
			case "list", "lst":
				list := lo.Must(am.GetCharacterList(ctx.Player.UserID))
				bindingId := getBindingId()

				var newChars []string
				for idx, item := range list {
					prefix := "[×]"
					if item.BindingGroupsNum > 0 {
						prefix = "[★]"
					}
					if bindingId == item.Id {
						prefix = "[√]"
					}
					suffix := ""
					if item.SheetType != "" {
						suffix = fmt.Sprintf(" #%s", item.SheetType)
					}

					// 格式参考:
					// 01[×] 张三 #dnd5e
					// 02[★] 李四 #coc7
					// 03[√] 王五 #coc7
					// 04[×] 赵六
					newChars = append(newChars, fmt.Sprintf("%2d %s %s%s", idx+1, prefix, item.Name, suffix))
				}

				if len(list) == 0 {
					ReplyToSender(ctx, msg, fmt.Sprintf("<%s>当前还没有角色列表", ctx.Player.Name))
				} else {
					ReplyToSender(ctx, msg, fmt.Sprintf("<%s>的角色列表为:\n%s\n[√]已绑 [×]未绑 [★]其他群绑定", ctx.Player.Name, strings.Join(newChars, "\n")))
				}
				return CmdExecuteResult{Matched: true, Solved: true}

			case "new":
				name := getNicknameRaw(true, false)

				VarSetValueStr(ctx, "$t角色名", name)
				if !am.CharCheckExists(ctx.Player.UserID, name) {
					item := lo.Must(am.CharNew(ctx.Player.UserID, name, ctx.Group.System))
					lo.Must0(am.CharBind(item.Id, ctx.Group.GroupID, ctx.Player.UserID))
					setCurPlayerName(name) // 修改当前角色名

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_新建"))
				} else {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_新建_已存在"))
				}

				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "rename":
				var charId string
				a := cmdArgs.GetArgN(2)
				b := cmdArgs.GetArgN(3)

				if b == "" {
					b = a
					charId = getBindingId()
				} else {
					charId, _ = am.CharIdGetByName(ctx.Player.UserID, a)
				}

				if a != "" && b != "" {
					if charId != "" {
						if !am.CharCheckExists(ctx.Player.UserID, b) {
							attrs := lo.Must(am.LoadById(charId))
							attrs.Name = b
							if charId == getBindingId() {
								// 如果是当前绑定的ID，连名字一起改
								setCurPlayerName(b)
							}
							attrs.LastModifiedTime = time.Now().Unix()
							attrs.SaveToDB(am.db, nil) // 直接保存
							ReplyToSender(ctx, msg, "操作完成")
						} else {
							ReplyToSender(ctx, msg, "此角色名已存在")
						}
					} else {
						ReplyToSender(ctx, msg, "未找到此角色")
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			case "tag":
				// 当不输入角色的时候，用当前角色填充，因此做到不写角色名就取消绑定的效果
				name := getNicknameRaw(false, true)

				VarSetValueStr(ctx, "$t角色名", name)
				if name != "" {
					VarSetValueStr(ctx, "$t角色名", name)
					charId := lo.Must(am.CharIdGetByName(ctx.Player.UserID, name))

					if charId == "" {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_失败"))
					} else {
						lo.Must0(am.CharBind(charId, ctx.Group.GroupID, ctx.Player.UserID))
						setCurPlayerName(name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_成功"))
					}
				} else {
					charId := getBindingId()

					if charId == "" {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_并未绑定"))
					} else {
						lo.Must0(am.CharBind("", ctx.Group.GroupID, ctx.Player.UserID))
						attrs := lo.Must(am.LoadById(charId))

						name := attrs.Name
						setCurPlayerName(name)
						VarSetValueStr(ctx, "$t角色名", name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_解除"))
					}
				}
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "load":
				name := getNicknameRaw(false, true)
				VarSetValueStr(ctx, "$t角色名", name)

				charId := lo.Must(am.CharIdGetByName(ctx.Player.UserID, name))
				attrsCur := lo.Must(d.AttrsManager.Load(ctx.Group.GroupID, ctx.Player.UserID))

				if attrsCur == nil {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
					// ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_序列化失败"))
				} else {
					attrs := lo.Must(am.LoadById(charId))

					attrsCur.Clear()
					attrs.Range(func(key string, value *ds.VMValue) bool {
						attrsCur.Store(key, value)
						return true
					})

					setCurPlayerName(name)

					if ctx.Player.AutoSetNameTemplate != "" {
						_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
					}

					VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_加载成功"))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "save":
				name := getNickname()

				if !am.CharCheckExists(ctx.Player.UserID, name) {
					newItem, _ := am.CharNew(ctx.Player.UserID, name, ctx.Group.System)
					attrs := lo.Must(am.Load(ctx.Group.GroupID, ctx.Player.UserID))

					if newItem != nil {
						attrsNew, err := am.LoadById(newItem.Id)
						if err != nil {
							// ReplyToSender(ctx, msg, fmt.Sprintf("错误: %s\n", err.Error()))
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_序列化失败"))
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						attrs.Range(func(key string, value *ds.VMValue) bool {
							attrsNew.Store(key, value)
							return true
						})

						VarSetValueStr(ctx, "$t角色名", name)
						VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))
						// replyToSender(ctx, msg, fmt.Sprintf("角色<%s>储存成功", Name))
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_储存成功"))
					} else {
						VarSetValueStr(ctx, "$t角色名", name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_储存失败_已绑定"))
					}
				} else {
					ReplyToSender(ctx, msg, "此角色名已存在")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "untagAll":
				var charId string
				name := getNicknameRaw(false, true)
				if name == "" {
					charId = getBindingId()
				} else {
					charId, _ = am.CharIdGetByName(ctx.Player.UserID, name)
				}

				var lst []string
				if charId != "" {
					lst = am.CharUnbindAll(charId)
				}

				for _, i := range lst {
					if i == ctx.Group.GroupID {
						ctx.Player.Name = msg.Sender.Nickname
						ctx.Player.UpdatedAtTime = time.Now().Unix()

						// TODO: 其他群的设置sn的怎么办？先不管了。。
						if ctx.Player.AutoSetNameTemplate != "" {
							_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
						}
					}
				}

				if len(lst) > 0 {
					ReplyToSender(ctx, msg, "绑定已全部解除:\n"+strings.Join(lst, "\n"))
				} else {
					ReplyToSender(ctx, msg, "这张卡片并未绑定到任何群")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "del", "rm":
				name := getNicknameRaw(false, true)
				if name == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				VarSetValueStr(ctx, "$t角色名", name)

				charId, _ := am.CharIdGetByName(ctx.Player.UserID, name)
				if charId == "" {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				lst := am.CharGetBindingGroupIdList(charId)
				if len(lst) > 0 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_删除失败_已绑定"))
					// ReplyToSender(ctx, msg, "角色已绑定到以下群:\n"+strings.Join(lst, "\n"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				err := am.CharDelete(charId)
				if err != nil {
					ReplyToSender(ctx, msg, "角色删除失败")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				VarSetValueStr(ctx, "$t角色名", name)
				VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))

				// 如果name原是序号，这里将被更新为角色名
				VarSetValueStr(ctx, "$t角色名", name)
				VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))

				text := DiceFormatTmpl(ctx, "核心:角色管理_删除成功")
				bindingCharId := getBindingId()
				if bindingCharId == charId {
					VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", msg.Sender.Nickname))
					text += "\n" + DiceFormatTmpl(ctx, "核心:角色管理_删除成功_当前卡")
					p := ctx.Player
					p.Name = msg.Sender.Nickname
				}
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		},
	}

	d.CmdMap["角色"] = cmdChar
	d.CmdMap["ch"] = cmdChar
	d.CmdMap["char"] = cmdChar
	d.CmdMap["character"] = cmdChar
	d.CmdMap["pc"] = cmdChar

	cmdReply := &CmdItemInfo{
		Name:      "reply",
		ShortHelp: ".reply on/off",
		Help:      "打开或关闭自定义回复:\n.reply on/off",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			switch val {
			case "on":
				onText := "开"
				if ctx.Group.ExtGetActive("reply") == nil {
					onText = "关"
				}
				extReply := ctx.Dice.ExtFind("reply")
				ctx.Group.ExtActive(extReply)
				VarSetValueStr(ctx, "$t旧群内状态", onText)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:开启自定义回复"))
			case "off":
				onText := "开"
				if ctx.Group.ExtGetActive("reply") == nil {
					onText = "关"
				}
				ctx.Group.ExtInactiveByName("reply")
				VarSetValueStr(ctx, "$t旧群内状态", onText)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:关闭自定义回复"))
			/*case "set":
			//CustomReplyItemType := strings.ReplaceAll(cmdArgs.GetArgN(2), "Type=", "")
			//CustomReplyItemContent := strings.ReplaceAll(cmdArgs.GetArgN(3), "Content=", "")
			CustomReplyFileName := "ReplyFromClient.yaml"
			// 初始化新的配置
			CustomReplyItemNewConfig := &ReplyConfig{
				Enable: true,
				Items:  make([]*ReplyItem, 1), // 初始化切片，长度为1
			}

			// 初始化第一个ReplyItem
			CustomReplyItemNewConfig.Items[0] = &ReplyItem{
				Enable: true,
			}

			// 保存配置
			SaveReplyConfig(d, CustomReplyFileName, CustomReplyItemNewConfig)*/
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["reply"] = cmdReply

	cmdStr := &CmdItemInfo{
		Name:      "str",
		ShortHelp: ".str<条目名称> <回执内容>",
		Help:      "打开或关闭自定义回复:\n.reply on/off",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			val := cmdArgs.GetArgN(1)
			val = strings.ToLower(val)
			subval := cmdArgs.GetArgN(2)
			subval = strings.ToLower(subval)
			cmdNum := len(cmdArgs.Args)
			switch val {
			case "selfname":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子名字", "风暴核心")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子名字"] {
							srcText := "风暴核心"
							d.TextMapRaw["核心"]["骰子名字"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子名字"] {
							d.TextMapRaw["核心"]["骰子名字"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "boton":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子开启", "{常量:APPNAME} 已启用 {常量:VERSION}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子开启"] {
							srcText := "{常量:APPNAME} 已启用 {常量:VERSION}"
							d.TextMapRaw["核心"]["骰子开启"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子开启"] {
							d.TextMapRaw["核心"]["骰子开启"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "botoff":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子关闭", "<{核心:骰子名字}> 停止服务")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子关闭"] {
							srcText := "<{核心:骰子名字}> 停止服务"
							d.TextMapRaw["核心"]["骰子关闭"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子关闭"] {
							d.TextMapRaw["核心"]["骰子关闭"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "replyon":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:开启自定义回复", "<{核心:骰子名字}>已在此群内开启自定义回复！\n群内工作状态:{$t旧群内状态}-->开")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["开启自定义回复"] {
							srcText := "<{核心:骰子名字}>已在此群内开启自定义回复！\n群内工作状态:{$t旧群内状态}-->开"
							d.TextMapRaw["核心"]["开启自定义回复"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["开启自定义回复"] {
							d.TextMapRaw["核心"]["开启自定义回复"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "replyoff":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:关闭自定义回复", "<{核心:骰子名字}>已在此群内关闭自定义回复！\n群内工作状态:{$t旧群内状态}-->关")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["关闭自定义回复"] {
							srcText := "<{核心:骰子名字}>已在此群内关闭自定义回复！\n群内工作状态:{$t旧群内状态}-->关"
							d.TextMapRaw["核心"]["关闭自定义回复"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["关闭自定义回复"] {
							d.TextMapRaw["核心"]["关闭自定义回复"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "addgroup":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子进群", "<{核心:骰子名字}> 已经就绪。可通过.help查看手册  \n[图]src=data/images/sealdice.png\nCOC/DND玩家可以使用.set coc/dnd在两种模式中切换\n已搭载自动重连，如遇风控不回可稍作等待")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子进群"] {
							srcText := "<{核心:骰子名字}> 已经就绪。可通过.help查看手册\n[图:data/images/sealdice.png]\nCOC/DND玩家可以使用.set coc/dnd在两种模式中切换\n已搭载自动重连，如遇风控不回可稍作等待"
							d.TextMapRaw["核心"]["骰子进群"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子进群"] {
							d.TextMapRaw["核心"]["骰子进群"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "addfriend":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子成为好友", "<{核心:骰子名字}> 已经就绪。可通过.help查看手册，请拉群测试，私聊容易被企鹅吃掉。\n[图]src=data/images/sealdice.png")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子成为好友"] {
							srcText := "<{核心:骰子名字}> 已经就绪。可通过.help查看手册，请拉群测试，私聊容易被企鹅吃掉。\n[图:data/images/sealdice.png]"
							d.TextMapRaw["核心"]["骰子成为好友"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子成为好友"] {
							d.TextMapRaw["核心"]["骰子成为好友"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "groupexitannounce":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子退群预告", "收到指令，5s后将退出当前群组")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子退群预告"] {
							srcText := "收到指令，5s后将退出当前群组"
							d.TextMapRaw["核心"]["骰子退群预告"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子退群预告"] {
							d.TextMapRaw["核心"]["骰子退群预告"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "groupexit":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子自动退群告别语", "由于长时间不使用，{核心:骰子名字}将退出本群，感谢您的使用。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子自动退群告别语"] {
							srcText := "由于长时间不使用，{核心:骰子名字}将退出本群，感谢您的使用。"
							d.TextMapRaw["核心"]["骰子自动退群告别语"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子自动退群告别语"] {
							d.TextMapRaw["核心"]["骰子自动退群告别语"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "savesetup":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子保存设置", "数据已保存")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子保存设置"] {
							srcText := "数据已保存"
							d.TextMapRaw["核心"]["骰子保存设置"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子保存设置"] {
							d.TextMapRaw["核心"]["骰子保存设置"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "additionalstatus":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰子状态附加文本", "供职于{$t供职群数}个群，其中{$t启用群数}个处于开启状态。{$t群内工作状态}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰子状态附加文本"] {
							srcText := "供职于{$t供职群数}个群，其中{$t启用群数}个处于开启状态。{$t群内工作状态}"
							d.TextMapRaw["核心"]["骰子状态附加文本"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰子状态附加文本"] {
							d.TextMapRaw["核心"]["骰子状态附加文本"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "reasonofrollprefix":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰点_原因", "由于{$t原因}，")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰点_原因"] {
							srcText := "由于{$t原因}，"
							d.TextMapRaw["核心"]["骰点_原因"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰点_原因"] {
							d.TextMapRaw["核心"]["骰点_原因"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "rolldiceeqvt":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰点_单项结果文本", "{$t表达式文本}{$t计算过程}={$t计算结果}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰点_单项结果文本"] {
							srcText := "{$t表达式文本}{$t计算过程}={$t计算结果}"
							d.TextMapRaw["核心"]["骰点_单项结果文本"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰点_单项结果文本"] {
							d.TextMapRaw["核心"]["骰点_单项结果文本"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "rolldice":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰点", "{$t原因句子}{$t玩家}掷出了 {$t结果文本}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰点"] {
							srcText := "{$t原因句子}{$t玩家}掷出了 {$t结果文本}"
							d.TextMapRaw["核心"]["骰点"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰点"] {
							d.TextMapRaw["核心"]["骰点"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "rollmultidice":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:骰点_多轮", "{$t原因句子}{$t玩家}掷骰{$t次数}次:\n{$t结果文本}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["骰点_多轮"] {
							srcText := "{$t原因句子}{$t玩家}掷骰{$t次数}次:\n{$t结果文本}"
							d.TextMapRaw["核心"]["骰点_多轮"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["骰点_多轮"] {
							d.TextMapRaw["核心"]["骰点_多轮"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pcnewemptycard":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_新建", "新建角色且自动绑定: {$t角色名}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_新建"] {
							srcText := "新建角色且自动绑定: {$t角色名}"
							d.TextMapRaw["核心"]["角色管理_新建"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_新建"] {
							d.TextMapRaw["核心"]["角色管理_新建"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pcnameexist":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_新建_已存在", "已存在同名角色")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_新建_已存在"] {
							srcText := "已存在同名角色"
							d.TextMapRaw["核心"]["角色管理_新建_已存在"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_新建_已存在"] {
							d.TextMapRaw["核心"]["角色管理_新建_已存在"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardset":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_绑定_成功", "切换角色\"{$t角色名}\"，绑定成功")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_成功"] {
							srcText := "切换角色\"{$t角色名}\"，绑定成功"
							d.TextMapRaw["核心"]["角色管理_绑定_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_成功"] {
							d.TextMapRaw["核心"]["角色管理_绑定_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardtagnotexist":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_绑定_失败", "角色\"{$t角色名}\"绑定失败，角色不存在")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_失败"] {
							srcText := "角色\"{$t角色名}\"绑定失败，角色不存在"
							d.TextMapRaw["核心"]["角色管理_绑定_失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_失败"] {
							d.TextMapRaw["核心"]["角色管理_绑定_失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccarduntag":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_绑定_解除", "角色\"{$t角色名}\"绑定已解除，切换至群内角色卡")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_解除"] {
							srcText := "角色\"{$t角色名}\"绑定已解除，切换至群内角色卡"
							d.TextMapRaw["核心"]["角色管理_绑定_解除"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_解除"] {
							d.TextMapRaw["核心"]["角色管理_绑定_解除"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardnotagged":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_绑定_并未绑定", "当前群内并未绑定角色")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_并未绑定"] {
							srcText := "当前群内并未绑定角色"
							d.TextMapRaw["核心"]["角色管理_绑定_并未绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_绑定_并未绑定"] {
							d.TextMapRaw["核心"]["角色管理_绑定_并未绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pcloaded":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_加载成功", "角色{$t玩家}加载成功，欢迎回来")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_加载成功"] {
							srcText := "角色{$t玩家}加载成功，欢迎回来"
							d.TextMapRaw["核心"]["角色管理_加载成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_加载成功"] {
							d.TextMapRaw["核心"]["角色管理_加载成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardalreadytagged":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_加载失败_已绑定", "当前群内是绑卡状态，请解除绑卡后进行此操作！")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_加载失败_已绑定"] {
							srcText := "当前群内是绑卡状态，请解除绑卡后进行此操作！"
							d.TextMapRaw["核心"]["角色管理_加载失败_已绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_加载失败_已绑定"] {
							d.TextMapRaw["核心"]["角色管理_加载失败_已绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardnotexist":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_角色不存在", "无法加载/删除角色：你所指定的角色不存在")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_角色不存在"] {
							srcText := "无法加载/删除角色：你所指定的角色不存在"
							d.TextMapRaw["核心"]["角色管理_角色不存在"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_角色不存在"] {
							d.TextMapRaw["核心"]["角色管理_角色不存在"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardnotserialized":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_序列化失败", "无法加载/保存角色：序列化失败")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_序列化失败"] {
							srcText := "无法加载/保存角色：序列化失败"
							d.TextMapRaw["核心"]["角色管理_序列化失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_序列化失败"] {
							d.TextMapRaw["核心"]["角色管理_序列化失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardsaved":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_储存成功", "角色\"{$t角色名}\"储存成功\n注: 非秘密团不用开团前存卡，跑团后save即可")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_储存成功"] {
							srcText := "角色\"{$t角色名}\"储存成功\n注: 非秘密团不用开团前存卡，跑团后save即可"
							d.TextMapRaw["核心"]["角色管理_储存成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_储存成功"] {
							d.TextMapRaw["核心"]["角色管理_储存成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardnotsavedbuttagged":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_储存失败_已绑定", "角色卡\"{$t角色名}\"是绑定状态，无法进行save操作")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_储存_失败_已绑定"] {
							srcText := "角色卡\"{$t角色名}\"是绑定状态，无法进行save操作"
							d.TextMapRaw["核心"]["角色管理_储存失败_已绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_储存失败_已绑定"] {
							d.TextMapRaw["核心"]["角色管理_储存失败_已绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccarddel":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_删除成功", "角色\"{$t角色名}\"删除成功")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_删除成功"] {
							srcText := "角色\"{$t角色名}\"删除成功"
							d.TextMapRaw["核心"]["角色管理_删除成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_删除成功"] {
							d.TextMapRaw["核心"]["角色管理_删除成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardnotdelbuttagged":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_删除失败_已绑定", "角色卡\"{$t角色名}\"是绑定状态，\".pc untagAll {$t角色名}\"解除绑卡后再操作吧")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_删除失败_已绑定"] {
							srcText := "角色卡\"{$t角色名}\"是绑定状态，\".pc untagAll {$t角色名}\"解除绑卡后再操作吧"
							d.TextMapRaw["核心"]["角色管理_删除失败_已绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_删除失败_已绑定"] {
							d.TextMapRaw["核心"]["角色管理_删除失败_已绑定"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}
			case "pccardcurrentdel":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:角色管理_删除成功_当前卡", "由于你删除的角色是当前角色，昵称和属性将被一同清空")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["角色管理_删除成功_当前卡"] {
							srcText := "由于你删除的角色是当前角色，昵称和属性将被一同清空"
							d.TextMapRaw["核心"]["角色管理_删除成功_当前卡"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["角色管理_删除成功_当前卡"] {
							d.TextMapRaw["核心"]["角色管理_删除成功_当前卡"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "notadmin":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:提示_无权限_非master/管理", "你不是管理员或master")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["提示_无权限_非master/管理"] {
							srcText := "你不是管理员或master"
							d.TextMapRaw["核心"]["提示_无权限_非master/管理"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["提示_无权限_非master/管理"] {
							d.TextMapRaw["核心"]["提示_无权限_非master/管理"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "notmaster":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:提示_无权限_非master", "你不是master")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["提示_无权限_非master"] {
							srcText := "你不是master"
							d.TextMapRaw["核心"]["提示_无权限_非master"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["提示_无权限_非master"] {
							d.TextMapRaw["核心"]["提示_无权限_非master"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "sendmsgmaster":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:留言_已记录", "您的留言已被记录，另外注意不要滥用此功能，祝您生活愉快，再会。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["留言_已记录"] {
							srcText := "您的留言已被记录，另外注意不要滥用此功能，祝您生活愉快，再会。"
							d.TextMapRaw["核心"]["留言_已记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["留言_已记录"] {
							d.TextMapRaw["核心"]["留言_已记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "admingroupexit":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:提示_手动退群前缀", "因长期不使用等原因，骰主后台操作退群")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["提示_手动退群前缀"] {
							srcText := "因长期不使用等原因，骰主后台操作退群"
							d.TextMapRaw["核心"]["提示_手动退群前缀"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["提示_手动退群前缀"] {
							d.TextMapRaw["核心"]["提示_手动退群前缀"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "censornotice":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:拦截_警告内容_提醒级", "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_提醒级"] {
							srcText := "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！"
							d.TextMapRaw["核心"]["拦截_警告内容_提醒级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_提醒级"] {
							d.TextMapRaw["核心"]["拦截_警告内容_提醒级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "censorcaution":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:拦截_警告内容_注意级", "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_注意级"] {
							srcText := "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！"
							d.TextMapRaw["核心"]["拦截_警告内容_注意级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_注意级"] {
							d.TextMapRaw["核心"]["拦截_警告内容_注意级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "censorwarning":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:拦截_警告内容_警告级", "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_警告级"] {
							srcText := "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！"
							d.TextMapRaw["核心"]["拦截_警告内容_警告级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_警告级"] {
							d.TextMapRaw["核心"]["拦截_警告内容_警告级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "censordanger":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:拦截_警告内容_危险级", "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_危险级"] {
							srcText := "你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！"
							d.TextMapRaw["核心"]["拦截_警告内容_危险级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["拦截_警告内容_危险级"] {
							d.TextMapRaw["核心"]["拦截_警告内容_危险级"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "spamperson":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:刷屏_警告内容_个人", "警告：您的指令频率过高，请注意。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["刷屏_警告内容_个人"] {
							srcText := "警告：您的指令频率过高，请注意。"
							d.TextMapRaw["核心"]["刷屏_警告内容_个人"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["刷屏_警告内容_个人"] {
							d.TextMapRaw["核心"]["刷屏_警告内容_个人"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "spamgroup":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:刷屏_警告内容_群组", "警告：该群组内指令频率过高，请注意。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["刷屏_警告内容_群组"] {
							srcText := "警告：该群组内指令频率过高，请注意。"
							d.TextMapRaw["核心"]["刷屏_警告内容_群组"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["刷屏_警告内容_群组"] {
							d.TextMapRaw["核心"]["刷屏_警告内容_群组"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "blockcommanrecieved":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:拦截_完全拦截_收到的指令", "命令包含不当内容，{核心:骰子名字}拒绝响应。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["拦截_完全拦截_收到的指令"] {
							srcText := "命令包含不当内容，{核心:骰子名字}拒绝响应。"
							d.TextMapRaw["核心"]["拦截_完全拦截_收到的指令"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["拦截_完全拦截_收到的指令"] {
							d.TextMapRaw["核心"]["拦截_完全拦截_收到的指令"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "blockmessagetosend":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:拦截_完全拦截_发出的消息", "试图使{核心:骰子名字}回复不当内容，拒绝响应。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["拦截_完全拦截_发出的消息"] {
							srcText := "试图使{核心:骰子名字}回复不当内容，拒绝响应。"
							d.TextMapRaw["核心"]["拦截_完全拦截_发出的消息"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["拦截_完全拦截_发出的消息"] {
							d.TextMapRaw["核心"]["拦截_完全拦截_发出的消息"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "blacktriggerperson":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:黑名单触发_当事人", "提醒：你引发了黑名单事件:\n{$t黑名单事件}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["黑名单触发_当事人"] {
							srcText := "提醒：你引发了黑名单事件:\n{$t黑名单事件}"
							d.TextMapRaw["核心"]["黑名单触发_当事人"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["黑名单触发_当事人"] {
							d.TextMapRaw["核心"]["黑名单触发_当事人"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "blacktriggergroup":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:黑名单触发_所在群", "提醒：当前群组发生了黑名单事件:\n{$t黑名单事件}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["黑名单触发_所在群"] {
							srcText := "提醒：当前群组发生了黑名单事件:\n{$t黑名单事件}"
							d.TextMapRaw["核心"]["黑名单触发_所在群"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["黑名单触发_所在群"] {
							d.TextMapRaw["核心"]["黑名单触发_所在群"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "blacktriggerinviter":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "核心:黑名单触发_邀请人", "提醒: 你邀请的骰子在群组<{$t事发群名}>({$t事发群号})中遭遇黑名单事件:\n{$t黑名单事件}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["核心"]["黑名单触发_邀请人"] {
							srcText := "提醒: 你邀请的骰子在群组<{$t事发群名}>({$t事发群号})中遭遇黑名单事件:\n{$t黑名单事件}"
							d.TextMapRaw["核心"]["黑名单触发_邀请人"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["核心"]["黑名单触发_邀请人"] {
							d.TextMapRaw["核心"]["黑名单触发_邀请人"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "jrrp":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "娱乐:今日人品", "{$t玩家} 今日人品为{$t人品}，{%\n    $t人品 > 95 ? '人品爆表！',\n    $t人品 > 80 ? '运气还不错！',\n    $t人品 > 50 ? '人品还行吧',\n    $t人品 > 10 ? '今天不太行',\n    1 ? '流年不利啊！'\n%}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["娱乐"]["今日人品"] {
							srcText := "{$t玩家} 今日人品为{$t人品}，{%\n    $t人品 > 95 ? '人品爆表！',\n    $t人品 > 80 ? '运气还不错！',\n    $t人品 > 50 ? '人品还行吧',\n    $t人品 > 10 ? '今天不太行',\n    1 ? '流年不利啊！'\n%}"
							d.TextMapRaw["娱乐"]["今日人品"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["娱乐"]["今日人品"] {
							d.TextMapRaw["娱乐"]["今日人品"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "decklist":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:抽牌_列表", "{$t原始列表}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["抽牌_列表"] {
							srcText := "{$t原始列表}"
							d.TextMapRaw["其它"]["抽牌_列表"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["抽牌_列表"] {
							d.TextMapRaw["其它"]["抽牌_列表"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "drawkey":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:抽牌_列表", "{$t原始列表}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["抽牌_列表"] {
							srcText := "{$t原始列表}"
							d.TextMapRaw["其它"]["抽牌_列表"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["抽牌_列表"] {
							d.TextMapRaw["其它"]["抽牌_列表"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "nodeck":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:抽牌_列表_没有牌组", "呃，没有发现任何牌组")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["抽牌_列表_没有牌组"] {
							srcText := "呃，没有发现任何牌组"
							d.TextMapRaw["其它"]["抽牌_列表_没有牌组"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["抽牌_列表_没有牌组"] {
							d.TextMapRaw["其它"]["抽牌_列表_没有牌组"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "deckcitenotfound":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:抽牌_找不到牌组", "找不到这个牌组")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["抽牌_找不到牌组"] {
							srcText := "找不到这个牌组"
							d.TextMapRaw["其它"]["抽牌_找不到牌组"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["抽牌_找不到牌组"] {
							d.TextMapRaw["其它"]["抽牌_找不到牌组"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "deckcitenotfoundbuthavesimilar":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:抽牌_找不到牌组_存在类似", "未找到牌组，但发现一些相似的:")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["抽牌_找不到牌组_存在类似"] {
							srcText := "未找到牌组，但发现一些相似的:"
							d.TextMapRaw["其它"]["抽牌_找不到牌组_存在类似"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["抽牌_找不到牌组_存在类似"] {
							d.TextMapRaw["其它"]["抽牌_找不到牌组_存在类似"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "deckspliter":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:抽牌_分隔符", `\n\n`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["抽牌_分隔符"] {
							srcText := `\n\n`
							d.TextMapRaw["其它"]["抽牌_分隔符"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["抽牌_分隔符"] {
							d.TextMapRaw["其它"]["抽牌_分隔符"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "deckresultprefix":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:抽牌_结果前缀", ``)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["抽牌_结果前缀"] {
							srcText := ``
							d.TextMapRaw["其它"]["抽牌_结果前缀"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["抽牌_结果前缀"] {
							d.TextMapRaw["其它"]["抽牌_结果前缀"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "randomnamegenerate":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:随机名字", "为{$t玩家}生成以下名字：\n{$t随机名字文本}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["随机名字"] {
							srcText := "为{$t玩家}生成以下名字：\n{$t随机名字文本}"
							d.TextMapRaw["其它"]["随机名字"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["随机名字"] {
							d.TextMapRaw["其它"]["随机名字"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "randomnamespliter":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:随机名字_分隔符", "、")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["随机名字_分隔符"] {
							srcText := "、"
							d.TextMapRaw["其它"]["随机名字_分隔符"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["随机名字_分隔符"] {
							d.TextMapRaw["其它"]["随机名字_分隔符"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "poke":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:戳一戳", "{核心:骰子名字}咕踊了一下")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["戳一戳"] {
							srcText := "{核心:骰子名字}咕踊了一下"
							d.TextMapRaw["其它"]["戳一戳"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["戳一戳"] {
							d.TextMapRaw["其它"]["戳一戳"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "ping":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "其它:ping响应", "pong！这里是{核心:骰子名字}{$t请求结果}")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["其它"]["ping响应"] {
							srcText := "pong！这里是{核心:骰子名字}{$t请求结果}"
							d.TextMapRaw["其它"]["ping响应"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["其它"]["ping响应"] {
							d.TextMapRaw["其它"]["ping响应"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "lognew":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_新建", `新的故事开始了，祝旅途愉快！\n记录已经开启`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_新建"] {
							srcText := `新的故事开始了，祝旅途愉快！\n记录已经开启`
							d.TextMapRaw["日志"]["记录_新建"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_新建"] {
							d.TextMapRaw["日志"]["记录_新建"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logon":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_开启_成功", `故事"{$t记录名称}"的记录已经继续开启，当前已记录文本{$t当前记录条数}`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_开启_成功"] {
							srcText := `故事"{$t记录名称}"的记录已经继续开启，当前已记录文本{$t当前记录条数}`
							d.TextMapRaw["日志"]["记录_开启_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_开启_成功"] {
							d.TextMapRaw["日志"]["记录_开启_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logonsuccess":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_开启_成功", `故事"{$t记录名称}"的记录已经继续开启，当前已记录文本{$t当前记录条数}`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_开启_成功"] {
							srcText := `故事"{$t记录名称}"的记录已经继续开启，当前已记录文本{$t当前记录条数}`
							d.TextMapRaw["日志"]["记录_开启_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_开启_成功"] {
							d.TextMapRaw["日志"]["记录_开启_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logonfailnolog":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_开启_失败_无此记录", `无法继续，没能找到记录: {$t记录名称}`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_无此记录"] {
							srcText := `无法继续，没能找到记录: {$t记录名称}`
							d.TextMapRaw["日志"]["记录_开启_失败_无此记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_无此记录"] {
							d.TextMapRaw["日志"]["记录_开启_失败_无此记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logonfailnotnew":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_开启_失败_尚未新建", `找不到记录，请使用.log new新建记录`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_尚未新建"] {
							srcText := `找不到记录，请使用.log new新建记录`
							d.TextMapRaw["日志"]["记录_开启_失败_尚未新建"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_尚未新建"] {
							d.TextMapRaw["日志"]["记录_开启_失败_尚未新建"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logonalready":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_开启_失败_未结束的记录", `当前已有记录中的日志{$t记录名称}，请先将其结束。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"] {
							srcText := `当前已有记录中的日志{$t记录名称}，请先将其结束。`
							d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"] {
							d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logonfailunfinished":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_开启_失败_未结束的记录", `当前已有记录中的日志{$t记录名称}，请先将其结束。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"] {
							srcText := `当前已有记录中的日志{$t记录名称}，请先将其结束。`
							d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"] {
							d.TextMapRaw["日志"]["记录_开启_失败_未结束的记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logoff":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_关闭_成功", `当前记录"{$t记录名称}"已经暂停，已记录文本{$t当前记录条数}条\n结束故事并传送日志请用.log end`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_关闭_成功"] {
							srcText := `当前记录"{$t记录名称}"已经暂停，已记录文本{$t当前记录条数}条\n结束故事并传送日志请用.log end`
							d.TextMapRaw["日志"]["记录_关闭_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_关闭_成功"] {
							d.TextMapRaw["日志"]["记录_关闭_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logoffsuccess":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_关闭_成功", `当前记录"{$t记录名称}"已经暂停，已记录文本{$t当前记录条数}条\n结束故事并传送日志请用.log end`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_关闭_成功"] {
							srcText := `当前记录"{$t记录名称}"已经暂停，已记录文本{$t当前记录条数}条\n结束故事并传送日志请用.log end`
							d.TextMapRaw["日志"]["记录_关闭_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_关闭_成功"] {
							d.TextMapRaw["日志"]["记录_关闭_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logofffail":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_关闭_失败", `没有找到正在进行的记录，已经是关闭状态。这可能表示您忘记了开启记录。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_关闭_失败"] {
							srcText := `没有找到正在进行的记录，已经是关闭状态。这可能表示您忘记了开启记录。`
							d.TextMapRaw["日志"]["记录_关闭_失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_关闭_失败"] {
							d.TextMapRaw["日志"]["记录_关闭_失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logexportnotcertainlog":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_取出_未指定记录", `命令格式错误：当前没有开启状态的记录，或没有通过参数指定要取出的日志。请参考帮助。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_取出_未指定记录"] {
							srcText := `命令格式错误：当前没有开启状态的记录，或没有通过参数指定要取出的日志。请参考帮助。`
							d.TextMapRaw["日志"]["记录_取出_未指定记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_取出_未指定记录"] {
							d.TextMapRaw["日志"]["记录_取出_未指定记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "loglistprefix":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_列出_导入语", `正在列出存在于此群的记录:`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_列出_导入语"] {
							srcText := `正在列出存在于此群的记录:`
							d.TextMapRaw["日志"]["记录_列出_导入语"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_列出_导入语"] {
							d.TextMapRaw["日志"]["记录_列出_导入语"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logend":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_结束", `故事落下了帷幕。\n记录已经关闭。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_结束"] {
							srcText := `故事落下了帷幕。\n记录已经关闭。`
							d.TextMapRaw["日志"]["记录_结束"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_结束"] {
							d.TextMapRaw["日志"]["记录_结束"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logendsuccess":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_结束", `故事落下了帷幕。\n记录已经关闭。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_结束"] {
							srcText := `故事落下了帷幕。\n记录已经关闭。`
							d.TextMapRaw["日志"]["记录_结束"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_结束"] {
							d.TextMapRaw["日志"]["记录_结束"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "lognewbutunfinished":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_新建_失败_未结束的记录", `上一段旅程{$t记录名称}还未结束，请先使用.log end结束故事。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_新建_失败_未结束的记录"] {
							srcText := `上一段旅程{$t记录名称}还未结束，请先使用.log end结束故事。`
							d.TextMapRaw["日志"]["记录_新建_失败_未结束的记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_新建_失败_未结束的记录"] {
							d.TextMapRaw["日志"]["记录_新建_失败_未结束的记录"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "loglengthnotice":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_条数提醒", `提示: 当前故事的文本已经记录了 {$t条数} 条`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_条数提醒"] {
							srcText := `提示: 当前故事的文本已经记录了 {$t条数} 条`
							d.TextMapRaw["日志"]["记录_条数提醒"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_条数提醒"] {
							d.TextMapRaw["日志"]["记录_条数提醒"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logdelete":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_删除_成功", "删除记录 {$t记录名称} 成功")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_删除_成功"] {
							srcText := "删除记录 {$t记录名称} 成功"
							d.TextMapRaw["日志"]["记录_删除_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_删除_成功"] {
							d.TextMapRaw["日志"]["记录_删除_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logdeletesuccess":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_删除_成功", "删除记录 {$t记录名称} 成功")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_删除_成功"] {
							srcText := "删除记录 {$t记录名称} 成功"
							d.TextMapRaw["日志"]["记录_删除_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_删除_成功"] {
							d.TextMapRaw["日志"]["记录_删除_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logdeletefailnotfound":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_删除_失败_找不到", "删除记录 {$t记录名称} 失败，可能是名字不对")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_删除_失败_找不到"] {
							srcText := "删除记录 {$t记录名称} 失败，可能是名字不对"
							d.TextMapRaw["日志"]["记录_删除_失败_找不到"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_删除_失败_找不到"] {
							d.TextMapRaw["日志"]["记录_删除_失败_找不到"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logdeletefailcontinuing":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_删除_失败_正在进行", "记录 {$t记录名称} 正在进行，无法删除。请先用 log end 结束记录，如不希望上传请用 log halt。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_删除_失败_正在进行"] {
							srcText := "记录 {$t记录名称} 正在进行，无法删除。请先用 log end 结束记录，如不希望上传请用 log halt。"
							d.TextMapRaw["日志"]["记录_删除_失败_正在进行"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_删除_失败_正在进行"] {
							d.TextMapRaw["日志"]["记录_删除_失败_正在进行"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "obenter":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:OB_开启", "你将成为观众（自动修改昵称和群名片[如有权限]，并不会给观众发送暗骰结果）。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["OB_开启"] {
							srcText := "你将成为观众（自动修改昵称和群名片[如有权限]，并不会给观众发送暗骰结果）。"
							d.TextMapRaw["日志"]["OB_开启"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["OB_开启"] {
							d.TextMapRaw["日志"]["OB_开启"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "obexit":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:OB_关闭", "你不再是观众了（自动修改昵称和群名片[如有权限]）。")
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["OB_关闭"] {
							srcText := "你不再是观众了（自动修改昵称和群名片[如有权限]）。"
							d.TextMapRaw["日志"]["OB_关闭"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["OB_关闭"] {
							d.TextMapRaw["日志"]["OB_关闭"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logupload":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_上传_成功", `跑团日志已上传服务器，链接如下：{$t日志链接}`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_上传_成功"] {
							srcText := `跑团日志已上传服务器，链接如下：{$t日志链接}`
							d.TextMapRaw["日志"]["记录_上传_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_上传_成功"] {
							d.TextMapRaw["日志"]["记录_上传_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "loguploadsuccess":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_上传_成功", `跑团日志已上传服务器，链接如下：{$t日志链接}`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_上传_成功"] {
							srcText := `跑团日志已上传服务器，链接如下：{$t日志链接}`
							d.TextMapRaw["日志"]["记录_上传_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_上传_成功"] {
							d.TextMapRaw["日志"]["记录_上传_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "loguploadfail":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_上传_失败", `跑团日志上传失败：{$t错误原因}\n若未出现线上日志地址，可换时间重试，或联系骰主在data/default/log-exports路径下取出日志\n文件名: 群号_日志名_随机数.zip\n注意此文件log end/get后才会生成`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_上传_失败"] {
							srcText := `跑团日志上传失败：{$t错误原因}\n若未出现线上日志地址，可换时间重试，或联系骰主在data/default/log-exports路径下取出日志\n文件名: 群号_日志名_随机数.zip\n注意此文件log end/get后才会生成`
							d.TextMapRaw["日志"]["记录_上传_失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_上传_失败"] {
							d.TextMapRaw["日志"]["记录_上传_失败"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logexport":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_导出_成功", `日志文件《{$t文件名字}》已上传至群文件，请自行到群文件查看。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_导出_成功"] {
							srcText := `日志文件《{$t文件名字}》已上传至群文件，请自行到群文件查看。`
							d.TextMapRaw["日志"]["记录_导出_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_导出_成功"] {
							d.TextMapRaw["日志"]["记录_导出_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "logexportsuccess":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:记录_导出_成功", `日志文件《{$t文件名字}》已上传至群文件，请自行到群文件查看。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["记录_导出_成功"] {
							srcText := `日志文件《{$t文件名字}》已上传至群文件，请自行到群文件查看。`
							d.TextMapRaw["日志"]["记录_导出_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["记录_导出_成功"] {
							d.TextMapRaw["日志"]["记录_导出_成功"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "syncname":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:名片_自动设置", `已自动设置名片格式为{$t名片格式}：{$t名片预览}\n如有权限会在属性更新时自动更新名片。使用.sn off可关闭。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["名片_自动设置"] {
							srcText := `已自动设置名片格式为{$t名片格式}：{$t名片预览}\n如有权限会在属性更新时自动更新名片。使用.sn off可关闭。`
							d.TextMapRaw["日志"]["名片_自动设置"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["名片_自动设置"] {
							d.TextMapRaw["日志"]["名片_自动设置"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			case "syncnamecancel":
				if cmdNum == 1 || subval == "help" {
					text := DiceFormatReplyshow(val, ctx, "日志:名片_取消设置", `已关闭对{$t玩家}的名片自动修改。`)
					ReplyToSender(ctx, msg, text)
				} else {
					if subval == "clr" || subval == "del" || subval == "default" {
						for index := range d.TextMapRaw["日志"]["名片_取消设置"] {
							srcText := `已关闭对{$t玩家}的名片自动修改。`
							d.TextMapRaw["日志"]["名片_取消设置"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s", "已重置词条: ", val))
					} else {
						srcText := strings.ReplaceAll(cmdArgs.RawArgs, cmdArgs.GetArgN(1), "")
						srcText = strings.TrimSpace(srcText)
						for index := range d.TextMapRaw["日志"]["名片_取消设置"] {
							d.TextMapRaw["日志"]["名片_取消设置"][index][0] = srcText
						}
						SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
						d.GenerateTextMap()
						d.SaveText()
						ReplyToSender(ctx, msg, fmt.Sprintf("%s%s%s%s", "已将词条: ", val, "设为: ", srcText))
					}
				}

			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	d.CmdMap["str"] = cmdStr
}

func getDefaultDicePoints(ctx *MsgContext) int64 {
	diceSides := int64(ctx.Player.DiceSideNum)
	if diceSides == 0 && ctx.Group != nil {
		diceSides = ctx.Group.DiceSideNum
	}
	if diceSides <= 0 {
		diceSides = 100
	}
	return diceSides
}

func setRuleByName(ctx *MsgContext, name string) {
	if name == "" {
		return
	}
	diceFaces := ""
	d := ctx.Dice

	modSwitch := false
	tipText := "\n提示:"

	d.GameSystemMap.Range(func(key string, tmpl *GameSystemTemplate) bool {
		isMatch := false
		for _, k := range tmpl.SetConfig.Keys {
			if strings.EqualFold(name, k) {
				isMatch = true
				break
			}
		}

		if isMatch {
			modSwitch = true
			ctx.Group.System = key
			ctx.Group.DiceSideNum = tmpl.SetConfig.DiceSides
			ctx.Group.UpdatedAtTime = time.Now().Unix()
			tipText += tmpl.SetConfig.EnableTip

			// TODO: 命令该要进步啦
			diceFaces = strconv.FormatInt(tmpl.SetConfig.DiceSides, 10)

			for _, name := range tmpl.SetConfig.RelatedExt {
				// 开启相关扩展
				ei := ctx.Dice.ExtFind(name)
				if ei != nil {
					ctx.Group.ExtActive(ei)
				}
			}
			return false
		}
		return true
	})

	num, err := strconv.ParseInt(diceFaces, 10, 64)
	if num < 0 {
		num = 0
	}
	if err == nil {
		ctx.Group.DiceSideNum = num
		if !modSwitch {
			if num == 20 {
				tipText += "20面骰。如果要进行DND游戏，建议执行.set dnd以确保开启dnd5e指令"
			} else if num == 100 {
				tipText += "100面骰。如果要进行COC游戏，建议执行.set coc以确保开启coc7指令"
			}
		}
		if tipText == "\n提示:" {
			tipText = ""
		}
	}
}
