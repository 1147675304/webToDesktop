// tools/desktop/native/linux.go
// Linux 原生窗口定制：通过 GTK Window 调用 GTK3/WebKit2GTK API
//
//   可用能力：
//     - 窗口透明度
//     - 窗口背景透明（RGBA visual + WebKit WebView 透明背景）
//     - 隐藏标题栏/边框
//     - 窗口置顶
//     - 全屏 / 最大化

//go:build linux

package native

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.0 x11

#include <gtk/gtk.h>
#include <gdk/gdk.h>
#include <webkit2/webkit2.h>
#include <string.h>
#include <stdio.h>

// X11 键盘钩子需要 GDK X11 后端支持
#ifdef GDK_WINDOWING_X11
#include <gdk/gdkx.h>
#include <X11/Xlib.h>
#include <X11/keysym.h>
#endif

// 确保窗口已实现（realize），创建底层 GdkWindow
static void ensureRealized(GtkWindow *win) {
	if (!gtk_widget_get_realized(GTK_WIDGET(win))) {
		gtk_widget_realize(GTK_WIDGET(win));
	}
}

// 设置窗口透明度 (0.0 全透明 ~ 1.0 不透明)
static void setOpacity(GtkWindow *win, double opacity) {
	gtk_widget_set_opacity(GTK_WIDGET(win), opacity);
}

// 移除窗口装饰（标题栏、边框）
static void setUndecorated(GtkWindow *win) {
	gtk_window_set_decorated(win, FALSE);
}

// 强制显示窗口及所有子控件（Wayland 无边框窗口兼容修复）
static void forceShow(GtkWindow *win) {
	gtk_widget_show_all(GTK_WIDGET(win));
	gtk_window_present(win);
}

// 设置窗口置顶（使用 GDK 级别 API，更可靠）
static void setKeepAbove(GtkWindow *win) {
	ensureRealized(win);
	GdkWindow *gdkWin = gtk_widget_get_window(GTK_WIDGET(win));
	if (gdkWin) {
		gdk_window_set_keep_above(gdkWin, TRUE);
	}
	gtk_window_set_keep_above(win, TRUE);
}

// 全屏模式（使用 GDK 级别 API）
static void setFullscreen(GtkWindow *win) {
	ensureRealized(win);
	GdkWindow *gdkWin = gtk_widget_get_window(GTK_WIDGET(win));
	if (gdkWin) {
		gdk_window_fullscreen(gdkWin);
	} else {
		gtk_window_fullscreen(win);
	}
}

// ———— 配置文件尺寸（最大化/全屏恢复时永远使用此尺寸） ————
// ★ 必须在 setWindowPosition / centerWindow 之前定义
static int g_configWidth  = 0;
static int g_configHeight = 0;

static void setConfigSize(int w, int h) {
	g_configWidth  = w;
	g_configHeight = h;
}

// ———— 窗口位置设置 ————
// pos: ""=跳过 "center"=居中 "x,y"=绝对坐标
static void setWindowPosition(GtkWindow *win, const char *pos) {
	if (!pos || !pos[0]) return;

	// ★ 先 resize 到配置尺寸，再获取实际尺寸计算位置
	if (g_configWidth > 0 && g_configHeight > 0) {
		gtk_window_resize(win, g_configWidth, g_configHeight);
	}
	// 处理待处理的 GTK 事件，确保 resize 生效
	while (gtk_events_pending()) gtk_main_iteration();

	gint width, height;
	gtk_window_get_size(win, &width, &height);

	gint x = 0, y = 0;
	if (strcmp(pos, "center") == 0) {
		GdkWindow *gdkWin = gtk_widget_get_window(GTK_WIDGET(win));
		GdkDisplay *display = gdk_window_get_display(gdkWin);
		GdkMonitor *monitor = gdk_display_get_monitor_at_window(display, gdkWin);
		GdkRectangle rect;
		gdk_monitor_get_geometry(monitor, &rect);
		x = rect.x + (rect.width - width) / 2;
		y = rect.y + (rect.height - height) / 2;
	} else if (sscanf(pos, "%d,%d", &x, &y) == 2) {
		// 直接使用解析的 x, y
	} else {
		g_printerr("[native] setWindowPosition: invalid format '%s', expected 'center' or 'x,y'\n", pos);
		return;
	}
	g_print("[native] setWindowPosition: %d,%d (%dx%d)\n", x, y, width, height);
	gtk_window_move(win, x, y);
}

// 窗口最大化（使用 GDK 级别 API）
static void setMaximized(GtkWindow *win) {
	ensureRealized(win);
	GdkWindow *gdkWin = gtk_widget_get_window(GTK_WIDGET(win));
	if (gdkWin) {
		gdk_window_maximize(gdkWin);
	} else {
		gtk_window_maximize(win);
	}
}

// 居中窗口到当前显示器（按配置文件尺寸）
static void centerWindow(GtkWindow *win) {
	if (g_configWidth <= 0 || g_configHeight <= 0) return;
	GdkWindow *gdkWin = gtk_widget_get_window(GTK_WIDGET(win));
	if (!gdkWin) return;
	GdkDisplay *display = gdk_window_get_display(gdkWin);
	GdkMonitor *monitor = gdk_display_get_monitor_at_window(display, gdkWin);
	GdkRectangle rect;
	gdk_monitor_get_geometry(monitor, &rect);
	gint x = rect.x + (rect.width  - g_configWidth)  / 2;
	gint y = rect.y + (rect.height - g_configHeight) / 2;
	gtk_window_resize(win, g_configWidth, g_configHeight);
	gtk_window_move(win, x, y);
}

// ———— 窗口拖拽 / 调整大小 ————
static void startWindowDrag(GtkWindow *win) {
	GdkDisplay *display = gdk_display_get_default();
	GdkSeat *seat = gdk_display_get_default_seat(display);
	GdkDevice *device = gdk_seat_get_pointer(seat);
	gint x, y;
	gdk_device_get_position(device, NULL, &x, &y);
	gtk_window_begin_move_drag(win, 1, x, y, GDK_CURRENT_TIME);
}

// mapHTEdgeToGDK 将 Windows HTxxx 边缘常量 (10-17) 映射到 GdkWindowEdge (0-7)。
static GdkWindowEdge mapHTEdgeToGDK(int edge) {
	switch (edge) {
		case 10: return GDK_WINDOW_EDGE_WEST;        // HTLEFT
		case 11: return GDK_WINDOW_EDGE_EAST;        // HTRIGHT
		case 12: return GDK_WINDOW_EDGE_NORTH;       // HTTOP
		case 13: return GDK_WINDOW_EDGE_NORTH_WEST;  // HTTOPLEFT
		case 14: return GDK_WINDOW_EDGE_NORTH_EAST;  // HTTOPRIGHT
		case 15: return GDK_WINDOW_EDGE_SOUTH;       // HTBOTTOM
		case 16: return GDK_WINDOW_EDGE_SOUTH_WEST;  // HTBOTTOMLEFT
		case 17: return GDK_WINDOW_EDGE_SOUTH_EAST;  // HTBOTTOMRIGHT
		default: return (GdkWindowEdge)edge;
	}
}

static void startWindowResize(GtkWindow *win, int edge) {
	GdkDisplay *display = gdk_display_get_default();
	GdkSeat *seat = gdk_display_get_default_seat(display);
	GdkDevice *device = gdk_seat_get_pointer(seat);
	gint x, y;
	gdk_device_get_position(device, NULL, &x, &y);

	// 注意：不在此处做 1×1 前置，拖拽由 GTK 实时接管。
	gtk_window_begin_resize_drag(win, mapHTEdgeToGDK(edge), 1, x, y, GDK_CURRENT_TIME);

	// 拖拽完成后：1×1 强制 WebKit 释放旧缓冲 → 恢复到当前尺寸
	gint curW, curH;
	gtk_window_get_size(win, &curW, &curH);
	gtk_window_resize(win, 1, 1);
	while (gtk_events_pending()) gtk_main_iteration();
	gtk_window_resize(win, curW, curH);
}

// ———— 窗口控制 ————
static void closeWindow_gtk(GtkWindow *win) {
	gtk_window_close(win);
}

static void toggleMaximize_gtk(GtkWindow *win) {
	// 1×1 前置：强制 WebKit 释放布局缓存，再执行最大化/还原
	gtk_window_resize(win, 1, 1);
	while (gtk_events_pending()) gtk_main_iteration();

	if (gtk_window_is_maximized(win)) {
		gtk_window_unmaximize(win);
		// ★ 还原后强制设为配置文件尺寸 + 居中
		while (gtk_events_pending()) gtk_main_iteration();
		centerWindow(win);
	} else {
		gtk_window_maximize(win);
	}
}

static void toggleFullscreen_gtk(GtkWindow *win) {
	// 1×1 前置：强制 WebKit 释放布局缓存，再执行全屏/退出全屏
	gtk_window_resize(win, 1, 1);
	while (gtk_events_pending()) gtk_main_iteration();

	GdkWindow *gdkWin = gtk_widget_get_window(GTK_WIDGET(win));
	if (gdkWin && gdk_window_get_state(gdkWin) & GDK_WINDOW_STATE_FULLSCREEN) {
		gtk_window_unfullscreen(win);
		// ★ 退出全屏后强制设为配置文件尺寸 + 居中
		while (gtk_events_pending()) gtk_main_iteration();
		centerWindow(win);
	} else {
		gtk_window_fullscreen(win);
	}
}

// ———— 窗口背景透明（Linux 毛玻璃/透明背景效果） ————

// 设置窗口使用 RGBA visual，使窗口支持 alpha 通道（透明）。
static void setWindowVisualRGBA(GtkWindow *win) {
	GdkScreen *screen = gtk_window_get_screen(win);
	GdkVisual *visual = gdk_screen_get_rgba_visual(screen);
	if (visual) {
		gtk_widget_set_visual(GTK_WIDGET(win), visual);
	}
}

// 启用窗口背景透明：RGBA visual + CSS 透明背景。
// 使用 GTK CSS Provider 而非 draw 信号，避免阻止子控件（WebView）绘制。
static void enableWindowBgTransparent(GtkWindow *win) {
	setWindowVisualRGBA(win);

	GtkCssProvider *provider = gtk_css_provider_new();
	const char *css = "window { background-color: transparent; }";
	gtk_css_provider_load_from_data(provider, css, -1, NULL);
	GtkStyleContext *context = gtk_widget_get_style_context(GTK_WIDGET(win));
	gtk_style_context_add_provider(context, GTK_STYLE_PROVIDER(provider),
		GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
	g_object_unref(provider);
}

// 查找 GtkWindow 的直接子控件 WebKitWebView，设置其背景颜色为透明。
// webview_go 将 WebKitWebView 作为 GtkWindow 的直接子控件添加。
static void setWebKitBgTransparent(GtkWindow *win) {
	GtkWidget *child = gtk_bin_get_child(GTK_BIN(win));
	if (!child) return;

	// 检查是否为 WebKitWebView 类型（通过 GObject 类型名称）
	if (g_strcmp0(G_OBJECT_TYPE_NAME(G_OBJECT(child)), "WebKitWebView") == 0) {
		GdkRGBA transparent = {0.0, 0.0, 0.0, 0.0};
		webkit_web_view_set_background_color(WEBKIT_WEB_VIEW(child), &transparent);
	}
}

// ———— 输入穿透控制 ————

// 显式设置窗口输入区域，防止合成器将透明区域自动穿透。
// region=NULL → 整个窗口区域接受输入（覆盖合成器默认行为）。
static void setInputShapeFull(GtkWindow *win) {
	gtk_widget_input_shape_combine_region(GTK_WIDGET(win), NULL);
}

// 禁用 WebKitWebView 的 GPU 硬件加速（仅影响当前 WebView 内容渲染）。
// 用于龙芯+麒麟等 GPU 驱动与合成器不兼容的平台。
static void disableWebKitHardwareAccel(GtkWindow *win) {
	GtkWidget *child = gtk_bin_get_child(GTK_BIN(win));
	if (!child) return;
	if (g_strcmp0(G_OBJECT_TYPE_NAME(G_OBJECT(child)), "WebKitWebView") == 0) {
		WebKitSettings *settings = webkit_web_view_get_settings(WEBKIT_WEB_VIEW(child));
		webkit_settings_set_hardware_acceleration_policy(settings,
			WEBKIT_HARDWARE_ACCELERATION_POLICY_NEVER);
		g_print("[native] WebKit hardware acceleration disabled\n");
	}
}

// 设置应用默认图标（在窗口创建前调用，所有窗口自动继承）。
static void setDefaultAppIcon(const char *path) {
	GError *err = NULL;
	gtk_window_set_default_icon_from_file(path, &err);
	if (err) {
		g_printerr("[native] setDefaultAppIcon failed: %s\n", err->message);
		g_error_free(err);
	}
}

// 从 PNG 文件设置单个窗口图标。
static void setWindowIcon(GtkWindow *win, const char *path) {
	GError *err = NULL;
	gtk_window_set_icon_from_file(win, path, &err);
	if (err) {
		g_printerr("[native] setWindowIcon failed: %s\n", err->message);
		g_error_free(err);
	}
}

// ———— 键盘快捷键拦截 ————
// 通过 C 层静态数组管理，Go 侧通过 syncBlockedKeysToC 同步

// C 层屏蔽键列表（由 Go 端 syncBlockedKeysToC 维护）
#define MAX_BLOCKED_KEYS 128
static char g_blockedKeys[MAX_BLOCKED_KEYS][128];
static int g_blockedCount = 0;

// 清空 C 层屏蔽键列表
static void clearBlockedKeysC(void) {
    g_blockedCount = 0;
}

// 向 C 层列表添加一条屏蔽键
static void addBlockedKeyC(const char* key) {
    if (g_blockedCount >= MAX_BLOCKED_KEYS) return;
    strncpy(g_blockedKeys[g_blockedCount], key, 127);
    g_blockedKeys[g_blockedCount][127] = '\0';
    g_blockedCount++;
}

// 查询 C 层列表，返回 TRUE=应拦截
static int isKeyBlockedC(const char* key) {
    for (int i = 0; i < g_blockedCount; i++) {
        if (strcmp(g_blockedKeys[i], key) == 0) {
            return 1;
        }
    }
    return 0;
}

// ———— 键盘事件通知（供前端监听） ————
static volatile int g_kbEventCounter = 0;
static char g_kbEventKey[128] = {0};

int getKbEventCounterC(void) { return g_kbEventCounter; }

void popKbEventC(char* buf, int bufsize) {
    strncpy(buf, g_kbEventKey, bufsize - 1);
    buf[bufsize - 1] = '\0';
    g_kbEventKey[0] = '\0';
    g_kbEventCounter = 0;
}

void pushKbEvent(const char* key) {
    strncpy(g_kbEventKey, key, sizeof(g_kbEventKey) - 1);
    g_kbEventKey[sizeof(g_kbEventKey) - 1] = '\0';
    g_kbEventCounter++;
}

// 保存信号处理器 ID，用于卸载
static guint g_kbSignalId = 0;

// 构建按键描述字符串并查询 C 层屏蔽列表
// 格式: "Alt+Tab", "Ctrl+Shift+S", "Super_L"
// 返回 TRUE=应拦截, FALSE=放行
static gboolean checkDynamicRegistry(guint keyval, guint state) {
    char desc[128] = {0};

    // 检测修饰键 (GDK 修饰符掩码)
    if (state & GDK_CONTROL_MASK) {
        strcat(desc, "Ctrl+");
    }
    if (state & GDK_MOD1_MASK) {
        strcat(desc, "Alt+");
    }
    if (state & GDK_SHIFT_MASK) {
        strcat(desc, "Shift+");
    }
    // Super 键 (GDK_SUPER_MASK) 不作为修饰符前缀，而是直接作为键名处理

    // 按键名映射
    switch (keyval) {
        case GDK_KEY_Super_L:       strcat(desc, "Super_L"); break;
        case GDK_KEY_Super_R:       strcat(desc, "Super_R"); break;
        case GDK_KEY_Alt_L:
        case GDK_KEY_Alt_R:         strcat(desc, "Alt_L"); break;
        case GDK_KEY_Control_L:
        case GDK_KEY_Control_R:     strcat(desc, "Control_L"); break;
        case GDK_KEY_Shift_L:
        case GDK_KEY_Shift_R:       strcat(desc, "Shift_L"); break;
        case GDK_KEY_Tab:           strcat(desc, "Tab"); break;
        case GDK_KEY_F4:            strcat(desc, "F4"); break;
        case GDK_KEY_Escape:        strcat(desc, "Esc"); break;
        case GDK_KEY_space:         strcat(desc, "Space"); break;
        case GDK_KEY_Return:        strcat(desc, "Enter"); break;
        case GDK_KEY_BackSpace:     strcat(desc, "Backspace"); break;
        case GDK_KEY_Delete:        strcat(desc, "Delete"); break;
        case GDK_KEY_Insert:        strcat(desc, "Insert"); break;
        case GDK_KEY_Home:          strcat(desc, "Home"); break;
        case GDK_KEY_End:           strcat(desc, "End"); break;
        case GDK_KEY_Page_Up:       strcat(desc, "PageUp"); break;
        case GDK_KEY_Page_Down:     strcat(desc, "PageDown"); break;
        case GDK_KEY_Left:          strcat(desc, "Left"); break;
        case GDK_KEY_Right:         strcat(desc, "Right"); break;
        case GDK_KEY_Up:            strcat(desc, "Up"); break;
        case GDK_KEY_Down:          strcat(desc, "Down"); break;
        case GDK_KEY_F1:  case GDK_KEY_F2:  case GDK_KEY_F3:  case GDK_KEY_F5:
        case GDK_KEY_F6:  case GDK_KEY_F7:  case GDK_KEY_F8:  case GDK_KEY_F9:
        case GDK_KEY_F10: case GDK_KEY_F11: case GDK_KEY_F12:
        {
            char tmp[16];
            sprintf(tmp, "F%d", (keyval - GDK_KEY_F1 + 1));
            strcat(desc, tmp);
            break;
        }
        default: {
            // GDK 键值可能是 Unicode 字符
            if (keyval >= 0x20 && keyval <= 0x7E) {
                // 可打印 ASCII 字符
                char tmp[2] = { (char)keyval, 0 };
                strcat(desc, tmp);
            } else {
                // 用十六进制键值表示
                char tmp[32];
                sprintf(tmp, "KEY_0x%X", (unsigned int)keyval);
                strcat(desc, tmp);
            }
            break;
        }
    }

    // 查询 C 层屏蔽列表（纯 C 检查，不回调 Go）
    if (isKeyBlockedC(desc)) {
        pushKbEvent(desc);
        return TRUE;
    }
    return FALSE;
}

// GTK 键盘事件处理器 — 拦截注册的快捷键
static gboolean keyboardPressHandler(GtkWidget *widget, GdkEventKey *event, gpointer user_data) {
	(void)(widget);
	(void)(user_data);

    guint kv = event->keyval;
    guint st = event->state;

    // ★ 检查动态注册中心（支持前端自定义快捷键和默认拦截列表）
    if (checkDynamicRegistry(kv, st)) {
        return TRUE;
    }

	return FALSE; // 放行
}

// X11 全局按键拦截（解决 Alt+Tab 等被窗口管理器截获的问题）
// 通过 XGrabKey 在窗口上设置被动抓取，使这些按键到达我们的窗口处理器
static void x11GrabSystemKeys(GtkWindow *win) {
#if defined(GDK_WINDOWING_X11)
	GdkDisplay *display = gtk_widget_get_display(GTK_WIDGET(win));
	if (!GDK_IS_X11_DISPLAY(display)) {
		g_print("[native] not X11 display, skipping X11 grabs\n");
		return;
	}

	Display *xdisplay = gdk_x11_display_get_xdisplay(display);
	if (!xdisplay) return;

	Window xwindow = gdk_x11_window_get_xid(gtk_widget_get_window(GTK_WIDGET(win)));
	if (!xwindow) return;

	// 获取按键码
	KeyCode tabCode = XKeysymToKeycode(xdisplay, XK_Tab);
	KeyCode f4Code  = XKeysymToKeycode(xdisplay, XK_F4);
	KeyCode escCode = XKeysymToKeycode(xdisplay, XK_Escape);
	KeyCode superL  = XKeysymToKeycode(xdisplay, XK_Super_L);
	KeyCode superR  = XKeysymToKeycode(xdisplay, XK_Super_R);

	// 在窗口上拦截 Alt+Tab, Alt+F4, Alt+Esc
	// 当窗口拥有焦点时，这些按键不会传递给窗口管理器
	XGrabKey(xdisplay, tabCode, Mod1Mask, xwindow, False, GrabModeAsync, GrabModeAsync);
	XGrabKey(xdisplay, f4Code,  Mod1Mask, xwindow, False, GrabModeAsync, GrabModeAsync);
	XGrabKey(xdisplay, escCode, Mod1Mask, xwindow, False, GrabModeAsync, GrabModeAsync);

	// 在根窗口上全局拦截 Super 键（任意修饰符）
	// 使其无论焦点在哪个窗口都无法触发系统菜单
	Window root = DefaultRootWindow(xdisplay);
	XGrabKey(xdisplay, superL, AnyModifier, root, False, GrabModeAsync, GrabModeAsync);
	XGrabKey(xdisplay, superR, AnyModifier, root, False, GrabModeAsync, GrabModeAsync);

	XFlush(xdisplay);
	g_print("[native] X11 game mode: grabbed system keys\n");
#else
	(void)(win);
#endif
}

static void x11UngrabSystemKeys(GtkWindow *win) {
#if defined(GDK_WINDOWING_X11)
	GdkDisplay *display = gtk_widget_get_display(GTK_WIDGET(win));
	if (!GDK_IS_X11_DISPLAY(display)) return;

	Display *xdisplay = gdk_x11_display_get_xdisplay(display);
	if (!xdisplay) return;

	Window root = DefaultRootWindow(xdisplay);
	XUngrabKey(xdisplay, AnyKey, AnyModifier, root);

	Window xwindow = gdk_x11_window_get_xid(gtk_widget_get_window(GTK_WIDGET(win)));
	if (xwindow) {
		XUngrabKey(xdisplay, AnyKey, AnyModifier, xwindow);
	}

	XFlush(xdisplay);
	g_print("[native] X11 game mode: released system keys\n");
#else
	(void)(win);
#endif
}

// 启用键盘快捷键拦截 — 安装 GTK 按键拦截 + X11 全局拦截
static void enableKeyboardHook(GtkWindow *win) {
	if (g_kbSignalId != 0) return; // 已经启用

	// 连接 GTK key-press-event 信号，拦截到达窗口的按键
	g_kbSignalId = g_signal_connect(win, "key-press-event",
		G_CALLBACK(keyboardPressHandler), NULL);

	// X11 全局拦截（窗口管理器级别的快捷键）
	x11GrabSystemKeys(win);

	g_print("[native] keyboard hook enabled\n");
}

// 禁用键盘快捷键拦截 — 卸载所有按键拦截
static void disableKeyboardHook(GtkWindow *win) {
	if (g_kbSignalId != 0) {
		g_signal_handler_disconnect(win, g_kbSignalId);
		g_kbSignalId = 0;
	}

	x11UngrabSystemKeys(win);

	g_print("[native] keyboard hook disabled\n");
}

// ———— 系统托盘（GtkStatusIcon） ————
static GtkStatusIcon *g_statusIcon = NULL;
static GtkWindow *g_trayWindow = NULL;  // 关联的主窗口
static gulong g_trayDeleteHandler = 0;  // delete-event 信号处理器 ID

// 托盘菜单项回调
static void onTrayShow(GtkMenuItem *item, gpointer data) {
    (void)(item); (void)(data);
    if (g_trayWindow) {
        gtk_window_deiconify(g_trayWindow);
        gtk_window_present(g_trayWindow);
    }
}

static void onTrayQuit(GtkMenuItem *item, gpointer data) {
    (void)(item); (void)(data);
    // 移除托盘图标
    if (g_statusIcon) {
        gtk_status_icon_set_visible(g_statusIcon, FALSE);
        g_object_unref(g_statusIcon);
        g_statusIcon = NULL;
    }
    // 关闭窗口（此时不再拦截 delete-event）
    if (g_trayWindow) {
        if (g_trayDeleteHandler != 0) {
            g_signal_handler_disconnect(g_trayWindow, g_trayDeleteHandler);
            g_trayDeleteHandler = 0;
        }
        gtk_window_close(g_trayWindow);
    }
    gtk_main_quit();
}

// 左键双击：显示窗口
static void onTrayActivate(GtkStatusIcon *icon, gpointer data) {
    (void)(icon); (void)(data);
    if (g_trayWindow) {
        gtk_window_deiconify(g_trayWindow);
        gtk_window_present(g_trayWindow);
    }
}

// 右键菜单
static void onTrayPopupMenu(GtkStatusIcon *icon, guint button, guint activate_time, gpointer data) {
    (void)(icon); (void)(button); (void)(data);
    GtkWidget *menu = gtk_menu_new();

    GtkWidget *showItem = gtk_menu_item_new_with_label("显示窗口");
    g_signal_connect(showItem, "activate", G_CALLBACK(onTrayShow), NULL);
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), showItem);

    GtkWidget *sep = gtk_separator_menu_item_new();
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), sep);

    GtkWidget *quitItem = gtk_menu_item_new_with_label("退出");
    g_signal_connect(quitItem, "activate", G_CALLBACK(onTrayQuit), NULL);
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), quitItem);

    gtk_widget_show_all(menu);
    gtk_menu_popup(GTK_MENU(menu), NULL, NULL,
        gtk_status_icon_position_menu, icon,
        button, activate_time);
}

// 拦截窗口关闭 → 隐藏到托盘
static gboolean onWindowDeleteForTray(GtkWidget *widget, GdkEvent *event, gpointer data) {
    (void)(widget); (void)(event); (void)(data);
    if (g_statusIcon) {
        gtk_widget_hide(widget);
        return TRUE; // 阻止默认关闭行为
    }
    return FALSE; // 正常关闭
}

static void createTrayIcon(GtkWindow *win, const char *iconPath) {
    if (g_statusIcon) return;

    g_trayWindow = win;
    g_statusIcon = gtk_status_icon_new();

    // 尝试从文件加载图标，失败则用系统默认图标
    if (iconPath && iconPath[0]) {
        gtk_status_icon_set_from_file(g_statusIcon, iconPath);
    } else {
        gtk_status_icon_set_from_icon_name(g_statusIcon, "applications-system");
    }
    gtk_status_icon_set_tooltip_text(g_statusIcon, "WebToDesktop");
    gtk_status_icon_set_visible(g_statusIcon, TRUE);

    // 信号连接
    g_signal_connect(g_statusIcon, "activate", G_CALLBACK(onTrayActivate), NULL);
    g_signal_connect(g_statusIcon, "popup-menu", G_CALLBACK(onTrayPopupMenu), NULL);

    // 拦截窗口关闭 → 隐藏到托盘
    g_trayDeleteHandler = g_signal_connect(win, "delete-event",
        G_CALLBACK(onWindowDeleteForTray), NULL);

    g_print("[native] createTrayIcon: OK\n");
}

static void removeTrayIcon(void) {
    if (!g_statusIcon) return;

    // 移除 delete-event 拦截
    if (g_trayWindow && g_trayDeleteHandler != 0) {
        g_signal_handler_disconnect(g_trayWindow, g_trayDeleteHandler);
        g_trayDeleteHandler = 0;
    }

    gtk_status_icon_set_visible(g_statusIcon, FALSE);
    g_object_unref(g_statusIcon);
    g_statusIcon = NULL;
    g_trayWindow = NULL;
    g_print("[native] removeTrayIcon: OK\n");
}
*/
import "C"
import (
	"os"
	"strings"
	"unsafe"
)

// Apply Linux 平台外观配置。
// 在 w.Dispatch() 回调中调用（确保 UI 线程）。
// 执行顺序: 全屏/最大化/位置 → 透明度 → 去边框 → 置顶
func Apply(winPtr unsafe.Pointer, cfg WindowConfig) {
	win := (*C.GtkWindow)(winPtr)

	// 1. 全屏/最大化/窗口位置（互斥，仅三选一）
	if cfg.Fullscreen {
		C.setFullscreen(win)
	} else if cfg.Maximized {
		C.setMaximized(win)
	} else {
		// 非最大化/非全屏 → resize 到配置尺寸 + 定位
		if cfg.WindowPosition == "" || cfg.WindowPosition == "center" {
			C.centerWindow(win)
		} else {
			cPos := C.CString(cfg.WindowPosition)
			C.setWindowPosition(win, cPos)
			C.free(unsafe.Pointer(cPos))
		}
	}

	// 2. 透明度
	if cfg.Opacity < 1.0 {
		C.setOpacity(win, C.double(cfg.Opacity))
	}

	// 3. 去边框（Apply 中的冗余保护，主要去边框工作已由 ApplyPreShow 完成）
	if cfg.Borderless {
		C.setUndecorated(win)
	}

	// 4. 窗口置顶
	if cfg.AlwaysOnTop {
		C.setKeepAbove(win)
	}

	// 5. 强制显示（Wayland 无边框窗口兼容）
	C.forceShow(win)

	// 6. WebView 背景透明：在 Dispatch 中执行（此时 WebView 已实现并作为子控件存在）
	if cfg.WebViewBgTransparent {
		C.setWebKitBgTransparent(win)
	}

	// 7. 输入穿透：false=捕获所有点击（显式设置全窗口输入形状）
	if !cfg.InputPassthrough {
		C.setInputShapeFull(win)
	}

	// 8. 键盘快捷键拦截（按键映射提前注册，钩子安装延迟到 webview.go 中执行）
	if cfg.KeyboardShortcuts {
		if len(cfg.KeyMappings) > 0 {
			InitKeyMappings(cfg.KeyMappings)
		}
	}

	// 9. 系统托盘图标
	if cfg.SystemTray {
		cPath := C.CString(g_trayIconPath)
		C.createTrayIcon(win, cPath)
		C.free(unsafe.Pointer(cPath))
	} else {
		C.removeTrayIcon()
	}

	// Acrylic/DarkTitleBar/RoundCorners 为 Windows 专属，Linux 忽略
}

// ApplyPreShow 在窗口显示前调用（GTK decoration 必须在 show 前设置）。
func ApplyPreShow(winPtr unsafe.Pointer, cfg WindowConfig) {
	if cfg.Borderless {
		C.setUndecorated((*C.GtkWindow)(winPtr))
	}
	if cfg.AlwaysOnTop {
		C.setKeepAbove((*C.GtkWindow)(winPtr))
	}
	// 窗口背景透明：必须在 show 前设置 RGBA visual + app-paintable
	if cfg.WebViewBgTransparent {
		C.enableWindowBgTransparent((*C.GtkWindow)(winPtr))
	}
}

// 调整方向常量（与 Windows 一致）
const (
	ResizeLeft        = 10
	ResizeRight       = 11
	ResizeTop         = 12
	ResizeTopLeft     = 13
	ResizeTopRight    = 14
	ResizeBottom      = 15
	ResizeBottomLeft  = 16
	ResizeBottomRight = 17
)

func SetWebView2BackgroundColor(ctrlPtr unsafe.Pointer, a, r, g, b byte)     {}
func ReapplyAcrylic(winPtr unsafe.Pointer)                                   {}
func EnableBorderlessResize(winPtr unsafe.Pointer)                           {}
func ResizeWebView2Controller(ctrlPtr unsafe.Pointer, winPtr unsafe.Pointer) {}

// DisableWebKitHardwareAccel 禁用 WebKit 渲染的 GPU 硬件加速。
// 龙芯+麒麟等平台 GPU 驱动不兼容时调用，仅影响 WebView 内容，不影响窗口合成。
func DisableWebKitHardwareAccel(winPtr unsafe.Pointer) {
	C.disableWebKitHardwareAccel((*C.GtkWindow)(winPtr))
}

// SetWindowIcon 从 PNG 文件路径设置窗口图标。
func SetWindowIcon(winPtr unsafe.Pointer, iconPath string) {
	cPath := C.CString(iconPath)
	C.setWindowIcon((*C.GtkWindow)(winPtr), cPath)
	C.free(unsafe.Pointer(cPath))
}

// SetDefaultAppIcon 设置应用默认图标（窗口创建前调用，所有窗口自动继承）。
func SetDefaultAppIcon(iconPath string) {
	cPath := C.CString(iconPath)
	C.setDefaultAppIcon(cPath)
	C.free(unsafe.Pointer(cPath))
}

// SetDefaultWindowSize 保存配置中的窗口尺寸（width × height）。
// 最大化/全屏恢复时永远使用此尺寸，而非用户拖拽后的尺寸。
func SetDefaultWindowSize(width, height int) {
	C.setConfigSize(C.int(width), C.int(height))
}

// DragWindow 触发窗口拖拽。
func DragWindow(winPtr unsafe.Pointer) {
	C.startWindowDrag((*C.GtkWindow)(winPtr))
}

// ResizeWindow 从指定边缘触发窗口缩放。
func ResizeWindow(winPtr unsafe.Pointer, edge int) {
	C.startWindowResize((*C.GtkWindow)(winPtr), C.int(edge))
}

// CloseWindow 关闭窗口。
func CloseWindow(winPtr unsafe.Pointer) {
	C.closeWindow_gtk((*C.GtkWindow)(winPtr))
}

// ToggleMaximize 最大化/还原切换。
func ToggleMaximize(winPtr unsafe.Pointer) {
	C.toggleMaximize_gtk((*C.GtkWindow)(winPtr))
}

// ToggleFullscreen 全屏切换。
func ToggleFullscreen(winPtr unsafe.Pointer) {
	C.toggleFullscreen_gtk((*C.GtkWindow)(winPtr))
}

// ToggleMinimize 最小化窗口到任务栏。
func ToggleMinimize(winPtr unsafe.Pointer) {
	C.gtk_window_iconify((*C.GtkWindow)(winPtr))
}

// InitSystemTray 初始化系统托盘图标（创建 GtkStatusIcon + 拦截 delete-event）。
func InitSystemTray(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	cPath := C.CString(g_trayIconPath)
	C.createTrayIcon((*C.GtkWindow)(winPtr), cPath)
	C.free(unsafe.Pointer(cPath))
}

// RemoveSystemTray 移除系统托盘图标。
func RemoveSystemTray(winPtr unsafe.Pointer) {
	C.removeTrayIcon()
}

// ShowWindowRestore 显示窗口并置前（用于托盘"显示窗口"）。
func ShowWindowRestore(winPtr unsafe.Pointer) {
	win := (*C.GtkWindow)(winPtr)
	C.gtk_window_deiconify(win)
	C.gtk_window_present(win)
}

var g_trayIconPath string

// SetTrayIconPath 设置托盘图标文件路径（Linux 在创建托盘前调用）。
func SetTrayIconPath(path string) {
	g_trayIconPath = path
}

// EnableInputPassthrough 启用输入穿透（Linux 下默认穿透，调用此函数无需额外操作）。
// Linux 通过不设置 input shape 来实现穿透，此函数为确保接口统一而保留。
func EnableInputPassthrough(winPtr unsafe.Pointer) {}

// DisableInputPassthrough 禁用输入穿透（Linux 下通过 setInputShapeFull 捕获所有点击）。
func DisableInputPassthrough(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.setInputShapeFull((*C.GtkWindow)(winPtr))
}

// IsInputPassthrough 查询当前是否启用输入穿透。
// Linux 下始终返回 true（Linux 默认不捕获输入，除非显式调用 DisableInputPassthrough）。
func IsInputPassthrough(winPtr unsafe.Pointer) bool {
	return true
}

// OpaqueRegion 不透光矩形（Linux 下为桩，接口统一）。
type OpaqueRegion struct {
	X, Y, W, H int
}

// SetOpaqueRegions Linux 下精准穿透通过 input shape 实现（目前为桩）。
func SetOpaqueRegions(winPtr unsafe.Pointer, regions []OpaqueRegion) {}

// ClearOpaqueRegions Linux 下清空区域后回退到默认穿透。
func ClearOpaqueRegions(winPtr unsafe.Pointer) {}

func PollPassthroughState(winPtr unsafe.Pointer) (bool, int, int) { return false, 0, 0 }
func PollCursorPos(winPtr unsafe.Pointer) (int32, int32)          { return -1, -1 }

// x11SocketExists 检查 X11 Unix socket 是否实际存在。
func x11SocketExists(display string) bool {
	idx := strings.LastIndex(display, ":")
	if idx < 0 {
		return false
	}
	numStr := display[idx+1:]
	if dotIdx := strings.IndexByte(numStr, '.'); dotIdx >= 0 {
		numStr = numStr[:dotIdx]
	}
	if numStr == "" {
		return false
	}
	_, err := os.Stat("/tmp/.X11-unix/X" + numStr)
	return err == nil
}

// waylandSocketExists 检查 Wayland socket 是否实际存在。
func waylandSocketExists(display string) bool {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/run/user/"
	}
	if _, err := os.Stat(dir + "/" + display); err == nil {
		return true
	}
	_, err := os.Stat("/mnt/wslg/runtime-dir/" + display)
	return err == nil
}

// HasDisplay 检查是否有可用的显示服务器（X11 或 Wayland）。
func HasDisplay() bool {
	if d := os.Getenv("DISPLAY"); d != "" && x11SocketExists(d) {
		return true
	}
	if d := os.Getenv("WAYLAND_DISPLAY"); d != "" && waylandSocketExists(d) {
		return true
	}
	return false
}

// syncBlockedKeysToC 将 Go 注册表中的快捷键同步到 C 层静态数组。
// keys 为 nil 时清空 C 层数组。
func syncBlockedKeysToC(keys []string) {
	C.clearBlockedKeysC()
	for _, k := range keys {
		cKey := C.CString(k)
		C.addBlockedKeyC(cKey)
		C.free(unsafe.Pointer(cKey))
	}
}

// syncKeyMappingsToC — 按键映射已移除，保留为空桩避免编译错误
func syncKeyMappingsToC(mappings map[string]string) {}

// EnableKeyboardHook 安装 GTK 按键拦截 + X11 全局键盘抓取。
// 在 w.Dispatch 回调中调用（需在 UI 线程）。
// defaults: 构建时配置的默认拦截快捷键列表。
func EnableKeyboardHook(winPtr unsafe.Pointer, defaults []string) {
	if winPtr == nil {
		return
	}
	// 初始化默认快捷键
	InitDefaultBlockedShortcuts(defaults)
	C.enableKeyboardHook((*C.GtkWindow)(winPtr))
}

// DisableKeyboardHook 卸载所有按键拦截，恢复系统快捷键。
func DisableKeyboardHook(winPtr unsafe.Pointer) {
	if winPtr == nil {
		return
	}
	C.disableKeyboardHook((*C.GtkWindow)(winPtr))
	ClearShortcuts()
}

// PollKbEvent 查询是否有被拦截的键盘事件。
// 返回事件描述字符串，无事件时返回空字符串。
// 每次调用仅返回最近一次事件（消费后重置）。
func PollKbEvent() string {
	if C.getKbEventCounterC() == 0 {
		return ""
	}
	var buf [128]C.char
	C.popKbEventC(&buf[0], 128)
	return C.GoString(&buf[0])
}
