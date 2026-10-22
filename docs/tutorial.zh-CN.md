# magpie Windows 版使用教程（品牌绿圆版）

这份教程写给拿到这个 fork 的人：你想在 Windows 上把它从源码跑起来，把桌面和任务栏的图标换成那颗品牌绿圆，并且弄明白托盘里那颗绿圆为什么曾经比旁边的图标小一圈。

功能层面的说明不在这里，请看 [README.zh-CN.md](../README.zh-CN.md) 的「快速开始」和 [完整参考](reference.md)（英文）。

---

## 一、这是什么

这是 [yetone/magpie](https://github.com/yetone/magpie) 的一个 fork。**功能和上游完全一样**，只有一处视觉上的改动：上游在所有地方显示的是一只 magpie 鸟，这个 fork 把它换成了**品牌绿实心圆**。

- 亮色主题的品牌绿：`#14B789`（rgb 20, 183, 137）
- 暗色主题的品牌绿：`#2AD39B`（rgb 42, 211, 155）

换过的地方：

| 地方 | 在哪儿 |
| :-- | :-- |
| 任务栏图标、窗口 / 资源管理器图标 | 注入进 exe 的 PE 资源（`build/windows/winres.json`） |
| 系统托盘图标 | 单独一个多尺寸 ICO（`internal/gui/tray-windows.ico`），见第四节 |
| 应用内顶栏 logo | `internal/gui/assets/index.html` 里内联的 SVG |
| 浏览器 favicon | `internal/gui/assets/icons/favicon.svg` |
| 登录页的 magpie 标记 | `internal/provider/magpie.svg`（供应商登录结束时浏览器落到的那个页面） |
| Agents 列表里 magpie 自己的图标 | `internal/gui/assets/icons/magpie.svg`（当作遮罩用，`app.css` 给了它品牌绿） |
| 站点 logo | `site/public/img/magpie.svg`、`site/public/img/magpie-small.svg` |
| Omarchy 桌面小组件的图形 | `internal/omarchy/widget/Widget.qml` |

**权威的源只有两处**，要改品牌形状就改这两处：

- `build/icon/magpie.svg` —— 圆本身。44 格的 viewBox，圆心在 (22, 22)，半径 21.6：路径从 0.40 走到 43.60。
- `build/icon/gen.go` —— 画法（怎么把 SVG 变成 PNG 和 ICO）和两处绿色：`0x14B789` 与暗色的 `0x2AD39B`。

**PNG 是生成出来的产物，不要手改 PNG。** 改了 SVG 或 `gen.go`，就重跑生成器（第五节），把生成物一起提交。

---

## 二、在 Windows 上从源码构建并跑起来

需要 Go 和 PowerShell 7。三步。

### 1) 注入 PE 图标资源

Windows 上任务栏和资源管理器用的图标在 exe 里面，得先把它塞进去：

```powershell
go run github.com/tc-hib/go-winres@v0.3.3 make --in build/windows/winres.json --arch amd64 --out rsrc --file-version 0.0.0 --product-version 0.0.0
```

`winres.json` 里那个图标组必须叫 `"#3"`：Wails 用 `NewIconFromResource(GetModuleHandle(""), 3)` 给窗口设图标，而 go-winres 只给 `"#n"` 这种键编数字 ID。早先它叫 `"APP"`，任务栏和 Alt-Tab 显示的是 Windows 的空白页（上游 #1777）。`internal/cli/winres_test.go` 有这条守卫。

### 2) 构建

```powershell
$env:CGO_ENABLED='0'; $env:GOOS='windows'; $env:GOARCH='amd64'
$ver = (git describe --tags --always --dirty) -replace '[^A-Za-z0-9._+-]',''
if (-not $ver) { $ver = 'dev' }
go build -tags production -trimpath -ldflags="-s -w -X main.version=$ver -H windowsgui" -o magpie.exe .
$env:CGO_ENABLED=$null; $env:GOOS=$null; $env:GOARCH=$null
```

两个容易踩的坑：

- **`-H windowsgui` 必须写在 `-ldflags` 里面。** 写到外面会报 `flag provided but not defined: -H`。
- **`git describe` 的输出要先过滤。** 版本串里可能有 Go 链接器不认的字符，所以先 `-replace '[^A-Za-z0-9._+-]',''`，再塞进 `-ldflags`。

### 3) 删掉中间产物

```powershell
del rsrc_windows_amd64.syso
```

这个 `.syso` 是 go-winres 生成的中间文件，不该留在仓库里。

### 跑起来

- 跑着的 exe 被 Windows 锁着，**换不掉**。要更新就先退出 magpie（托盘图标右键 → 退出 magpie），再替换 `magpie.exe`。
- 网关在 `127.0.0.1:3425`。
- 想知道手上这个二进制自己的版本戳：

```powershell
(Invoke-RestMethod http://127.0.0.1:3425/api/hello).version
```

`/api/hello` 回的是 `{"name":"magpie","version":"…"}`，那个 `version` 就是这个二进制的版本戳。

> 别用 `/api/version` 查版本：那个路由是**给 Ollama 客户端的兼容层**，不管 magpie 是哪个版本，它都回 `"0.12.6"`（`internal/gateway/ollama.go`）。magpie 自己的版本在 `/api/hello`（`internal/gateway/gateway.go`）。

设置文件在 `~/.config/magpie`（Windows 上就是 `C:\Users\<你的用户名>\.config\magpie`）；便携版则放在 exe 旁边的 `data\` 目录里（放一个 `.portable` 文件在旁边也算便携）。

---

## 三、桌面快捷方式（Windows 上最需要手做的一步）

### 为什么需要手做

Windows 任务栏上的图标取的是**窗口自己的图标**（`WM_SETICON`），也就是第二节注入进 exe 的那个资源。给 `.lnk` 写 AppUserModelID 对任务栏**没有效果**——任务栏不会因为你的快捷方式换图标。

所以要让任务栏和桌面上的图标都好看，得同时做两件事：

1. 改 app 的图标资源（第二节第 1 步已经做了）；
2. 让桌面快捷方式指向一个 `.ico`，这样**桌面图标**和它下面的任务栏按钮才是绿圆。

### 怎么做

先把绿圆存成一个 `.ico`，比如放在 `%LOCALAPPDATA%\Magpie\magpie-green.ico`。然后用 PowerShell 建快捷方式：

```powershell
$sh = New-Object -ComObject WScript.Shell
$lnk = $sh.CreateShortcut("$env:USERPROFILE\Desktop\Magpie.lnk")
$lnk.TargetPath       = 'G:\zcode-project\magpie\magpie.exe'
$lnk.WorkingDirectory = 'G:\zcode-project\magpie'
$lnk.IconLocation     = "$env:LOCALAPPDATA\Magpie\magpie-green.ico,0"
$lnk.Save()
```

路径换成你自己的。

**托盘图标不用管。** 托盘里那颗是程序自己画的（第四节），跟快捷方式没关系。

---

## 四、系统托盘图标

### 改之前是什么样

托盘里那颗绿圆只有 **18×18 物理像素**，比同一排的应用小一圈。上游 issue [yetone/magpie#2404](https://github.com/yetone/magpie/issues/2404) 报的是同一类问题：报告者在 Windows 11 的通知区域逐像素量过，同排 7 个图标里只有 magpie 是 13×13（100% 缩放下），别的都填满了 16×16 的槽位。

### 为什么不是源图画布的错

圆盘自己在 64px 的图上就占了 43.2/44，也就是约 62.8px，接近 **62/64 = 96.9%**。留白不是问题所在。

### 真正的原因

Wails v3 的 `pkg/w32/icon.go` 里，`CreateSmallHIconFromImage` 对 **PNG 是原样传给 Windows 的**——不缩放，由 Windows 自己去把 64 缩到系统要的那一档（`SM_CXSMICON`）。**尺寸是在这一步丢掉的**：一张 64px 的图缩到 20px，图自带的边距也跟着缩，可见部分就只剩 18px。

### 解法：喂多尺寸 ICO

同一段代码对 ICO 会走 `selectICOImage`：它在「**不超过**系统要的那一档」里挑最近的一档，于是每一档都**原样用上、1:1 不缩放**。

本 fork 的 `internal/gui/tray-windows.ico` 带 16/20/24/28/32/36/40/48/64 九档，覆盖 100%~250% 的缩放：

| 缩放 | 100% | 125% | 150% | 175% | 200% | 225% | 250% |
| :-- | :-- | :-- | :-- | :-- | :-- | :-- | :-- |
| 系统要的尺寸 | 16 | 20 | 24 | 28 | 32 | 36 | 40 |

少一档，系统就会拿下一档去放大，那颗绿圆就会发虚。

### 实测结果

改完托盘里那颗绿圆是 **20×20 物理像素**，和同排（ChatGPT、终端、蓝盾）一样大。本机 2560×1440、125% 缩放，系统要的尺寸是 20。

### 守卫测试

`internal/gui/trayglyph_test.go` 里的 `TestTrayWindowsIconSizes` 守着两件事：

1. 九档（16/20/24/28/32/36/40/48/64）必须都在。缺一档测试就红，并说明「缺的那一档会被系统缩放，画出来是软的」。
2. 每档的圆盘必须**填满自己的画布**（至少到尺寸减一）。留白的画布会被画得比旁边的图标小。

所以想改品牌形状，或者把 `build/icon/gen.go` 里的 `trayWindowsFill`（现在是 `1.0`）调小，这个测试会先红给你看。

---

## 五、重新生成图标

`make icons` 会重新生成全部图标，包括 macOS 的 icns。但它最后两步 `iconutil` 和 `xcrun actool` **只有 macOS 有**，在 Windows 上会失败。

在 Windows 上单独重跑你需要的那几条：

```powershell
go run build/icon/gen.go tray internal/gui/tray.png
go run build/icon/gen.go tray-flap internal/gui/trayflap
go run build/icon/gen.go app 1024 internal/gui/icon-1024.png
go run build/icon/gen.go app 256 internal/gui/icon-256.png
go run build/icon/gen.go ico internal/gui/tray-windows.ico
foreach ($s in 16,32,48,64,128,256) { go run build/icon/gen.go app $s "build/windows/icon-$s.png" }
```

几点：

- 生成完**要重新构建 exe**。托盘图标是 `//go:embed` 进二进制的（`internal/gui/app.go`），不重编，图标不会变。
- **生成物要一起提交。** `internal/gui/tray-windows.ico` 就在仓库里，它是 exe 的一部分。
- `build/darwin/Assets.car` 只能在 Mac 上用 `xcrun actool` 重编，Windows 上动不了。

---

## 六、日常使用

功能和上游一致，这里只列最常用的：

- **点一下托盘图标** → 弹出紧凑面板：模型、路由、用量、允许项。
- **托盘图标右键** → 打开 magpie（主窗口）/ 版本 / 退出 magpie。
- **加 provider**：Providers → 添加 provider，选个预设贴 key，或者登录一个订阅。
- **给每个 Agent 选模型**：Agents 页，每行一个 Agent。
- **路由组**：让 Agent 把一组模型当作一个模型来选，额度用完自动切到下一个账号或模型。
- **方案（Profile）**：把所有 Agent 的配置存成一个名字（比如「省钱」「专注」），一键全部切回去。
- **看额度**：`magpie quota`。
- **全终端**：`magpie tui`。
- **菜单栏显示用量和花费**：设置 → 「菜单栏显示额度」选要显示的订阅，「菜单栏金额」控制要不要连花费一起显示。

---

## 七、常见问题 / 已知限制

### 托盘图标比同排应用小

本 fork 已经修好了（第四节）。如果你改了品牌源，或者把 `build/icon/gen.go` 里的 `trayWindowsFill` 调小，`TestTrayWindowsIconSizes` 会红。

### Windows 托盘右键菜单打不开

这是上游 Wails v3 的已知问题，不是这个 fork 引入的。

Wails v3 beta.24 的 `pkg/application/systemtray_windows.go` 的窗口过程只处理 `WM_RBUTTONUP`，但它的托盘图标用的是 `NOTIFYICON_VERSION_4`——在这个版本下右键是以 `WM_CONTEXTMENU` 送到的，于是右键处理器根本不触发，菜单打不开。

上游 PR [wailsapp/wails#6162](https://github.com/wailsapp/wails/pull/6162) 修了这个问题（2026-10-15 合入 master），**v3.0.0-beta.34 起包含这个修复**。本 fork 用的还是 beta.24，**没有为此升级 Wails**，所以这个现象还在。

### 改完图标没变化

- 托盘图标是编译进 exe 的，要**重新构建并重启** magpie。
- macOS 的 Dock 图标还要在 Mac 上重编 `Assets.car`。
- 桌面图标看的是快捷方式的 `IconLocation`，不是 exe 里的资源。

### GUI 回归测试在本机只能跑 chromium

本机没装 WebKit，所以只跑 Chromium；而且必须串行：

```powershell
$env:BROWSER='chromium'
node --test --test-concurrency=1 internal/gui/tests/logo-mark.test.cjs
```

`logo-mark.test.cjs` 读的是 `build/icon/magpie.svg` 和 `build/icon/gen.go` 里的品牌形状和两种绿色：改了源文件，它会说出哪儿不一样。

---

## 八、这个 fork 相对上游的三处改动

改动只有这三个 commit：

| commit | 说明 |
| :-- | :-- |
| `a9fd5201` | `gui: the app's mark is a green disc, not a magpie` —— 顶栏 logo、favicon、应用图标 |
| `a821f534` | `gui: the rest of magpie's marks are the app's disc too` —— 登录页标记、Agents 列表里 magpie 自己的图标、站点 logo、Omarchy 小组件图形 |
| `56fe6195` | `gui: the tray's icon is the size of the icons beside it` —— 托盘图标改成多尺寸 ICO |

`git log --oneline -3` 能看到它们。
