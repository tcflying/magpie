# magpie Windows 版使用教程（品牌黄圆版）

这份教程写给拿到这个 fork 的人：你想在 Windows 上把它从源码跑起来，把桌面和任务栏的图标换成那颗品牌黄圆，并且弄明白托盘里那颗黄圆为什么曾经比旁边的图标小一圈。

功能层面的说明不在这里，请看 [README.zh-CN.md](../README.zh-CN.md) 的「快速开始」和 [完整参考](reference.md)（英文）。

---

## 一、这是什么

这是 [yetone/magpie](https://github.com/yetone/magpie) 的一个 fork。**功能和上游完全一样**，只有视觉上的改动：上游在所有地方显示的是一只 magpie 鸟，这个 fork 把它换成了**品牌黄实心圆**。

- 亮色主题的品牌黄：`#FACC15`（rgb 250, 204, 21）
- 暗色主题的品牌黄：`#FDE047`（rgb 253, 224, 71）

这个 fork 最早做出来的是**绿圆**，后来只是把同一个圆盘换成黄色——一次普通的后续修改。绿圆那一版还留在仓库的历史里，`git log --oneline` 往下翻能看到。

换过的地方：

| 地方 | 在哪儿 |
| :-- | :-- |
| 任务栏图标、窗口 / 资源管理器图标 | 注入进 exe 的 PE 资源（`build/windows/winres.json`） |
| 系统托盘图标 | 单独一个多尺寸 ICO（`internal/gui/tray-windows.ico`），见第四节 |
| 应用内顶栏 logo | `internal/gui/assets/index.html` 里内联的 SVG |
| 浏览器 favicon | `internal/gui/assets/icons/favicon.svg` |
| 登录页的 magpie 标记 | `internal/provider/magpie.svg`（供应商登录结束时浏览器落到的那个页面） |
| Agents 列表里 magpie 自己的图标 | `internal/gui/assets/icons/magpie.svg`（当作遮罩用，`app.css` 给了它品牌黄） |
| 站点 logo | `site/public/img/magpie.svg`、`site/public/img/magpie-small.svg` |
| Omarchy 桌面小组件的图形 | `internal/omarchy/widget/Widget.qml` |

**形状的权威源只有一处**，要改品牌形状就改它：

- `build/icon/magpie.svg` —— 圆本身。44 格的 viewBox，圆心在 (22, 22)，半径 21.6：路径从 0.40 走到 43.60。
- `build/icon/gen.go` —— 画法（怎么把 SVG 变成 PNG 和 ICO）和两处颜色：`0xFACC15` 与暗色的 `0xFDE047`。

同一组颜色值还写在 `internal/gui/assets/app.css` 的 `--logo-fill`、`internal/gui/assets/icons/favicon.svg`、`internal/provider/magpie.svg` 和 `internal/provider/signin.go` 里。改颜色时这几处要一起改。

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

在**仓库根目录**里跑（先 `cd` 到仓库根）。产物直接写到 `ops/yellow-20261023/artifacts/`，目录没有就建一个：

```powershell
$out = 'ops\yellow-20261023\artifacts'
New-Item -ItemType Directory -Force -Path $out | Out-Null

$envs = 'CGO_ENABLED','GOOS','GOARCH'
$old = @{}
foreach ($n in $envs) { $old[$n] = [Environment]::GetEnvironmentVariable($n, 'Process') }
try {
  $env:CGO_ENABLED = '0'; $env:GOOS = 'windows'; $env:GOARCH = 'amd64'
  $ver = (git describe --tags --always --dirty --long) -replace '[^A-Za-z0-9._+-]',''
  if (-not $ver) { $ver = 'dev' }
  $ver = "$ver+yellow"
  go build -tags production -trimpath -ldflags="-s -w -X main.version=$ver -H windowsgui" -o "$out\magpie-yellow.exe" .
} finally {
  foreach ($n in $envs) {
    if ($null -eq $old[$n]) { Remove-Item "env:$n" -ErrorAction SilentlyContinue }
    else { Set-Item "env:$n" $old[$n] }
  }
}
```

几点：

- **产物是 `ops\yellow-20261023\artifacts\magpie-yellow.exe`**（官方上游那份叫 `magpie.exe`，别混了）。`ops/` 整个目录被 `.gitignore` 挡着，不会进 Git。
- **三个环境变量只是临时改一下**，`finally` 里照原样还原：原来没有的还删掉，不动你 shell 里原本的设置。

Go 自己的缓存和临时文件默认落在 C 盘（`%LOCALAPPDATA%\go-build`、`%USERPROFILE%\go\pkg\mod`）。要把开发环境整体放 D 盘，就在**构建前**先指到 D 盘（路径换成你自己的）：

```powershell
$env:GOCACHE='D:\go-cache\build'; $env:GOMODCACHE='D:\go-cache\mod'; $env:GOTMPDIR='D:\go-cache\tmp'
New-Item -ItemType Directory -Force -Path D:\go-cache\build,D:\go-cache\mod,D:\go-cache\tmp | Out-Null
```

构建只写 `ops/` 和 Go 自己的缓存，不碰用户数据：设置、账号、Provider 都在 `%USERPROFILE%\.config\magpie`（便携版是 exe 旁边的 `data\`），构建过程不读也不改它们。

**版本串是防自动更新的关键**，别改：

- `--long` 让 `git describe` 一定带上「离上一个 tag 几个提交 + 提交号」，哪怕 HEAD 正好落在 tag 上，也是 `v0.1.1327-0-g6d812fc3` 这种形式。少了 `--long`，在 tag 上构建只会得到 `v0.1.1327`，程序就会当成正式发布版。
- 程序判断「是不是正式发布版」看的就是这一段：`internal/update/update.go` 的 `Released()`。版本里带 `-<数字>-g<提交号>`（或者 `-dirty`），它就认定这不是发布版，官方更新不会来替换它。
- `+yellow` 只是给这个构建打标记。`+` 后面是构建元信息，程序读版本时不算它（`parse()` 会先把它切掉）。

所以版本串长这样：`v0.1.1327-5-g6d812fc3+yellow`；工作区有改动时是 `v0.1.1327-5-g6d812fc3-dirty+yellow`。

两个容易踩的坑：

- **`-H windowsgui` 必须写在 `-ldflags` 里面。** 写到外面会报 `flag provided but not defined: -H`。
- **`git describe` 的输出要先过滤。** 版本串里可能有 Go 链接器不认的字符，所以先 `-replace '[^A-Za-z0-9._+-]',''`，再塞进 `-ldflags`（`+` 在白名单里，别被滤掉）。

### 3) 删掉中间产物

```powershell
del rsrc_windows_amd64.syso
```

这个 `.syso` 是 go-winres 生成的中间文件，不该留在仓库里。

### 跑起来

- 跑着的 exe 被 Windows 锁着，**换不掉**。要更新就先退出 magpie（托盘图标右键 → 退出 magpie），再替换 `magpie-yellow.exe`。
- 网关在 `127.0.0.1:3425`。
- 想知道手上这个二进制自己的版本戳：

```powershell
(Invoke-RestMethod http://127.0.0.1:3425/api/hello).version
```

`/api/hello` 回的是 `{"name":"magpie","version":"…"}`，那个 `version` 就是这个二进制的版本戳。

> 别用 `/api/version` 查版本：那个路由是**给 Ollama 客户端的兼容层**，不管 magpie 是哪个版本，它都回 `"0.12.6"`（`internal/gateway/ollama.go`）。magpie 自己的版本在 `/api/hello`（`internal/gateway/gateway.go`）。

设置文件在 `~/.config/magpie`（Windows 上就是 `C:\Users\<你的用户名>\.config\magpie`）；便携版则放在 exe 旁边的 `data\` 目录里（放一个 `.portable` 文件在旁边也算便携）。

### 想要在终端里敲命令？再编一个控制台版

上面第二步的 `-H windowsgui` 会让 exe 变成 **GUI subsystem** 的程序：双击没有黑框，这是对的，
但代价是它在终端里**一句输出都没有**。

```powershell
D:\path\to\magpie-yellow.exe --version     # 什么都没有，退出码却是 0
D:\path\to\magpie-yellow.exe ls            # 同样什么都没有
```

不是程序挂了，是 Windows GUI subsystem 的进程默认**不继承标准句柄**（`GetStdHandle` 返回
NULL），magpie 里也没有 `AttachConsole` 那段代码，所以 `fmt.Println` 全部写不进去。
第六节说的 `magpie quota`、`magpie tui` 这些命令，在这个构建上是用不了的。

所以再编一个**控制台版**，它不带 GUI 代码，体积也小：

```powershell
$env:CGO_ENABLED='0'; $env:GOOS='windows'; $env:GOARCH='amd64'
go build -tags nogui -trimpath -ldflags="-s -w -X main.version=$ver" -o "$out\magpie-yellow-cli.exe" .
```

于是：

| 文件 | 是什么 | 怎么用 |
| :-- | :-- | :-- |
| `magpie-yellow.exe` | 桌面应用（GUI subsystem） | 桌面快捷方式、托盘、窗口 |
| `magpie-yellow-cli.exe` | 控制台 CLI（console subsystem） | 终端里敲 `magpie-yellow-cli.exe ls` / `quota` / `tui` |

注意两件事：

- 两个 exe 的版本戳要一致（`-X main.version=$ver` 用同一个 `$ver`），这样
  `magpie-yellow-cli.exe --version` 和 `/api/hello` 回的版本才对得上。
- `-tags nogui` 会把 GUI 代码整个剔掉，所以它不占端口、不上托盘，只是个命令行工具。
  两个 exe 别同时跑着抢 3425 端口。

---

## 三、桌面快捷方式（Windows 上最需要手做的一步）

### 为什么需要手做

Windows 任务栏上的图标取的是**窗口自己的图标**（`WM_SETICON`），也就是第二节注入进 exe 的那个资源。给 `.lnk` 写 AppUserModelID 对任务栏**没有效果**——任务栏不会因为你的快捷方式换图标。

所以要让任务栏和桌面上的图标都好看，得同时做两件事：

1. 改 app 的图标资源（第二节第 1 步已经做了）；
2. 让桌面快捷方式指向一个 `.ico`，这样**桌面图标**和它下面的任务栏按钮才是黄圆。

### 怎么做

先把黄圆存成一个 `.ico`，比如放在 `%LOCALAPPDATA%\Magpie\magpie-yellow.ico`。然后用 PowerShell 建快捷方式：

```powershell
$sh = New-Object -ComObject WScript.Shell
$lnk = $sh.CreateShortcut("$env:USERPROFILE\Desktop\Magpie.lnk")
$lnk.TargetPath       = 'D:\mmx-project\magpie\ops\yellow-20261023\artifacts\magpie-yellow.exe'
$lnk.WorkingDirectory = 'D:\mmx-project\magpie\ops\yellow-20261023\artifacts'
$lnk.IconLocation     = "$env:LOCALAPPDATA\Magpie\magpie-yellow.ico,0"
$lnk.Save()
```

路径换成你自己的。快捷方式要指向本 fork 那个 `magpie-yellow.exe`，别指向官方的 `magpie.exe`。

**托盘图标不用管。** 托盘里那颗是程序自己画的（第四节），跟快捷方式没关系。

---

## 四、系统托盘图标

### 改之前是什么样

托盘里那颗黄圆只有 **18×18 物理像素**，比同一排的应用小一圈。上游 issue [yetone/magpie#2404](https://github.com/yetone/magpie/issues/2404) 报的是同一类问题：报告者在 Windows 11 的通知区域逐像素量过，同排 7 个图标里只有 magpie 是 13×13（100% 缩放下），别的都填满了 16×16 的槽位。

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

少一档，系统就会拿下一档去放大，那颗黄圆就会发虚。

### 实测结果

改完托盘里那颗黄圆是 **20×20 物理像素**，和同排（ChatGPT、终端、蓝盾）一样大。本机 2560×1440、125% 缩放，系统要的尺寸是 20。

### 守卫测试

`internal/gui/trayglyph_test.go` 里的 `TestTrayWindowsIconSizes` 守着两件事：

1. 九档（16/20/24/28/32/36/40/48/64）必须都在。缺一档测试就红，并说明「缺的那一档会被系统缩放，画出来是软的」。
2. 每档的圆盘必须**填满自己的画布**（至少到尺寸减一）。留白的画布会被画得比旁边的图标小。

所以想改品牌形状，或者把 `build/icon/gen.go` 里的 `trayWindowsFill`（现在是 `1.0`）调小，这个测试会先红给你看。

---

## 五、重新生成图标

上游的 `make icons` 会重新生成全部图标，包括 macOS 的 icns。但它最后两步 `iconutil` 和 `xcrun actool` **只有 macOS 有**，在 Windows 上会失败。

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
- **看额度**：`magpie-yellow-cli.exe quota`（控制台版才有输出，见第二节末尾那节）。
- **全终端**：`magpie-yellow-cli.exe tui`（同上）。
- **菜单栏显示用量和花费**：设置 → 「菜单栏显示额度」选要显示的订阅，「菜单栏金额」控制要不要连花费一起显示。

---

## 七、常见问题 / 已知限制

### 官方版会把黄圆换回鸟

黄版是从这个 fork 的源码构建的，版本串是 `git describe --long` 的结果加 `+yellow`。程序据此认定它不是正式发布版（`internal/update/update.go` 的 `Released()`），官方更新不会来替换它。GUI 里没有安装官方更新的按钮，这是有意这样。

更新黄版的办法就是从这个 fork 的源码重新构建，再替换掉旧 exe。

官方 release 是从上游构建的，里面是那只 magpie 鸟，**不含这个定制黄圆**。所以别手动装官方版去覆盖它，覆盖了黄色就没了。

### 托盘图标比同排应用小

本 fork 已经修好了（第四节）。如果你改了品牌源，或者把 `build/icon/gen.go` 里的 `trayWindowsFill` 调小，`TestTrayWindowsIconSizes` 会红。

### Windows 托盘右键菜单打不开（已修）

这是上游 Wails v3 的问题，不是这个 fork 引入的。**本 fork 已通过升级 Wails 修掉。**

原因：Wails v3 beta.24 的 `pkg/application/systemtray_windows.go` 的窗口过程只处理 `WM_RBUTTONUP`，
但它的托盘图标用的是 `NOTIFYICON_VERSION_4`——在这个版本下右键是以 `WM_CONTEXTMENU` 送到的，
于是右键处理器根本不触发，菜单打不开。

上游 PR [wailsapp/wails#6162](https://github.com/wailsapp/wails/pull/6162) 修了这个问题
（2026-10-15 合入 master），**v3.0.0-beta.34 起包含这个修复**。本 fork 的 `go.mod` 现在是
`github.com/wailsapp/wails/v3 v3.0.0-beta.34`，所以托盘图标右键能出菜单了
（打开 magpie / 版本 / 退出 magpie 三项）。

升级 Wails 跨了十个 beta，两个构建（`production` 和 `nogui`）都实测能过。

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

`logo-mark.test.cjs` 读的是 `build/icon/magpie.svg` 和 `build/icon/gen.go` 里的品牌形状和两种颜色：改了源文件，它会说出哪儿不一样。

---

## 八、这个 fork 相对上游改了哪些地方

改动都是视觉和文档上的，功能跟上游一样：

| 改动 | 说明 |
| :-- | :-- |
| 标记换成纯色圆盘 | 顶栏 logo、favicon、应用图标、登录页标记、Agents 列表里 magpie 自己的图标、站点 logo、Omarchy 小组件图形 |
| 圆盘的颜色 | 品牌黄 `#FACC15`（暗色 `#FDE047`）。这个 fork 最早做的是绿圆，改成黄圆是之后的普通后续修改，绿圆那一版还在历史里 |
| Windows 托盘图标 | 改成多尺寸 ICO，九档尺寸，和旁边的图标一样大（第四节） |
| Wails 版本 | 从上游的 `v3.0.0-beta.24` 升到 `v3.0.0-beta.34`，顺带修好了 Windows 托盘右键菜单（第七节） |
| 文档 | 这份教程，和 README 顶部的 fork 说明 |

这个 fork 的历史在 2026-10-24 重建在上游 commit `84341795` 之上（当天上游 main 的位置）。想看它相对上游到底改了什么，在自己的 clone 里跑：

```powershell
git log --oneline 84341795..HEAD
git diff --stat 84341795..HEAD
```

上游自己会往前走，所以这里不写「领先几个、落后几个」这种会过期的数字。
