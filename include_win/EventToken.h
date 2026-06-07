// tools/desktop/include_win/EventToken.h
// 最小化 EventRegistrationToken 定义，弥补 MinGW 缺失的 WinRT/WRL 头文件。
// WebView2.h 仅使用 EventRegistrationToken 作为 COM 事件注册的 token 类型。
#pragma once

#ifndef __eventtoken_h__
#define __eventtoken_h__

#ifdef __cplusplus
extern "C" {
#endif

typedef struct EventRegistrationToken {
    __int64 value;
} EventRegistrationToken;

#ifdef __cplusplus
}
#endif

#endif // __eventtoken_h__
