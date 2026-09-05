import { CODEX_BACKUP_DOWNLOAD_URL, CODEX_OFFICIAL_DOWNLOAD_URL } from '@/constants/codexDownload'

// JSON and TOML basic strings share these escapes; TOML also forbids raw DEL.
export function escapeTomlBasicString(value: string): string {
  return JSON.stringify(value).slice(1, -1).replace(/\u007f/g, '\\u007f')
}

const shellString = (value: string) => `'${value.replace(/'/g, "'\\''")}'`
const powerShellString = (value: string) => `'${value.replace(/'/g, "''")}'`

export function generateMacCodexCommand(configContent: string, authContent: string): string {
  const script = `set -Eeuo pipefail
umask 077
CODEX_DIR="$HOME/.codex"
CONFIG_FILE="$CODEX_DIR/config.toml"
AUTH_FILE="$CODEX_DIR/auth.json"
TEMP_CONFIG_FILE=""
TEMP_AUTH_FILE=""
BACKUP_CONFIG_FILE=""
BACKUP_AUTH_FILE=""
REPLACE_STARTED=0
COMMITTED=0
STEP="检查配置目录"
TARGET="$CODEX_DIR"

fail() { printf '\\n配置失败：%s\\n目标：%s\\n%s\\n' "$STEP" "$TARGET" "$1" >&2; exit 1; }

restore_file() {
  if [ -n "$2" ]; then cp -p "$2" "$1"; else rm -f "$1"; fi
}

cleanup() {
  local status=$?
  trap - EXIT ERR
  set +e
  if [ "$status" -ne 0 ]; then
    if [ "$REPLACE_STARTED" -eq 1 ] && [ "$COMMITTED" -eq 0 ]; then
      local restore_failed=0
      restore_file "$CONFIG_FILE" "$BACKUP_CONFIG_FILE" || restore_failed=1
      restore_file "$AUTH_FILE" "$BACKUP_AUTH_FILE" || restore_failed=1
      if [ "$restore_failed" -eq 0 ]; then
        echo "已恢复配置前的两个文件状态。" >&2
      else
        printf '恢复失败，请保留备份并手动恢复：\\n%s\\n%s\\n' "$BACKUP_CONFIG_FILE" "$BACKUP_AUTH_FILE" >&2
      fi
    else
      [ -z "$BACKUP_CONFIG_FILE" ] || rm -f "$BACKUP_CONFIG_FILE"
      [ -z "$BACKUP_AUTH_FILE" ] || rm -f "$BACKUP_AUTH_FILE"
      echo "旧配置未被替换。" >&2
    fi
  fi
  [ -z "$TEMP_CONFIG_FILE" ] || rm -f "$TEMP_CONFIG_FILE"
  [ -z "$TEMP_AUTH_FILE" ] || rm -f "$TEMP_AUTH_FILE"
}
trap cleanup EXIT
trap 'fail "系统命令失败（退出码 $?），请查看上方具体错误。"' ERR

if [ ! -d "$CODEX_DIR" ] || [ -L "$CODEX_DIR" ]; then
  fail "未找到 Codex 配置目录，无法确认 Codex 已安装并完成初始化。
请先安装 Codex，启动一次并退出，然后重新执行配置。
官方下载：${CODEX_OFFICIAL_DOWNLOAD_URL}
备用网盘：${CODEX_BACKUP_DOWNLOAD_URL}"
fi
STEP="检查 Codex 进程"
if pgrep -if '(^|/)Codex( |$)' >/dev/null; then
  fail "检测到 Codex App 仍在运行。请彻底退出 Codex 后重新执行。"
else
  status=$?
  [ "$status" -eq 1 ] || fail "无法检查 Codex 进程（退出码 $status）。"
fi
for TARGET in "$CONFIG_FILE" "$AUTH_FILE"; do
  if { [ -e "$TARGET" ] && [ ! -f "$TARGET" ]; } || [ -L "$TARGET" ]; then
    fail "目标不是普通配置文件，请检查该路径。"
  fi
done

STEP="写入临时配置"
TARGET="$CODEX_DIR"
TEMP_CONFIG_FILE="$(mktemp "$CODEX_DIR/.sub2api-config.XXXXXX")"
TEMP_AUTH_FILE="$(mktemp "$CODEX_DIR/.sub2api-auth.XXXXXX")"
printf '%s' ${shellString(configContent)} > "$TEMP_CONFIG_FILE"
printf '%s' ${shellString(authContent)} > "$TEMP_AUTH_FILE"

STEP="校验临时配置内容"
TARGET="$TEMP_CONFIG_FILE"
cmp -s "$TEMP_CONFIG_FILE" <(printf '%s' ${shellString(configContent)}) || fail "config.toml 内容校验失败。"
TARGET="$TEMP_AUTH_FILE"
cmp -s "$TEMP_AUTH_FILE" <(printf '%s' ${shellString(authContent)}) || fail "auth.json 内容校验失败。"
osascript -l JavaScript -e 'ObjC.import("Foundation"); var data=$.NSFileHandle.fileHandleWithStandardInput.readDataToEndOfFile; var text=$.NSString.alloc.initWithDataEncoding(data,$.NSUTF8StringEncoding).js; var value=JSON.parse(text); var keys=Object.keys(value); if (keys.length !== 1 || keys[0] !== "OPENAI_API_KEY" || typeof value.OPENAI_API_KEY !== "string" || value.OPENAI_API_KEY.length === 0) throw new Error("Invalid auth.json");' < "$TEMP_AUTH_FILE" >/dev/null 2>&1 || fail "auth.json JSON 解析校验失败。"

STEP="备份旧配置"
TARGET="$CONFIG_FILE"
if [ -f "$CONFIG_FILE" ]; then
  BACKUP_CONFIG_FILE="$(mktemp "$CONFIG_FILE.sub2api.bak-XXXXXX")"
  if ! cp "$CONFIG_FILE" "$BACKUP_CONFIG_FILE"; then
    rm -f "$BACKUP_CONFIG_FILE"
    BACKUP_CONFIG_FILE=""
    fail "config.toml 备份失败。"
  fi
fi
TARGET="$AUTH_FILE"
if [ -f "$AUTH_FILE" ]; then
  BACKUP_AUTH_FILE="$(mktemp "$AUTH_FILE.sub2api.bak-XXXXXX")"
  if ! cp "$AUTH_FILE" "$BACKUP_AUTH_FILE"; then
    rm -f "$BACKUP_AUTH_FILE"
    BACKUP_AUTH_FILE=""
    fail "auth.json 备份失败。"
  fi
fi

STEP="替换配置文件"
REPLACE_STARTED=1
TARGET="$CONFIG_FILE"
mv -f "$TEMP_CONFIG_FILE" "$CONFIG_FILE"
TEMP_CONFIG_FILE=""
TARGET="$AUTH_FILE"
mv -f "$TEMP_AUTH_FILE" "$AUTH_FILE"
TEMP_AUTH_FILE=""
COMMITTED=1
printf '\\nCodex 配置完成。\\n已写入：\\n%s\\n%s\\n' "$CONFIG_FILE" "$AUTH_FILE"
if [ -n "$BACKUP_CONFIG_FILE$BACKUP_AUTH_FILE" ]; then
  printf '旧配置备份：\\n%s\\n%s\\n' "$BACKUP_CONFIG_FILE" "$BACKUP_AUTH_FILE"
fi
echo "现在可以打开 Codex。若仍有问题，请联系网页右上角客服。"
open "$CODEX_DIR" >/dev/null 2>&1 || true`

  // Avoid a payload line terminating the outer heredoc in the user's shell.
  let delimiter = 'SUB2API_CODEX_SETUP'
  while (script.split('\n').includes(delimiter)) delimiter += '_'
  return `if /bin/bash <<'${delimiter}'
${script}
${delimiter}
then
  :
else
  echo "Codex 配置命令已失败，当前终端仍保持打开，请根据上方错误处理后重试。"
fi
`
}

export function generateWindowsCodexCommand(configContent: string, authContent: string): string {
  return `& {
  $ErrorActionPreference = 'Stop'
  Set-StrictMode -Version Latest
  $codexDir = Join-Path $env:USERPROFILE '.codex'
  $configFile = Join-Path $codexDir 'config.toml'
  $authFile = Join-Path $codexDir 'auth.json'
  $tempConfigFile = $null
  $tempAuthFile = $null
  $backupConfigFile = $null
  $backupAuthFile = $null
  $replaceStarted = $false
  $committed = $false
  $step = '检查配置目录'
  $target = $codexDir
  function Replace-CodexFile([string]$TempPath, [string]$TargetPath) {
    if ([System.IO.File]::Exists($TargetPath)) {
      [System.IO.File]::Replace($TempPath, $TargetPath, [NullString]::Value)
    } else {
      [System.IO.File]::Move($TempPath, $TargetPath)
    }
  }
  try {
    if (!(Test-Path -LiteralPath $codexDir -PathType Container)) {
      throw "未找到 Codex 配置目录，无法确认 Codex 已安装并完成初始化。
请先安装 Codex，启动一次并退出，然后重新执行配置。
官方下载：${CODEX_OFFICIAL_DOWNLOAD_URL}
备用网盘：${CODEX_BACKUP_DOWNLOAD_URL}"
    }
    $step = '检查 Codex 进程'
    if (Get-Process -Name Codex -ErrorAction SilentlyContinue) {
      throw '检测到 Codex App 仍在运行。请彻底退出 Codex 后重新执行。'
    }
    foreach ($target in @($configFile, $authFile)) {
      if (Test-Path -LiteralPath $target) {
        $item = Get-Item -LiteralPath $target -Force
        if ($item.PSIsContainer -or ($item.Attributes -band [System.IO.FileAttributes]::ReparsePoint)) {
          throw '目标不是普通配置文件，请检查该路径。'
        }
      }
    }
    $step = '写入临时配置'
    $target = $codexDir
    $id = [Guid]::NewGuid().ToString('N')
    $tempConfigFile = Join-Path $codexDir ('.sub2api-config-' + $id + '.tmp')
    $tempAuthFile = Join-Path $codexDir ('.sub2api-auth-' + $id + '.tmp')
    $configContent = ${powerShellString(configContent)}
    $authContent = ${powerShellString(authContent)}
    $utf8 = [System.Text.UTF8Encoding]::new($false)
    [System.IO.File]::WriteAllText($tempConfigFile, $configContent, $utf8)
    [System.IO.File]::WriteAllText($tempAuthFile, $authContent, $utf8)

    $step = '校验临时配置内容'
    $target = $tempConfigFile
    if ([System.IO.File]::ReadAllText($tempConfigFile, $utf8) -cne $configContent) { throw 'config.toml 内容校验失败。' }
    $target = $tempAuthFile
    if ([System.IO.File]::ReadAllText($tempAuthFile, $utf8) -cne $authContent) { throw 'auth.json 内容校验失败。' }
    try { $json = [System.IO.File]::ReadAllText($tempAuthFile, $utf8) | ConvertFrom-Json }
    catch { throw 'auth.json JSON 解析校验失败。' }
    if ($null -eq $json) { throw 'auth.json 解析校验失败。' }
    $properties = @($json.PSObject.Properties)
    if ($properties.Count -ne 1 -or $properties[0].Name -cne 'OPENAI_API_KEY' -or
        $properties[0].Value -isnot [string] -or [string]::IsNullOrEmpty($properties[0].Value)) {
      throw 'auth.json 解析校验失败。'
    }

    $step = '备份旧配置'
    $target = $configFile
    if ([System.IO.File]::Exists($configFile)) {
      $backupConfigFile = "$configFile.sub2api.bak-$id"
      [System.IO.File]::Copy($configFile, $backupConfigFile, $false)
    }
    $target = $authFile
    if ([System.IO.File]::Exists($authFile)) {
      $backupAuthFile = "$authFile.sub2api.bak-$id"
      [System.IO.File]::Copy($authFile, $backupAuthFile, $false)
    }
    $step = '替换配置文件'
    $replaceStarted = $true
    $target = $configFile
    Replace-CodexFile $tempConfigFile $configFile
    $tempConfigFile = $null
    $target = $authFile
    Replace-CodexFile $tempAuthFile $authFile
    $tempAuthFile = $null
    $committed = $true
    Write-Host 'Codex 配置完成。' -ForegroundColor Green
    Write-Host "已写入：$configFile"
    Write-Host "已写入：$authFile"
    if ($backupConfigFile -or $backupAuthFile) {
      Write-Host "旧配置备份：$backupConfigFile"
      Write-Host "旧配置备份：$backupAuthFile"
    }
    Write-Host '现在可以打开 Codex。若仍有问题，请联系网页右上角客服。'
    try { Invoke-Item -LiteralPath $codexDir -ErrorAction Stop } catch { Write-Host '配置已完成，但无法自动打开文件夹。' }
  } catch {
    Write-Host "配置失败：$step" -ForegroundColor Red
    Write-Host "目标：$target" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    if ($replaceStarted -and !$committed) {
      $restoreFailed = $false
      foreach ($entry in @(@($configFile, $backupConfigFile), @($authFile, $backupAuthFile))) {
        try {
          if ($entry[1]) { [System.IO.File]::Copy($entry[1], $entry[0], $true) }
          else { [System.IO.File]::Delete($entry[0]) }
        } catch {
          $restoreFailed = $true
          Write-Host "恢复失败：$($entry[0])" -ForegroundColor Red
          Write-Host $_.Exception.Message -ForegroundColor Red
          Write-Host "请保留备份：$($entry[1])" -ForegroundColor Red
        }
      }
      if (!$restoreFailed) { Write-Host '已恢复配置前的两个文件状态。' }
    } else {
      foreach ($path in @($backupConfigFile, $backupAuthFile)) {
        if ($path) { Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue }
      }
      Write-Host '旧配置未被替换。'
    }
  } finally {
    foreach ($path in @($tempConfigFile, $tempAuthFile)) {
      if ($path) { Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue }
    }
  }
}
`
}
