[root@localhost webtodesktop]# make
可构建的前端项目:

  [1] demo                 — WebToDesktop 功能演示


请选择项目编号 (输入序号或名称): 1

请选择目标平台:
  [1] 当前平台
  [2] Linux
  [3] Windows
  [4] Windows + 控制台 (调试用)

请选择平台编号 [默认: 1]: 
已选择项目: demo  目标平台: current

项目目录: demo
远程服务器: about:blank
代理前缀:   /api/,/storage/
签名请求头: X-Desktop-Signature

>>> 复制前端产物 (demo)...


>>> Gzip 压缩前端资源 (-9)...

>>> 构建 Go 二进制 (current)...
  目标: 当前平台
# github.com/webview/webview_go
In file included from /usr/include/glib-2.0/gobject/gobject.h:24:0,
                 from /usr/include/glib-2.0/gobject/gbinding.h:29,
                 from /usr/include/glib-2.0/glib-object.h:22,
                 from /usr/include/glib-2.0/gio/gioenums.h:28,
                 from /usr/include/glib-2.0/gio/giotypes.h:28,
                 from /usr/include/glib-2.0/gio/gio.h:26,
                 from /usr/include/gtk-3.0/gdk/gdkapplaunchcontext.h:28,
                 from /usr/include/gtk-3.0/gdk/gdk.h:32,
                 from /usr/include/gtk-3.0/gtk/gtk.h:30,
                 from /root/go/pkg/mod/github.com/webview/webview_go@v0.0.0-20240831120633-6173450d4dd6/libs/webview/include/webview.h:1108,
                 from webview.cc:1:
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h: 在函数‘GPowerProfileMonitor* g_power_profile_monitor(gpointer)’中:
/usr/include/glib-2.0/gobject/gtype.h:1581:73: 警告：‘GType g_power_profile_monitor_get_type()’ is deprecated: Not available before 2.70 [-Wdeprecated-declarations]
     return G_TYPE_CHECK_INSTANCE_CAST (ptr, module_obj_name##_get_type (), ModuleObjName); }               \
                                                                         ^
/usr/include/glib-2.0/gobject/gtype.h:2300:61: 附注：in definition of macro ‘_G_TYPE_CIC’
     ((ct*) g_type_check_instance_cast ((GTypeInstance*) ip, gt))
                                                             ^~
/usr/include/glib-2.0/gobject/gtype.h:1581:12: 附注：in expansion of macro ‘G_TYPE_CHECK_INSTANCE_CAST’
     return G_TYPE_CHECK_INSTANCE_CAST (ptr, module_obj_name##_get_type (), ModuleObjName); }               \
            ^~~~~~~~~~~~~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h:43:1: 附注：in expansion of macro ‘G_DECLARE_INTERFACE’
 G_DECLARE_INTERFACE (GPowerProfileMonitor, g_power_profile_monitor, g, power_profile_monitor, GObject)
 ^~~~~~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h:43:44: 附注：在此声明
 G_DECLARE_INTERFACE (GPowerProfileMonitor, g_power_profile_monitor, g, power_profile_monitor, GObject)
                                            ^
/usr/include/glib-2.0/gobject/gtype.h:1573:9: 附注：in definition of macro ‘G_DECLARE_INTERFACE’
   GType module_obj_name##_get_type (void);                                                                 \
         ^~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h: 在函数‘gboolean g_IS_power_profile_monitor(gpointer)’中:
/usr/include/glib-2.0/gobject/gtype.h:1583:73: 警告：‘GType g_power_profile_monitor_get_type()’ is deprecated: Not available before 2.70 [-Wdeprecated-declarations]
     return G_TYPE_CHECK_INSTANCE_TYPE (ptr, module_obj_name##_get_type ()); }                              \
                                                                         ^
/usr/include/glib-2.0/gobject/gtype.h:2314:60: 附注：in definition of macro ‘_G_TYPE_CIT’
   GTypeInstance *__inst = (GTypeInstance*) ip; GType __t = gt; gboolean __r; \
                                                            ^~
/usr/include/glib-2.0/gobject/gtype.h:1583:12: 附注：in expansion of macro ‘G_TYPE_CHECK_INSTANCE_TYPE’
     return G_TYPE_CHECK_INSTANCE_TYPE (ptr, module_obj_name##_get_type ()); }                              \
            ^~~~~~~~~~~~~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h:43:1: 附注：in expansion of macro ‘G_DECLARE_INTERFACE’
 G_DECLARE_INTERFACE (GPowerProfileMonitor, g_power_profile_monitor, g, power_profile_monitor, GObject)
 ^~~~~~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h:43:44: 附注：在此声明
 G_DECLARE_INTERFACE (GPowerProfileMonitor, g_power_profile_monitor, g, power_profile_monitor, GObject)
                                            ^
/usr/include/glib-2.0/gobject/gtype.h:1573:9: 附注：in definition of macro ‘G_DECLARE_INTERFACE’
   GType module_obj_name##_get_type (void);                                                                 \
         ^~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h: 在函数‘GPowerProfileMonitorInterface* g_power_profile_monitor_GET_IFACE(gpointer)’中:
/usr/include/glib-2.0/gobject/gtype.h:1585:76: 警告：‘GType g_power_profile_monitor_get_type()’ is deprecated: Not available before 2.70 [-Wdeprecated-declarations]
     return G_TYPE_INSTANCE_GET_INTERFACE (ptr, module_obj_name##_get_type (), ModuleObjName##Interface); } \
                                                                            ^
/usr/include/glib-2.0/gobject/gtype.h:2310:103: 附注：in definition of macro ‘_G_TYPE_IGI’
 #define _G_TYPE_IGI(ip, gt, ct)         ((ct*) g_type_interface_peek (((GTypeInstance*) ip)->g_class, gt))
                                                                                                       ^~
/usr/include/glib-2.0/gobject/gtype.h:1585:12: 附注：in expansion of macro ‘G_TYPE_INSTANCE_GET_INTERFACE’
     return G_TYPE_INSTANCE_GET_INTERFACE (ptr, module_obj_name##_get_type (), ModuleObjName##Interface); } \
            ^~~~~~~~~~~~~~~~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h:43:1: 附注：in expansion of macro ‘G_DECLARE_INTERFACE’
 G_DECLARE_INTERFACE (GPowerProfileMonitor, g_power_profile_monitor, g, power_profile_monitor, GObject)
 ^~~~~~~~~~~~~~~~~~~
/usr/include/glib-2.0/gio/gpowerprofilemonitor.h:43:44: 附注：在此声明
 G_DECLARE_INTERFACE (GPowerProfileMonitor, g_power_profile_monitor, g, power_profile_monitor, GObject)
                                            ^
/usr/include/glib-2.0/gobject/gtype.h:1573:9: 附注：in definition of macro ‘G_DECLARE_INTERFACE’
   GType module_obj_name##_get_type (void);                                                                 \
         ^~~~~~~~~~~~~~~

============================================
  构建完成!
============================================
-rwxr-xr-x 1 root root 9.2M  6月  8 11:39 /home/webtodesktop/build/webtodesktop
WebToDesktop v1.0.0 启动中...
远程服务器: about:blank
代理前缀: [/api/ /storage/]
[bridge] scanning 18 exported methods on *bridge.Bridge:
[bridge]   skip Call (not Handle*)
[bridge] auto-registered: clearCredentials → handleClearCredentials
[bridge] auto-registered: closeWindow → handleCloseWindow
[bridge] auto-registered: deleteCredentials → handleDeleteCredentials
[bridge] auto-registered: dragWindow → handleDragWindow
[bridge] auto-registered: getAppInfo → handleGetAppInfo
[bridge] auto-registered: getCredentials → handleGetCredentials
[bridge] auto-registered: getWindowConfig → handleGetWindowConfig
[bridge] auto-registered: listMethods → handleListMethods
[bridge] auto-registered: resizeWindow → handleResizeWindow
[bridge] auto-registered: restartApp → handleRestartApp
[bridge] auto-registered: saveCredentials → handleSaveCredentials
[bridge] auto-registered: saveWindowConfig → handleSaveWindowConfig
[bridge] auto-registered: toggleFullscreen → handleToggleFullscreen
[bridge] auto-registered: toggleMaximize → handleToggleMaximize
[bridge] auto-registered: toggleMinimize → handleToggleMinimize
[bridge]   skip Register (not Handle*)
[bridge]   skip SetWebView (not Handle*)

(webtodesktop:574272): GLib-GObject-CRITICAL **: 11:40:13.622: g_value_set_boxed: assertion 'G_VALUE_HOLDS_BOXED (value)' failed

(WebKitWebProcess:574288): GLib-GObject-CRITICAL **: 11:40:14.627: g_value_set_boxed: assertion 'G_VALUE_HOLDS_BOXED (value)' failed

(gst-plugin-scanner:574307): GStreamer-WARNING **: 11:40:17.860: Failed to load plugin '/usr/lib64/gstreamer-1.0/libgstmatroska.so': /usr/lib64/gstreamer-1.0/libgstmatroska.so: undefined symbol: gst_buffer_new_memdup

(process:574335): GLib-GObject-WARNING **: 11:40:22.536: ../gobject/gsignal.c:2614: signal 'changed::' is invalid for instance '0x5581afb8e430' of type 'GSettings'

(process:574335): GLib-GObject-WARNING **: 11:40:22.537: ../gobject/gsignal.c:2614: signal 'changed::' is invalid for instance '0x5581afb8ee60' of type 'GSettings'

(process:574335): GLib-GObject-WARNING **: 11:40:22.537: ../gobject/gsignal.c:2614: signal 'changed::' is invalid for instance '0x5581afb8e7a0' of type 'GSettings'

(process:574335): GLib-GObject-WARNING **: 11:40:22.538: ../gobject/gsignal.c:2614: signal 'changed::' is invalid for instance '0x5581afb8ed70' of type 'GSettings'

(process:574335): GLib-GObject-WARNING **: 11:40:22.538: ../gobject/gsignal.c:2614: signal 'changed::' is invalid for instance '0x5581afb8ef50' of type 'GSettings'
^A

