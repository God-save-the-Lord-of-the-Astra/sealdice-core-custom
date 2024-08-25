package dice

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func LuaVarInit(LuaVM *lua.LState, d *Dice, ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	/*//----------------------------------------------------------------
	msgTable := LuaVM.NewTable()

	// 设置Message的字段
	msgTable.RawSetString("Time", lua.LNumber(msg.Time))
	msgTable.RawSetString("MessageType", lua.LString(msg.MessageType))
	msgTable.RawSetString("GroupID", lua.LString(msg.GroupID))
	msgTable.RawSetString("GuildID", lua.LString(msg.GuildID))
	msgTable.RawSetString("ChannelID", lua.LString(msg.ChannelID))
	msgTable.RawSetString("Message", lua.LString(msg.Message))
	msgTable.RawSetString("Platform", lua.LString(msg.Platform))
	msgTable.RawSetString("GroupName", lua.LString(msg.GroupName))

	// 设置Sender的字段
	senderTable := LuaVM.NewTable()
	senderTable.RawSetString("Nickname", lua.LString(msg.Sender.Nickname))
	senderTable.RawSetString("UserID", lua.LString(msg.Sender.UserID))

	// 将senderTable添加到msgTable中
	msgTable.RawSetString("sender", senderTable)

	// Shiki散装变量兼容

	ShikiMsgFromQQ := strings.ReplaceAll(msg.Sender.UserID, "QQ:", "")
	ShikiMsgFromGroup := strings.ReplaceAll(msg.GroupID, "QQ-Group:", "")
	ShikiMsgFromUID, _ := strconv.Atoi(strings.ReplaceAll(msg.Sender.UserID, "QQ:", ""))
	ShikiMsgFromGID, _ := strconv.Atoi(strings.ReplaceAll(msg.GroupID, "QQ-Group:", ""))
	ShikiMsgFromMsg := cmdArgs.RawText
	ShikiMsgSuffix := cmdArgs.RawArgs
	ShikiMsgCmdTable := cmdArgs.Args
	msgTable.RawSetString("fromQQ", lua.LString(ShikiMsgFromQQ))
	msgTable.RawSetString("fromGroup", lua.LString(ShikiMsgFromGroup))
	msgTable.RawSetString("fromUID", lua.LNumber(ShikiMsgFromUID))
	msgTable.RawSetString("fromGID", lua.LNumber(ShikiMsgFromGID))
	msgTable.RawSetString("fromMsg", lua.LString(ShikiMsgFromMsg))
	msgTable.RawSetString("suffix", lua.LString(ShikiMsgSuffix))
	MsgCmdTable := LuaVM.NewTable()
	for _, arg := range ShikiMsgCmdTable {
		MsgCmdTable.Append(lua.LString(arg))
	}
	LuaVM.SetField(msgTable, "CmdTab", MsgCmdTable)

	//----------------------------------------------------------------
	ctxTable := LuaVM.NewTable()
	LuaVM.SetField(ctxTable, "MessageType", lua.LString(ctx.MessageType))
	LuaVM.SetField(ctxTable, "IsCurGroupBotOn", lua.LBool(ctx.IsCurGroupBotOn))
	LuaVM.SetField(ctxTable, "IsPrivate", lua.LBool(ctx.IsPrivate))
	LuaVM.SetField(ctxTable, "CommandID", lua.LNumber(ctx.CommandID))
	LuaVM.SetField(ctxTable, "PrivilegeLevel", lua.LNumber(ctx.PrivilegeLevel))
	LuaVM.SetField(ctxTable, "GroupRoleLevel", lua.LNumber(ctx.GroupRoleLevel))
	LuaVM.SetField(ctxTable, "DelegateText", lua.LString(ctx.DelegateText))
	LuaVM.SetField(ctxTable, "AliasPrefixText", lua.LString(ctx.AliasPrefixText))

	// Group info as a nested table
	if ctx.Group != nil {
		groupTable := LuaVM.NewTable()
		LuaVM.SetField(groupTable, "GroupID", lua.LString(ctx.Group.GroupID))
		LuaVM.SetField(groupTable, "GuildID", lua.LString(ctx.Group.GuildID))
		LuaVM.SetField(groupTable, "ChannelID", lua.LString(ctx.Group.ChannelID))
		LuaVM.SetField(groupTable, "GroupName", lua.LString(ctx.Group.GroupName))
		LuaVM.SetField(groupTable, "RecentDiceSendTime", lua.LNumber(ctx.Group.RecentDiceSendTime))
		LuaVM.SetField(groupTable, "ShowGroupWelcome", lua.LBool(ctx.Group.ShowGroupWelcome))
		LuaVM.SetField(groupTable, "GroupWelcomeMessage", lua.LString(ctx.Group.GroupWelcomeMessage))
		LuaVM.SetField(groupTable, "EnteredTime", lua.LNumber(ctx.Group.EnteredTime))
		LuaVM.SetField(groupTable, "InviteUserID", lua.LString(ctx.Group.InviteUserID))
		LuaVM.SetField(ctxTable, "Group", groupTable)
	}

	// Player info as a nested table
	if ctx.Player != nil {
		playerTable := LuaVM.NewTable()
		LuaVM.SetField(playerTable, "Name", lua.LString(ctx.Player.Name))
		LuaVM.SetField(playerTable, "UserID", lua.LString(ctx.Player.UserID))
		LuaVM.SetField(playerTable, "LastCommandTime", lua.LNumber(ctx.Player.LastCommandTime))
		LuaVM.SetField(playerTable, "AutoSetNameTemplate", lua.LString(ctx.Player.AutoSetNameTemplate))
		LuaVM.SetField(ctxTable, "Player", playerTable)
	}

	//----------------------------------------------------------------
	cmdArgsTable := LuaVM.NewTable()
	// 注册基本字段
	LuaVM.SetField(cmdArgsTable, "Command", lua.LString(cmdArgs.Command))

	// 注册切片类型字段 (Args)
	argsTable := LuaVM.NewTable()
	for _, arg := range cmdArgs.Args {
		argsTable.Append(lua.LString(arg))
	}
	LuaVM.SetField(cmdArgsTable, "Args", argsTable)

	// 注册结构体数组 (Kwargs)
	kwargsTable := LuaVM.NewTable()
	for _, kwarg := range cmdArgs.Kwargs {
		kwargTable := LuaVM.NewTable()
		LuaVM.SetField(kwargTable, "Name", lua.LString(kwarg.Name))
		LuaVM.SetField(kwargTable, "ValueExists", lua.LBool(kwarg.ValueExists))
		LuaVM.SetField(kwargTable, "Value", lua.LString(kwarg.Value))
		LuaVM.SetField(kwargTable, "AsBool", lua.LBool(kwarg.AsBool))
		kwargsTable.Append(kwargTable)
	}
	LuaVM.SetField(cmdArgsTable, "Kwargs", kwargsTable)

	// 注册结构体数组 (At)
	atInfoTable := LuaVM.NewTable()
	for _, at := range cmdArgs.At {
		atTable := LuaVM.NewTable()
		LuaVM.SetField(atTable, "UserID", lua.LString(at.UserID))
		atInfoTable.Append(atTable)
	}
	LuaVM.SetField(cmdArgsTable, "At", atInfoTable)

	// 注册其他基本字段
	LuaVM.SetField(cmdArgsTable, "RawArgs", lua.LString(cmdArgs.RawArgs))
	LuaVM.SetField(cmdArgsTable, "AmIBeMentioned", lua.LBool(cmdArgs.AmIBeMentioned))
	LuaVM.SetField(cmdArgsTable, "AmIBeMentionedFirst", lua.LBool(cmdArgs.AmIBeMentionedFirst))
	LuaVM.SetField(cmdArgsTable, "SomeoneBeMentionedButNotMe", lua.LBool(cmdArgs.SomeoneBeMentionedButNotMe))
	LuaVM.SetField(cmdArgsTable, "IsSpaceBeforeArgs", lua.LBool(cmdArgs.IsSpaceBeforeArgs))
	LuaVM.SetField(cmdArgsTable, "CleanArgs", lua.LString(cmdArgs.CleanArgs))
	LuaVM.SetField(cmdArgsTable, "SpecialExecuteTimes", lua.LNumber(cmdArgs.SpecialExecuteTimes))
	LuaVM.SetField(cmdArgsTable, "RawText", lua.LString(cmdArgs.RawText))

	//----------------------------------------------------------------
	LuaVM.SetGlobal("cmdArgs", cmdArgsTable)
	// Set the table to the global variable "ctx"
	LuaVM.SetGlobal("ctx", ctxTable)
	// 将msgTable注册为全局变量"msg"
	LuaVM.SetGlobal("msg", msgTable)*/
	//----------------------------------------------------------------
	msgUD := LuaVM.NewUserData()
	msgUD.Value = msg
	msgMeta := LuaVM.NewTypeMetatable("Message")
	msgUD.Metatable = msgMeta
	LuaVM.SetGlobal("Message", msgMeta)
	LuaVM.SetField(msgMeta, "__index", LuaVM.SetFuncs(LuaVM.NewTable(), map[string]lua.LGFunction{
		"Time": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LNumber(msg.Time))
			return 1
		},
		"MessageType": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LString(msg.MessageType))
			return 1
		},
		"GroupID": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LString(msg.GroupID))
			return 1
		},
		"GuildID": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LString(msg.GuildID))
			return 1
		},
		"ChannelID": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LString(msg.ChannelID))
			return 1
		},
		"Message": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LString(msg.Message))
			return 1
		},
		"Platform": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LString(msg.Platform))
			return 1
		},
		"GroupName": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			LuaVM.Push(lua.LString(msg.GroupName))
			return 1
		},
		"Sender": func(LuaVM *lua.LState) int {
			msg := LuaVM.CheckUserData(1).Value.(*Message)
			senderTable := LuaVM.NewTable()
			senderTable.RawSetString("Nickname", lua.LString(msg.Sender.Nickname))
			senderTable.RawSetString("UserID", lua.LString(msg.Sender.UserID))
			LuaVM.Push(senderTable)
			return 1
		},
	}))

	LuaVM.SetGlobal("msg", msgUD)

	//----------------------------------------------------------------
	ctxUD := LuaVM.NewUserData()
	ctxUD.Value = ctx
	ctxMeta := LuaVM.NewTypeMetatable("MsgContext")
	ctxUD.Metatable = ctxMeta
	LuaVM.SetGlobal("MsgContext", ctxMeta)
	LuaVM.SetField(ctxMeta, "__index", LuaVM.SetFuncs(LuaVM.NewTable(), map[string]lua.LGFunction{
		"MessageType": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LString(ctx.MessageType))
			return 1
		},
		"IsCurGroupBotOn": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LBool(ctx.IsCurGroupBotOn))
			return 1
		},
		"IsPrivate": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LBool(ctx.IsPrivate))
			return 1
		},
		"CommandID": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LNumber(ctx.CommandID))
			return 1
		},
		"PrivilegeLevel": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LNumber(ctx.PrivilegeLevel))
			return 1
		},
		"GroupRoleLevel": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LNumber(ctx.GroupRoleLevel))
			return 1
		},
		"DelegateText": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LString(ctx.DelegateText))
			return 1
		},
		"AliasPrefixText": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			LuaVM.Push(lua.LString(ctx.AliasPrefixText))
			return 1
		},
		"Group": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			if ctx.Group != nil {
				groupTable := LuaVM.NewTable()
				LuaVM.SetField(groupTable, "GroupID", lua.LString(ctx.Group.GroupID))
				LuaVM.SetField(groupTable, "GuildID", lua.LString(ctx.Group.GuildID))
				LuaVM.SetField(groupTable, "ChannelID", lua.LString(ctx.Group.ChannelID))
				LuaVM.SetField(groupTable, "GroupName", lua.LString(ctx.Group.GroupName))
				LuaVM.SetField(groupTable, "RecentDiceSendTime", lua.LNumber(ctx.Group.RecentDiceSendTime))
				LuaVM.SetField(groupTable, "ShowGroupWelcome", lua.LBool(ctx.Group.ShowGroupWelcome))
				LuaVM.SetField(groupTable, "GroupWelcomeMessage", lua.LString(ctx.Group.GroupWelcomeMessage))
				LuaVM.SetField(groupTable, "EnteredTime", lua.LNumber(ctx.Group.EnteredTime))
				LuaVM.SetField(groupTable, "InviteUserID", lua.LString(ctx.Group.InviteUserID))
				LuaVM.Push(groupTable)
			} else {
				LuaVM.Push(lua.LNil)
			}
			return 1
		},
		"Player": func(LuaVM *lua.LState) int {
			ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
			if ctx.Player != nil {
				playerTable := LuaVM.NewTable()
				LuaVM.SetField(playerTable, "Name", lua.LString(ctx.Player.Name))
				LuaVM.SetField(playerTable, "UserID", lua.LString(ctx.Player.UserID))
				LuaVM.SetField(playerTable, "LastCommandTime", lua.LNumber(ctx.Player.LastCommandTime))
				LuaVM.SetField(playerTable, "AutoSetNameTemplate", lua.LString(ctx.Player.AutoSetNameTemplate))
				LuaVM.Push(playerTable)
			} else {
				LuaVM.Push(lua.LNil)
			}
			return 1
		},
	}))

	LuaVM.SetGlobal("ctx", ctxUD)

	//----------------------------------------------------------------
	cmdArgsUD := LuaVM.NewUserData()
	cmdArgsUD.Value = cmdArgs
	cmdArgsMeta := LuaVM.NewTypeMetatable("CmdArgs")
	cmdArgsUD.Metatable = cmdArgsMeta
	LuaVM.SetGlobal("CmdArgs", cmdArgsMeta)
	LuaVM.SetField(cmdArgsMeta, "__index", LuaVM.SetFuncs(LuaVM.NewTable(), map[string]lua.LGFunction{
		"Command": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LString(cmdArgs.Command))
			return 1
		},
		"Args": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			argsTable := LuaVM.NewTable()
			for _, arg := range cmdArgs.Args {
				argsTable.Append(lua.LString(arg))
			}
			LuaVM.Push(argsTable)
			return 1
		},
		"Kwargs": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			kwargsTable := LuaVM.NewTable()
			for _, kwarg := range cmdArgs.Kwargs {
				kwargTable := LuaVM.NewTable()
				LuaVM.SetField(kwargTable, "Name", lua.LString(kwarg.Name))
				LuaVM.SetField(kwargTable, "ValueExists", lua.LBool(kwarg.ValueExists))
				LuaVM.SetField(kwargTable, "Value", lua.LString(kwarg.Value))
				LuaVM.SetField(kwargTable, "AsBool", lua.LBool(kwarg.AsBool))
				kwargsTable.Append(kwargTable)
			}
			LuaVM.Push(kwargsTable)
			return 1
		},
		"At": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			atInfoTable := LuaVM.NewTable()
			for _, at := range cmdArgs.At {
				atTable := LuaVM.NewTable()
				LuaVM.SetField(atTable, "UserID", lua.LString(at.UserID))
				atInfoTable.Append(atTable)
			}
			LuaVM.Push(atInfoTable)
			return 1
		},
		"RawArgs": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LString(cmdArgs.RawArgs))
			return 1
		},
		"AmIBeMentioned": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LBool(cmdArgs.AmIBeMentioned))
			return 1
		},
		"AmIBeMentionedFirst": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LBool(cmdArgs.AmIBeMentionedFirst))
			return 1
		},
		"SomeoneBeMentionedButNotMe": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LBool(cmdArgs.SomeoneBeMentionedButNotMe))
			return 1
		},
		"IsSpaceBeforeArgs": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LBool(cmdArgs.IsSpaceBeforeArgs))
			return 1
		},
		"CleanArgs": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LString(cmdArgs.CleanArgs))
			return 1
		},
		"SpecialExecuteTimes": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LNumber(cmdArgs.SpecialExecuteTimes))
			return 1
		},
		"RawText": func(LuaVM *lua.LState) int {
			cmdArgs := LuaVM.CheckUserData(1).Value.(*CmdArgs)
			LuaVM.Push(lua.LString(cmdArgs.RawText))
			return 1
		},
	}))

	LuaVM.SetGlobal("cmdArgs", cmdArgsUD)

	DiceUD := LuaVM.NewUserData()
	DiceUD.Value = d
	DiceMeta := LuaVM.NewTypeMetatable("Dice")
	LuaVM.SetGlobal("Dice", DiceMeta)
	LuaVM.SetGlobal("d", DiceUD)

	//----------------------------------------------------------------
	// Shiki散装变量兼容
	ShikiMsgTable := LuaVM.NewTable()
	ShikiMsgFromQQ := strings.ReplaceAll(msg.Sender.UserID, "QQ:", "")
	ShikiMsgFromGroup := strings.ReplaceAll(msg.GroupID, "QQ-Group:", "")
	ShikiMsgFromUID, _ := strconv.Atoi(strings.ReplaceAll(msg.Sender.UserID, "QQ:", ""))
	ShikiMsgFromGID, _ := strconv.Atoi(strings.ReplaceAll(msg.GroupID, "QQ-Group:", ""))
	ShikiMsgFromMsg := cmdArgs.RawText
	ShikiMsgSuffix := cmdArgs.RawArgs
	ShikiMsgCmdTable := cmdArgs.Args

	// Register Shiki variables
	LuaVM.SetField(ShikiMsgTable, "fromQQ", lua.LString(ShikiMsgFromQQ))
	LuaVM.SetField(ShikiMsgTable, "fromGroup", lua.LString(ShikiMsgFromGroup))
	LuaVM.SetField(ShikiMsgTable, "fromUID", lua.LNumber(ShikiMsgFromUID))
	LuaVM.SetField(ShikiMsgTable, "fromGID", lua.LNumber(ShikiMsgFromGID))
	LuaVM.SetField(ShikiMsgTable, "fromMsg", lua.LString(ShikiMsgFromMsg))
	LuaVM.SetField(ShikiMsgTable, "suffix", lua.LString(ShikiMsgSuffix))
	MsgCmdTable := LuaVM.NewTable()
	for _, arg := range ShikiMsgCmdTable {
		MsgCmdTable.Append(lua.LString(arg))
	}
	LuaVM.SetField(ShikiMsgTable, "CmdTab", MsgCmdTable)
	LuaVM.SetGlobal("shikimsg", ShikiMsgTable)
	//----------------------------------------------------------------
	// Dream 散装变量兼容
	DreamMsgGroup_ID := strings.ReplaceAll(msg.GroupID, "QQ-Group:", "")
	DreamMsgSender_ID := strings.ReplaceAll(msg.Sender.UserID, "QQ:", "")
	DreamMsgGroup_Nick := ctx.Group.GroupName
	DreamMsgSender_Nick := ctx.Player.Name
	DreamMsgMessage_Text := cmdArgs.RawText
	DreamMsgSender_Jrrp, _ := VarGetValueInt64(ctx, "$人品")
	DreamMsgTable := LuaVM.NewTable()
	DreamGroupTable := LuaVM.NewTable()
	DreamMessageTable := LuaVM.NewTable()
	DreamSenderTable := LuaVM.NewTable()
	LuaVM.SetField(DreamGroupTable, "id", lua.LString(DreamMsgGroup_ID))
	LuaVM.SetField(DreamGroupTable, "nick", lua.LString(DreamMsgGroup_Nick))
	LuaVM.SetField(DreamMsgTable, "group", DreamGroupTable)
	LuaVM.SetField(DreamMessageTable, "txt", lua.LString(DreamMsgMessage_Text))
	LuaVM.SetField(DreamMsgTable, "message", DreamMessageTable)
	LuaVM.SetField(DreamSenderTable, "id", lua.LString(DreamMsgSender_ID))
	LuaVM.SetField(DreamSenderTable, "nick", lua.LString(DreamMsgSender_Nick))
	LuaVM.SetField(DreamSenderTable, "jrrp", lua.LNumber(DreamMsgSender_Jrrp))
	LuaVM.SetField(DreamMsgTable, "sender", DreamSenderTable)
	LuaVM.SetGlobal("dreammsg", DreamMsgTable)
}

// ----------------------------------------------------------------
func luaVarSetValueStr(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	s := LuaVM.CheckString(2)
	v := LuaVM.CheckString(3)
	VarSetValueStr(ctx, s, v)
	return 0 // 返回 0 表示无返回值
}

func luaVarSetValueInt(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	s := LuaVM.CheckString(2)
	v := LuaVM.CheckInt64(3)
	VarSetValueInt64(ctx, s, v)
	return 0 // 返回 0 表示无返回值
}

func luaVarDelValue(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	s := LuaVM.CheckString(2)
	VarDelValue(ctx, s)
	return 0 // 返回 0 表示无返回值
}

func luaVarGetValueInt(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	s := LuaVM.CheckString(1)
	res, exists := VarGetValueInt64(ctx, s)
	if !exists {
		return 0 // 返回 0 表示没有值
	}
	LuaVM.Push(lua.LNumber(res)) // 推送结果到 Lua 栈
	return 1                     // 返回 1 表示成功
}

func luaVarGetValueStr(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	s := LuaVM.CheckString(2)
	res, exists := VarGetValueStr(ctx, s)
	if !exists {
		return 0 // 返回 0 表示没有值
	}
	LuaVM.Push(lua.LString(res)) // 推送结果到 Lua 栈
	return 1                     // 返回 1 表示成功
}

func luaAddBan(LuaVM *lua.LState) int {
	id := LuaVM.CheckString(1)
	d := LuaVM.CheckUserData(2).Value.(*Dice)
	place := LuaVM.CheckString(3)
	reason := LuaVM.CheckString(4)
	ctx := LuaVM.CheckUserData(5).Value.(*MsgContext)
	d.BanList.AddScoreBase(id, d.BanList.ThresholdBan, place, reason, ctx)
	d.BanList.SaveChanged(d)
	return 1 // 返回 1 表示成功
}

func luaAddTrust(LuaVM *lua.LState) int {
	d := LuaVM.CheckUserData(1).Value.(*Dice)
	id := LuaVM.CheckString(2)
	place := LuaVM.CheckString(3)
	reason := LuaVM.CheckString(4)
	d.BanList.SetTrustByID(id, place, reason)
	d.BanList.SaveChanged(d)
	return 1 // 返回 1 表示成功
}

func luaRemoveBan(LuaVM *lua.LState) int {
	d := LuaVM.CheckUserData(1).Value.(*Dice)
	id := LuaVM.CheckString(2)
	_, ok := d.BanList.GetByID(id)
	if !ok {
		return 0 // 返回 0 表示没有值
	}
	d.BanList.DeleteByID(d, id)
	return 1 // 返回 1 表示成功
}

func luaReplyGroup(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	msg := LuaVM.CheckUserData(2).Value.(*Message)
	text := LuaVM.CheckString(3)
	ReplyGroup(ctx, msg, text)
	return 1 // 返回 1 表示成功
}

func luaReplyPerson(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	msg := LuaVM.CheckUserData(2).Value.(*Message)
	text := LuaVM.CheckString(3)
	ReplyPerson(ctx, msg, text)
	return 1 // 返回 1 表示成功
}

func luaReplyToSender(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	msg := LuaVM.CheckUserData(2).Value.(*Message)
	text := LuaVM.CheckString(3)
	ReplyToSender(ctx, msg, text)
	return 1 // 返回 1 表示成功
}
func luaMemberBan(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	groupID := LuaVM.CheckString(2)
	userID := LuaVM.CheckString(3)
	duration := LuaVM.CheckInt64(4)
	MemberBan(ctx, groupID, userID, duration)
	return 1 // 返回 1 表示成功
}
func luaMemberKick(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	groupID := LuaVM.CheckString(2)
	userID := LuaVM.CheckString(3)
	MemberKick(ctx, groupID, userID)
	return 1 // 返回 1 表示成功
}
func luaDiceFormat(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	s := LuaVM.CheckString(2)
	res := DiceFormat(ctx, s)
	LuaVM.Push(lua.LString(res))
	return 1 // 返回 1 表示成功
}

func luaDiceFormatTmpl(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	s := LuaVM.CheckString(2)
	res := DiceFormatTmpl(ctx, s)
	LuaVM.Push(lua.LString(res))
	return 1 // 返回 1 表示成功
}

func luaShikiSendMsg(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	msg := LuaVM.CheckUserData(2).Value.(*Message)
	text := LuaVM.CheckString(3)
	msg_fromGroup := LuaVM.CheckString(4)
	msg_fromQQ := LuaVM.CheckString(5)
	if msg_fromQQ == "" {
		msg_fromQQ = ctx.Player.UserID
	}
	if msg_fromGroup != "" && strings.HasPrefix(msg_fromGroup, "QQ-Group:") == false {
		msg_fromGroup = fmt.Sprintf("%s%s", "QQ-Group:", msg_fromGroup)
	}
	if strings.HasPrefix(msg_fromQQ, "QQ:") == false {
		msg_fromQQ = fmt.Sprintf("%s%s", "QQ:", msg_fromQQ)
	}
	msg.Sender.UserID = msg_fromQQ
	msg.GroupID = msg_fromGroup
	if msg_fromGroup == "" {
		ctx.IsPrivate = true
		msg.MessageType = "private"
		msg.Time = int64(time.Now().Unix())
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		ReplyPerson(ctx, msg, text)
	} else {
		msg.Time = int64(time.Now().Unix())
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		ReplyGroup(ctx, msg, text)
	}
	return 1 // 返回 1 表示成功
}

// ----------------------------------------------------------------
func luaDreamJSONEncode(LuaVM *lua.LState) int {
	lv := LuaVM.CheckTable(1)
	goMap := toGoMap(lv)
	jsonData, err := json.Marshal(goMap)
	if err != nil {
		LuaVM.Push(lua.LNil)
		LuaVM.Push(lua.LString(err.Error()))
		return 2
	}

	LuaVM.Push(lua.LString(jsonData))
	return 1 // 返回 1 表示成功
}

// luaJSONDecode decodes a JSON string into a Lua table.
func luaDreamJSONDecode(LuaVM *lua.LState) int {
	jsonStr := LuaVM.CheckString(1)

	var goMap map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &goMap)
	if err != nil {
		LuaVM.Push(lua.LNil)
		LuaVM.Push(lua.LString(err.Error()))
		return 2
	}

	luaTable := toLuaTable(LuaVM, goMap)
	LuaVM.Push(luaTable)
	return 1 // 返回 1 表示成功
}

// toGoMap converts a Lua table to a Go map.
func toGoMap(lv *lua.LTable) map[string]interface{} {
	goMap := make(map[string]interface{})
	lv.ForEach(func(key lua.LValue, value lua.LValue) {
		goMap[key.String()] = toGoValue(value)
	})
	return goMap
}

// toGoValue converts a Lua value to a Go value.
func toGoValue(lv lua.LValue) interface{} {
	switch lv.Type() {
	case lua.LTString:
		return lv.String()
	case lua.LTNumber:
		return float64(lua.LVAsNumber(lv))
	case lua.LTBool:
		return lua.LVAsBool(lv)
	case lua.LTTable:
		return toGoMap(lv.(*lua.LTable))
	default:
		return nil
	}
}

// toLuaTable converts a Go map to a Lua table.
func toLuaTable(LuaVM *lua.LState, goMap map[string]interface{}) *lua.LTable {
	luaTable := LuaVM.NewTable()
	for key, value := range goMap {
		luaTable.RawSetString(key, toLuaValue(LuaVM, value))
	}
	return luaTable
}

// toLuaValue converts a Go value to a Lua value.
func toLuaValue(LuaVM *lua.LState, value interface{}) lua.LValue {
	switch v := value.(type) {
	case string:
		return lua.LString(v)
	case float64:
		return lua.LNumber(v)
	case bool:
		return lua.LBool(v)
	case map[string]interface{}:
		return toLuaTable(LuaVM, v)
	default:
		return lua.LNil
	}
}

// String sub function
func luaDreamStringSub(LuaVM *lua.LState) int {
	str := LuaVM.CheckString(1)
	start := LuaVM.CheckInt(2)
	end := LuaVM.CheckInt(3)
	LuaVM.Push(lua.LString(string([]rune(str)[start-1 : end])))
	return 1
}

// String part function
func luaDreamStringPart(LuaVM *lua.LState) int {
	str := LuaVM.CheckString(1)
	sep := LuaVM.CheckString(2)
	parts := strings.Split(str, sep)
	table := LuaVM.NewTable()
	for i, part := range parts {
		table.RawSetInt(i+1, lua.LString(part))
	}
	LuaVM.Push(table)
	return 1
}

// String find function
func luaDreamStringFind(LuaVM *lua.LState) int {
	str := LuaVM.CheckString(1)
	substr := LuaVM.CheckString(2)
	count := strings.Count(str, substr)
	LuaVM.Push(lua.LNumber(count))
	return 1
}

// String toTable function
func luaDreamStringToTable(LuaVM *lua.LState) int {
	str := LuaVM.CheckString(1)
	runes := []rune(str)
	table := LuaVM.NewTable()
	for i, r := range runes {
		table.RawSetInt(i+1, lua.LString(string(r)))
	}
	LuaVM.Push(table)
	return 1
}

// String len function
func luaDreamStringLen(LuaVM *lua.LState) int {
	str := LuaVM.CheckString(1)
	length := len([]rune(str))
	LuaVM.Push(lua.LNumber(length))
	return 1
}

// String format function
func luaDreamStringFormat(LuaVM *lua.LState) int {
	str := LuaVM.CheckString(1)
	tab := LuaVM.CheckTable(2)
	vars := make(map[string]string)
	tab.ForEach(func(key, value lua.LValue) {
		vars[key.String()] = value.String()
	})
	for k, v := range vars {
		str = strings.ReplaceAll(str, "{"+k+"}", v)
	}
	LuaVM.Push(lua.LString(str))
	return 1
}

func luaDreamTableType(LuaVM *lua.LState) int {
	table := LuaVM.CheckTable(1)
	isArray := true
	isObject := true

	table.ForEach(func(key, value lua.LValue) {
		if key.Type() == lua.LTNumber {
			isObject = false
		} else if key.Type() == lua.LTString {
			isArray = false
		}
	})

	if isArray {
		LuaVM.Push(lua.LString("array"))
	} else if isObject {
		LuaVM.Push(lua.LString("object"))
	} else {
		LuaVM.Push(lua.LNil)
	}
	return 1
}

// Function to make a table orderly
func luaDreamTableOrderly(LuaVM *lua.LState) int {
	table := LuaVM.CheckTable(1)
	newTable := LuaVM.NewTable()

	table.ForEach(func(key, value lua.LValue) {
		if key.Type() == lua.LTNumber {
			newTable.Append(value)
		}
	})
	LuaVM.Push(newTable)
	return 1
}
func luaDreamtableToString(LuaVM *lua.LState) int {
	table := LuaVM.CheckTable(1)
	result, err := tableToStringInDreamTable(table, "", make(map[*lua.LTable]bool))
	if err != nil {
		LuaVM.RaiseError(err.Error())
		return 0
	}
	LuaVM.Push(lua.LString(result))
	return 1
}

func tableToStringInDreamTable(table *lua.LTable, indent string, visited map[*lua.LTable]bool) (string, error) {
	if visited[table] {
		return "", fmt.Errorf("circular references")
	}
	visited[table] = true

	var sb strings.Builder
	sb.WriteString("{")
	newIndent := indent + "  "

	table.ForEach(func(key lua.LValue, value lua.LValue) {
		keyStr := luaValueToStringInDreamTable(key)
		valueStr := ""
		if value.Type() == lua.LTTable {
			if visited[value.(*lua.LTable)] {
				valueStr = "circular reference"
			} else {
				var err error
				valueStr, err = tableToStringInDreamTable(value.(*lua.LTable), newIndent, visited)
				if err != nil {
					valueStr = "error"
				}
			}
		} else {
			valueStr = luaValueToStringInDreamTable(value)
		}
		sb.WriteString(fmt.Sprintf("\n%s[%s] -> %s", newIndent, keyStr, valueStr))
	})

	if sb.String() != "{" {
		sb.WriteString(fmt.Sprintf("\n%s}", indent))
	} else {
		sb.WriteString("}")
	}

	return sb.String(), nil
}

func luaValueToStringInDreamTable(value lua.LValue) string {
	switch value.Type() {
	case lua.LTNil:
		return "nil"
	case lua.LTBool:
		return fmt.Sprintf("%t", lua.LVAsBool(value))
	case lua.LTNumber:
		return fmt.Sprintf("%v", lua.LVAsNumber(value))
	case lua.LTString:
		return fmt.Sprintf("%q", lua.LVAsString(value))
	case lua.LTFunction:
		return "function"
	case lua.LTTable:
		return "table"
	case lua.LTUserData:
		return "userdata"
	default:
		return "unknown"
	}
}

// Function to get the length of a table
func luaDreamTableGetNumber(LuaVM *lua.LState) int {
	table := LuaVM.CheckTable(1)
	length := 0

	table.ForEach(func(key, value lua.LValue) {
		length++
	})
	LuaVM.Push(lua.LNumber(length))
	return 1
}

// Function to sort a table
func luaDreamTableSort(LuaVM *lua.LState) int {
	table := LuaVM.CheckTable(1)
	key := LuaVM.OptString(2, "")

	var values []lua.LValue
	table.ForEach(func(_, value lua.LValue) {
		values = append(values, value)
	})

	if key == "" {
		sort.Slice(values, func(i, j int) bool {
			return values[i].(lua.LNumber) > values[j].(lua.LNumber)
		})
	} else {
		sort.Slice(values, func(i, j int) bool {
			return values[i].(*lua.LTable).RawGetString(key).(lua.LNumber) > values[j].(*lua.LTable).RawGetString(key).(lua.LNumber)
		})
	}

	newTable := LuaVM.NewTable()
	for _, value := range values {
		newTable.Append(value)
	}
	LuaVM.Push(newTable)
	return 1
}

// Function to clone a table
func luaDreamTableClone(LuaVM *lua.LState) int {
	table := LuaVM.CheckTable(1)
	newTable := LuaVM.NewTable()

	table.ForEach(func(key, value lua.LValue) {
		newTable.RawSet(key, value)
	})
	LuaVM.Push(newTable)
	return 1
}

// Function to check if two tables are equal
func luaDreamTableEqual(LuaVM *lua.LState) int {
	table1 := LuaVM.CheckTable(1)
	table2 := LuaVM.CheckTable(2)

	if reflect.DeepEqual(table1, table2) {
		LuaVM.Push(lua.LTrue)
	} else {
		LuaVM.Push(lua.LFalse)
	}
	return 1
}

// Function to replace substrings in an array of strings
func luaDreamTableGsub(LuaVM *lua.LState) int {
	table := LuaVM.CheckTable(1)
	old := LuaVM.CheckString(2)
	new := LuaVM.CheckString(3)

	newTable := LuaVM.NewTable()
	table.ForEach(func(_, value lua.LValue) {
		str := strings.Replace(value.String(), old, new, -1)
		newTable.Append(lua.LString(str))
	})
	LuaVM.Push(newTable)
	return 1
}

func luaDreamTableAdd(LuaVM *lua.LState) int {
	// 创建一个新的表用于存储合并结果
	newTable := LuaVM.NewTable()

	// 获取传入参数的数量
	numArgs := LuaVM.GetTop()

	// 遍历所有传入的参数
	for i := 1; i <= numArgs; i++ {
		// 检查参数是否为表
		table := LuaVM.CheckTable(i)

		// 将当前表的所有键值对添加到新表中
		table.ForEach(func(key, value lua.LValue) {
			newTable.RawSet(key, value)
		})
	}

	// 将合并后的新表压入 Lua 栈
	LuaVM.Push(newTable)

	// 返回结果表的数量
	return 1
}

// Base64 encode function
func luaDreamBase64Encode(LuaVM *lua.LState) int {
	input := LuaVM.CheckString(1)
	encoded := base64.StdEncoding.EncodeToString([]byte(input))
	LuaVM.Push(lua.LString(encoded))
	return 1
}

// Base64 decode function
func luaDreamBase64Decode(LuaVM *lua.LState) int {
	input := LuaVM.CheckString(1)
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		LuaVM.Push(lua.LNil)
		LuaVM.Push(lua.LString(err.Error()))
		return 2
	}
	LuaVM.Push(lua.LString(decoded))
	return 1
}

// MD5 hash function
func luaDreamMd5Hash(LuaVM *lua.LState) int {
	input := LuaVM.CheckString(1)
	hash := md5.New()
	io.WriteString(hash, input)
	hashed := fmt.Sprintf("%x", hash.Sum(nil))
	LuaVM.Push(lua.LString(hashed))
	return 1
}

// SHA256 hash function
func luaDreamSha256Hash(LuaVM *lua.LState) int {
	input := LuaVM.CheckString(1)
	hash := sha256.New()
	io.WriteString(hash, input)
	hashed := fmt.Sprintf("%x", hash.Sum(nil))
	LuaVM.Push(lua.LString(hashed))
	return 1
}

// BKDRHash function
func luaDreamBKDRHash(LuaVM *lua.LState) int {
	input := LuaVM.CheckString(1)
	seed := LuaVM.CheckInt(2)
	hash := BKDRHashInDreamBKDR(input, seed)
	LuaVM.Push(lua.LString(hash))
	return 1
}

// BKDRHash算法
func BKDRHashInDreamBKDR(s string, seed int) string {
	const seed_a = 131  // 31 131 1313 13131 131313 etc.
	const seed_b = 1313 // 131 1313 13131 131313 etc.
	hash := 0
	for _, c := range s {
		hash = (hash*seed_a + int(c)) % seed_b
	}
	return fmt.Sprintf("%d", hash)
}

//----------------------------------------------------------------

func luaZhaoDiceSDKSystemReload(LuaVM *lua.LState) int {
	ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
	var dm = ctx.Dice.Parent
	dm.RebootRequestChan <- 1
	return 1
}

func luaZhaoDiceSDKTrim(L *lua.LState) int {
	// 获取第一个参数，并确保它是一个字符串
	str := L.CheckString(1)
	// 去除字符串两端的空白字符
	trimmedStr := strings.TrimSpace(str)
	// 将结果压入 Lua 栈
	L.Push(lua.LString(trimmedStr))
	// 返回结果的数量
	return 1
}

func luaZhaoDiceSDKContains(L *lua.LState) int {
	str1 := L.CheckString(1)
	str2 := L.CheckString(2)
	flg := strings.Contains(str1, str2)
	L.Push(lua.LBool(flg))
	// 返回结果的数量
	return 1
}

//----------------------------------------------------------------

/*
	func luaShikiEventMsg(LuaVM *lua.LState) int {
		ctx := LuaVM.CheckUserData(1).Value.(*MsgContext)
		msg := LuaVM.CheckUserData(2).Value.(*Message)
		cmdArgs := LuaVM.CheckUserData(3).Value.(*CmdArgs)
		text := LuaVM.CheckString(4)
		msg_fromGroup := LuaVM.CheckString(5)
		msg_fromQQ := LuaVM.CheckString(6)
		if msg_fromGroup != "" && strings.HasPrefix(msg_fromGroup, "QQ-Group:") == false {
			msg_fromGroup = fmt.Sprintf("%s%s", "QQ-Group:", msg_fromGroup)
		}
		if strings.HasPrefix(msg_fromQQ, "QQ:") == false {
			msg_fromQQ = fmt.Sprintf("%s%s", "QQ:", msg_fromQQ)
		}
		msg.Sender.UserID = msg_fromQQ
		msg.GroupID = msg_fromGroup
		if msg_fromGroup == "" {
			ctx.IsPrivate = true
			msg.MessageType = "private"
			msg.Time = int64(time.Now().Unix())
			ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		} else {
			msg.Time = int64(time.Now().Unix())
			ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		}

		return 1 // 返回 1 表示成功
	}
*/

func LuaFuncInit(LuaVM *lua.LState, ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	LuaVM.SetGlobal("VarSetValueStr", LuaVM.NewFunction(luaVarSetValueStr))
	LuaVM.SetGlobal("VarSetValueInt", LuaVM.NewFunction(luaVarSetValueInt))
	LuaVM.SetGlobal("VarDelValue", LuaVM.NewFunction(luaVarDelValue))
	LuaVM.SetGlobal("VarGetValueInt", LuaVM.NewFunction(luaVarGetValueInt))
	LuaVM.SetGlobal("VarGetValueStr", LuaVM.NewFunction(luaVarGetValueStr))
	LuaVM.SetGlobal("AddBan", LuaVM.NewFunction(luaAddBan))
	LuaVM.SetGlobal("AddTrust", LuaVM.NewFunction(luaAddTrust))
	LuaVM.SetGlobal("RemoveBan", LuaVM.NewFunction(luaRemoveBan))
	LuaVM.SetGlobal("ReplyGroup", LuaVM.NewFunction(luaReplyGroup))
	LuaVM.SetGlobal("ReplyPerson", LuaVM.NewFunction(luaReplyPerson))
	LuaVM.SetGlobal("ReplyToSender", LuaVM.NewFunction(luaReplyToSender))
	LuaVM.SetGlobal("MemberBan", LuaVM.NewFunction(luaMemberBan))
	LuaVM.SetGlobal("MemberKick", LuaVM.NewFunction(luaMemberKick))
	LuaVM.SetGlobal("DiceFormat", LuaVM.NewFunction(luaDiceFormat))
	LuaVM.SetGlobal("DiceFormatTmpl", LuaVM.NewFunction(luaDiceFormatTmpl))
	LuaVM.SetGlobal("shikisendMsg", LuaVM.NewFunction(luaShikiSendMsg))
	//----------------------------------------------------------------
	DreamLib := LuaVM.NewTable()
	DreamLib.RawSetString("_VERSION", lua.LString("ver4.9.6(206)"))
	DreamLib.RawSetString("version", lua.LString("Dream by 筑梦师V2.0&乐某人 for Tempest Dice"))
	DreamJson := LuaVM.NewTable()
	DreamString := LuaVM.NewTable()
	DreamTable := LuaVM.NewTable()
	DreamBase64 := LuaVM.NewTable()
	DreamMd5 := LuaVM.NewTable()
	DreamSha256 := LuaVM.NewTable()
	DreamBKDR := LuaVM.NewTable()
	LuaVM.SetField(DreamLib, "json", DreamJson)
	LuaVM.SetField(DreamJson, "encode", LuaVM.NewFunction(luaDreamJSONEncode))
	LuaVM.SetField(DreamJson, "decode", LuaVM.NewFunction(luaDreamJSONDecode))
	LuaVM.SetField(DreamLib, "string", DreamString)
	LuaVM.SetField(DreamString, "sub", LuaVM.NewFunction(luaDreamStringSub))
	LuaVM.SetField(DreamString, "part", LuaVM.NewFunction(luaDreamStringPart))
	LuaVM.SetField(DreamString, "find", LuaVM.NewFunction(luaDreamStringFind))
	LuaVM.SetField(DreamString, "totable", LuaVM.NewFunction(luaDreamStringToTable))
	LuaVM.SetField(DreamString, "len", LuaVM.NewFunction(luaDreamStringLen))
	LuaVM.SetField(DreamString, "format", LuaVM.NewFunction(luaDreamStringFormat))
	LuaVM.SetField(DreamLib, "table", DreamTable)
	LuaVM.SetField(DreamTable, "type", LuaVM.NewFunction(luaDreamTableType))
	LuaVM.SetField(DreamTable, "orderly", LuaVM.NewFunction(luaDreamTableOrderly))
	LuaVM.SetField(DreamTable, "getnumber", LuaVM.NewFunction(luaDreamTableGetNumber))
	LuaVM.SetField(DreamTable, "sort", LuaVM.NewFunction(luaDreamTableSort))
	LuaVM.SetField(DreamTable, "clone", LuaVM.NewFunction(luaDreamTableClone))
	LuaVM.SetField(DreamTable, "equal", LuaVM.NewFunction(luaDreamTableEqual))
	LuaVM.SetField(DreamTable, "gsub", LuaVM.NewFunction(luaDreamTableGsub))
	LuaVM.SetField(DreamTable, "add", LuaVM.NewFunction(luaDreamTableAdd))
	LuaVM.SetField(DreamTable, "tostring", LuaVM.NewFunction(luaDreamtableToString))
	LuaVM.SetField(DreamLib, "base64", DreamBase64)
	LuaVM.SetField(DreamBase64, "encode", LuaVM.NewFunction(luaDreamBase64Encode))
	LuaVM.SetField(DreamBase64, "decode", LuaVM.NewFunction(luaDreamBase64Decode))
	LuaVM.SetField(DreamLib, "md5", DreamMd5)
	LuaVM.SetField(DreamMd5, "hash", LuaVM.NewFunction(luaDreamMd5Hash))
	LuaVM.SetField(DreamLib, "sha256", DreamSha256)
	LuaVM.SetField(DreamSha256, "hash", LuaVM.NewFunction(luaDreamSha256Hash))
	LuaVM.SetField(DreamLib, "BKDR", DreamBKDR)
	LuaVM.SetField(DreamBKDR, "hash", LuaVM.NewFunction(luaDreamBKDRHash))
	LuaVM.SetGlobal("dream", DreamLib)
	//----------------------------------------------------------------
	ZhaoDiceSDK := LuaVM.NewTable()
	ZhaoDiceSDKSystem := LuaVM.NewTable()
	LuaVM.SetField(ZhaoDiceSDK, "trim", LuaVM.NewFunction(luaZhaoDiceSDKTrim))
	LuaVM.SetField(ZhaoDiceSDK, "contains", LuaVM.NewFunction(luaZhaoDiceSDKContains))
	LuaVM.SetField(ZhaoDiceSDK, "system", ZhaoDiceSDKSystem)
	LuaVM.SetField(ZhaoDiceSDKSystem, "reload", LuaVM.NewFunction(luaZhaoDiceSDKSystemReload))
	LuaVM.SetGlobal("zhaodicesdk", ZhaoDiceSDK)
}
