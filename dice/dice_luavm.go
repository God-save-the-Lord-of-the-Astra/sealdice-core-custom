package dice

import (
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func LuaVarInit(LuaVM *lua.LState, ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
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
	LuaVM.SetGlobal("Message", msgMeta)
	LuaVM.SetField(msgMeta, "__index", LuaVM.SetFuncs(LuaVM.NewTable(), map[string]lua.LGFunction{
		"Time": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LNumber(msg.Time))
			return 1
		},
		"MessageType": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LString(msg.MessageType))
			return 1
		},
		"GroupID": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LString(msg.GroupID))
			return 1
		},
		"GuildID": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LString(msg.GuildID))
			return 1
		},
		"ChannelID": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LString(msg.ChannelID))
			return 1
		},
		"Message": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LString(msg.Message))
			return 1
		},
		"Platform": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LString(msg.Platform))
			return 1
		},
		"GroupName": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
			L.Push(lua.LString(msg.GroupName))
			return 1
		},
		"Sender": func(L *lua.LState) int {
			msg := L.CheckUserData(1).Value.(*Message)
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
	LuaVM.SetGlobal("MsgContext", ctxMeta)
	LuaVM.SetField(ctxMeta, "__index", LuaVM.SetFuncs(LuaVM.NewTable(), map[string]lua.LGFunction{
		"MessageType": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LString(ctx.MessageType))
			return 1
		},
		"IsCurGroupBotOn": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LBool(ctx.IsCurGroupBotOn))
			return 1
		},
		"IsPrivate": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LBool(ctx.IsPrivate))
			return 1
		},
		"CommandID": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LNumber(ctx.CommandID))
			return 1
		},
		"PrivilegeLevel": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LNumber(ctx.PrivilegeLevel))
			return 1
		},
		"GroupRoleLevel": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LNumber(ctx.GroupRoleLevel))
			return 1
		},
		"DelegateText": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LString(ctx.DelegateText))
			return 1
		},
		"AliasPrefixText": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			L.Push(lua.LString(ctx.AliasPrefixText))
			return 1
		},
		"Group": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
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
				L.Push(lua.LNil)
			}
			return 1
		},
		"Player": func(L *lua.LState) int {
			ctx := L.CheckUserData(1).Value.(*MsgContext)
			if ctx.Player != nil {
				playerTable := LuaVM.NewTable()
				LuaVM.SetField(playerTable, "Name", lua.LString(ctx.Player.Name))
				LuaVM.SetField(playerTable, "UserID", lua.LString(ctx.Player.UserID))
				LuaVM.SetField(playerTable, "LastCommandTime", lua.LNumber(ctx.Player.LastCommandTime))
				LuaVM.SetField(playerTable, "AutoSetNameTemplate", lua.LString(ctx.Player.AutoSetNameTemplate))
				LuaVM.Push(playerTable)
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
	}))

	LuaVM.SetGlobal("ctx", ctxUD)

	//----------------------------------------------------------------
	cmdArgsUD := LuaVM.NewUserData()
	cmdArgsUD.Value = cmdArgs
	cmdArgsMeta := LuaVM.NewTypeMetatable("CmdArgs")
	LuaVM.SetGlobal("CmdArgs", cmdArgsMeta)
	LuaVM.SetField(cmdArgsMeta, "__index", LuaVM.SetFuncs(LuaVM.NewTable(), map[string]lua.LGFunction{
		"Command": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LString(cmdArgs.Command))
			return 1
		},
		"Args": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			argsTable := LuaVM.NewTable()
			for _, arg := range cmdArgs.Args {
				argsTable.Append(lua.LString(arg))
			}
			L.Push(argsTable)
			return 1
		},
		"Kwargs": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			kwargsTable := LuaVM.NewTable()
			for _, kwarg := range cmdArgs.Kwargs {
				kwargTable := LuaVM.NewTable()
				LuaVM.SetField(kwargTable, "Name", lua.LString(kwarg.Name))
				LuaVM.SetField(kwargTable, "ValueExists", lua.LBool(kwarg.ValueExists))
				LuaVM.SetField(kwargTable, "Value", lua.LString(kwarg.Value))
				LuaVM.SetField(kwargTable, "AsBool", lua.LBool(kwarg.AsBool))
				kwargsTable.Append(kwargTable)
			}
			L.Push(kwargsTable)
			return 1
		},
		"At": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			atInfoTable := LuaVM.NewTable()
			for _, at := range cmdArgs.At {
				atTable := LuaVM.NewTable()
				LuaVM.SetField(atTable, "UserID", lua.LString(at.UserID))
				atInfoTable.Append(atTable)
			}
			L.Push(atInfoTable)
			return 1
		},
		"RawArgs": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LString(cmdArgs.RawArgs))
			return 1
		},
		"AmIBeMentioned": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LBool(cmdArgs.AmIBeMentioned))
			return 1
		},
		"AmIBeMentionedFirst": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LBool(cmdArgs.AmIBeMentionedFirst))
			return 1
		},
		"SomeoneBeMentionedButNotMe": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LBool(cmdArgs.SomeoneBeMentionedButNotMe))
			return 1
		},
		"IsSpaceBeforeArgs": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LBool(cmdArgs.IsSpaceBeforeArgs))
			return 1
		},
		"CleanArgs": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LString(cmdArgs.CleanArgs))
			return 1
		},
		"SpecialExecuteTimes": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LNumber(cmdArgs.SpecialExecuteTimes))
			return 1
		},
		"RawText": func(L *lua.LState) int {
			cmdArgs := L.CheckUserData(1).Value.(*CmdArgs)
			L.Push(lua.LString(cmdArgs.RawText))
			return 1
		},
	}))

	LuaVM.SetGlobal("cmdArgs", cmdArgsUD)

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
}

// ----------------------------------------------------------------
func luaVarSetValueStr(LuaVM *lua.LState) int {
	ud := LuaVM.GetGlobal("ctx_ud").(*lua.LUserData)
	ctx := ud.Value.(*MsgContext)

	s := LuaVM.ToString(1)
	v := LuaVM.ToString(2)
	VarSetValueStr(ctx, s, v)
	return 0 // 返回 0 表示无返回值
}

func luaVarSetValueInt(LuaVM *lua.LState) int {
	ud := LuaVM.GetGlobal("ctx_ud").(*lua.LUserData)
	ctx := ud.Value.(*MsgContext)

	s := LuaVM.ToString(1)
	v := LuaVM.ToInt64(2)
	VarSetValueInt64(ctx, s, v)
	return 0 // 返回 0 表示无返回值
}

func luaVarDelValue(LuaVM *lua.LState) int {
	ud := LuaVM.GetGlobal("ctx_ud").(*lua.LUserData)
	ctx := ud.Value.(*MsgContext)

	s := LuaVM.ToString(1)
	VarDelValue(ctx, s)
	return 0 // 返回 0 表示无返回值
}

func luaVarGetValueInt(LuaVM *lua.LState) int {
	ud := LuaVM.GetGlobal("ctx_ud").(*lua.LUserData)
	ctx := ud.Value.(*MsgContext)

	s := LuaVM.ToString(1)
	res, exists := VarGetValueInt64(ctx, s)
	if !exists {
		return 0 // 返回 0 表示没有值
	}
	LuaVM.Push(lua.LNumber(res)) // 推送结果到 Lua 栈
	return 1                     // 返回 1 表示成功
}

func luaVarGetValueStr(LuaVM *lua.LState) int {
	ud := LuaVM.GetGlobal("ctx_ud").(*lua.LUserData)
	ctx := ud.Value.(*MsgContext)

	s := LuaVM.ToString(1)
	res, exists := VarGetValueStr(ctx, s)
	if !exists {
		return 0 // 返回 0 表示没有值
	}
	LuaVM.Push(lua.LString(res)) // 推送结果到 Lua 栈
	return 1                     // 返回 1 表示成功
}

func LuaFuncInit(LuaVM *lua.LState, ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	ctxUerData := LuaVM.NewUserData()
	ctxUerData.Value = ctx
	msgUerData := LuaVM.NewUserData()
	msgUerData.Value = msg
	cmdArgsUerData := LuaVM.NewUserData()
	cmdArgsUerData.Value = cmdArgs
	LuaVM.SetGlobal("ctx_ud", ctxUerData)
	LuaVM.SetGlobal("msg_ud", msgUerData)
	LuaVM.SetGlobal("cmdArgs_ud", cmdArgsUerData)

	// 注册函数
	LuaVM.SetGlobal("VarSetValueStr", LuaVM.NewFunction(luaVarSetValueStr))
	LuaVM.SetGlobal("VarSetValueInt", LuaVM.NewFunction(luaVarSetValueInt))
	LuaVM.SetGlobal("VarDelValue", LuaVM.NewFunction(luaVarDelValue))
	LuaVM.SetGlobal("VarGetValueInt", LuaVM.NewFunction(luaVarGetValueInt))
	LuaVM.SetGlobal("VarGetValueStr", LuaVM.NewFunction(luaVarGetValueStr))

}
